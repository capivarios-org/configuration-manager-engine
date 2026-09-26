package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/capivarios-org/configuration-manager-engine/internal/cache"
	"github.com/capivarios-org/configuration-manager-engine/internal/domain"
	"github.com/capivarios-org/configuration-manager-engine/internal/sse"
)

// Server encapsula os handlers e dependências da API do Data Plane
type Server struct {
	cache  *cache.EngineCache
	sseHub *sse.Hub
	mux    *http.ServeMux
}

// NewServer inicializa o roteador HTTP do Data Plane
func NewServer(c *cache.EngineCache, hub *sse.Hub) *Server {
	s := &Server{
		cache:  c,
		sseHub: hub,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// Healthcheck
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Smart Polling Endpoints (ETag / 304 Not Modified)
	s.mux.HandleFunc("GET /api/v1/flags/{key}", s.handleGetFlag)
	s.mux.HandleFunc("GET /api/v1/folders/{key}", s.handleGetFolder)

	// Server-Sent Events Endpoint
	s.mux.HandleFunc("GET /api/v1/stream", s.handleStream)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "healthy",
		"service": "configuration-manager-data-plane",
	})
}

// handleGetFlag entrega o valor da flag em sub-millisecond com suporte a ETag/304
func (s *Server) handleGetFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, `{"error":"flag key is required"}`, http.StatusBadRequest)
		return
	}

	// Enforcement de governança
	if policy, ok := s.cache.GetPolicy(key); ok {
		if policy.TransportPolicy == domain.TransportSSEOnly {
			http.Error(w, `{"error":"transport policy violation: SSE_ONLY required"}`, http.StatusForbidden)
			return
		}
		if policy.ConsumePolicy == domain.ConsumeFolderOnly {
			http.Error(w, `{"error":"consume policy violation: FOLDER_ONLY required"}`, http.StatusForbidden)
			return
		}
	}

	entry, found := s.cache.GetFlag(key)
	if !found {
		http.Error(w, `{"error":"flag not found"}`, http.StatusNotFound)
		return
	}

	// Smart Polling: Verificação de If-None-Match para retorno 304 Not Modified imediato
	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Trim(match, "W/") == strings.Trim(entry.ETag, "W/") {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", entry.ETag)
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(entry.Value)
}

// handleGetFolder entrega o agrupamento completo do folder em sub-millisecond com suporte a ETag/304
func (s *Server) handleGetFolder(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, `{"error":"folder key is required"}`, http.StatusBadRequest)
		return
	}

	// Enforcement de governança
	if policy, ok := s.cache.GetPolicy(key); ok {
		if policy.TransportPolicy == domain.TransportSSEOnly {
			http.Error(w, `{"error":"transport policy violation: SSE_ONLY required"}`, http.StatusForbidden)
			return
		}
		if policy.ConsumePolicy == domain.ConsumeFlagOnly {
			http.Error(w, `{"error":"consume policy violation: FLAG_ONLY required"}`, http.StatusForbidden)
			return
		}
	}

	entry, found := s.cache.GetFolder(key)
	if !found {
		http.Error(w, `{"error":"folder not found"}`, http.StatusNotFound)
		return
	}

	// Smart Polling: Verificação de If-None-Match para retorno 304 Not Modified imediato
	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Trim(match, "W/") == strings.Trim(entry.ETag, "W/") {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", entry.ETag)
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(entry.Value)
}

// handleStream conecta o cliente ao stream de Server-Sent Events
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	s.sseHub.ServeHTTP(w, r)
}
