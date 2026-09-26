package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/capivarios-org/configuration-manager-engine/internal/cache"
	"github.com/capivarios-org/configuration-manager-engine/internal/config"
	"github.com/capivarios-org/configuration-manager-engine/internal/sse"
	"github.com/capivarios-org/configuration-manager-engine/internal/storage"
	transporthttp "github.com/capivarios-org/configuration-manager-engine/internal/transport/http"
)

func main() {
	cfg := config.LoadConfig()
	log.Printf("[Data Plane] Iniciando Configuration Manager Engine (Port: %d, Driver: %s)...", cfg.Port, cfg.DBDriver)

	engineCache := cache.NewEngineCache()
	sseHub := sse.NewHub()

	// Tentativa de conexão com o banco de dados compartilhado
	dbClient, err := storage.NewDBClient(cfg)
	if err != nil {
		log.Printf("[Data Plane] [Aviso] Falha ao inicializar cliente do banco (%s): %v", cfg.DBDriver, err)
	} else {
		defer dbClient.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := dbClient.Ping(ctx); err != nil {
			log.Printf("[Data Plane] [Aviso] Banco de dados inacessível no momento: %v (seguindo em modo cache-only)", err)
		} else {
			log.Printf("[Data Plane] Conectado com sucesso ao banco %s (%s:%d/%s)", cfg.DBDriver, cfg.DBHost, cfg.DBPort, cfg.DBName)
			syncFromDB(context.Background(), dbClient, engineCache, sseHub)
		}
		cancel()

		// Background Sync Loop
		go startSyncLoop(dbClient, engineCache, sseHub, cfg.SyncInterval)
	}

	router := transporthttp.NewServer(engineCache, sseHub)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // Necessário 0 para conexões persistentes de SSE
		IdleTimeout:  60 * time.Second,
	}

	// Canal para shutdown gracioso
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[Data Plane] Servidor HTTP escutando em http://0.0.0.0:%d", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Data Plane] Erro no servidor HTTP: %v", err)
		}
	}()

	<-stop
	log.Println("[Data Plane] Encerrando serviço graciosamente...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Data Plane] Erro durante o shutdown do HTTP: %v", err)
	}

	log.Println("[Data Plane] Finalizado com sucesso.")
}

// syncFromDB sincroniza flags e folders do banco compartilhado para o cache RAM e notifica clientes via SSE
func syncFromDB(ctx context.Context, client *storage.DBClient, c *cache.EngineCache, hub *sse.Hub) {
	flags, err := client.FetchAllFlags(ctx)
	if err == nil {
		for _, f := range flags {
			entry, err := c.SetFlag(f)
			if err == nil {
				_ = hub.Broadcast("flag_updated", map[string]any{
					"key":     f.Key,
					"etag":    entry.ETag,
					"version": entry.Version,
				})
			}
		}
	}

	folders, err := client.FetchAllFolders(ctx)
	if err == nil {
		for _, folder := range folders {
			entry, err := c.SetFolder(folder)
			if err == nil {
				_ = hub.Broadcast("folder_updated", map[string]any{
					"key":     folder.Key,
					"etag":    entry.ETag,
					"version": entry.Version,
				})
			}
		}
	}
}

func startSyncLoop(client *storage.DBClient, c *cache.EngineCache, hub *sse.Hub, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := client.Ping(ctx); err == nil {
			syncFromDB(ctx, client, c, hub)
		}
		cancel()
	}
}
