package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Message representa um evento enviado via SSE
type Message struct {
	Event string
	Data  []byte
}

// Client representa uma conexão ativa de um SDK cliente
type Client struct {
	id      string
	channel chan Message
}

// Hub gerencia as conexões ativas de SSE e broadcasting de eventos
type Hub struct {
	clients map[*Client]bool
	mu      sync.RWMutex
}

// NewHub inicializa um novo Hub para Server-Sent Events
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
	}
}

// Register adiciona um novo cliente ao Hub
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = true
}

// Unregister remove um cliente desconectado
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.channel)
	}
}

// Broadcast envia uma mensagem para todos os clientes conectados
func (h *Hub) Broadcast(eventName string, payload any) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msg := Message{
		Event: eventName,
		Data:  bytes,
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.channel <- msg:
		default:
			// Canal cheio, evita travar o broadcaster
		}
	}
	return nil
}

// ServeHTTP gerencia a conexão HTTP persistente de streaming (SSE)
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	client := &Client{
		id:      r.RemoteAddr,
		channel: make(chan Message, 100),
	}

	h.Register(client)
	defer h.Unregister(client)

	// Heartbeat para manter a conexão aberta em proxies e load balancers
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Envia evento inicial de conexão estabelecida
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"ok\",\"timestamp\":%d}\n\n", time.Now().Unix())
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return

		case msg := <-client.channel:
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Event, string(msg.Data))
			flusher.Flush()

		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat %d\n\n", time.Now().Unix())
			flusher.Flush()
		}
	}
}
