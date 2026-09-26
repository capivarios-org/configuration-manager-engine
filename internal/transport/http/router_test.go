package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/capivarios-org/configuration-manager-engine/internal/cache"
	"github.com/capivarios-org/configuration-manager-engine/internal/domain"
	"github.com/capivarios-org/configuration-manager-engine/internal/sse"
)

func TestRouter_SmartPollingAndETag(t *testing.T) {
	c := cache.NewEngineCache()
	hub := sse.NewHub()
	server := NewServer(c, hub)

	// Inserir flag teste
	flag := domain.Flag{
		Key:       "dark-mode",
		Value:     json.RawMessage(`{"theme": "dark"}`),
		IsEnabled: true,
		Version:   1,
		UpdatedAt: time.Now(),
	}
	entry, err := c.SetFlag(flag)
	if err != nil {
		t.Fatalf("failed to seed flag: %v", err)
	}

	// 1. Primeira chamada: Deve retornar 200 OK com ETag
	req := httptest.NewRequest("GET", "/api/v1/flags/dark-mode", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", w.Code)
	}

	etag := w.Header().Get("ETag")
	if etag != entry.ETag {
		t.Fatalf("expected ETag %s, got %s", entry.ETag, etag)
	}

	// 2. Segunda chamada com If-None-Match: Deve retornar 304 Not Modified imediato
	req304 := httptest.NewRequest("GET", "/api/v1/flags/dark-mode", nil)
	req304.Header.Set("If-None-Match", etag)
	w304 := httptest.NewRecorder()
	server.ServeHTTP(w304, req304)

	if w304.Code != http.StatusNotModified {
		t.Fatalf("expected status 304 Not Modified, got %d", w304.Code)
	}

	if w304.Body.Len() > 0 {
		t.Fatalf("expected empty body for 304, got %d bytes", w304.Body.Len())
	}
}

func TestRouter_GovernanceEnforcement(t *testing.T) {
	c := cache.NewEngineCache()
	hub := sse.NewHub()
	server := NewServer(c, hub)

	// Adicionar política que proíbe Smart Polling
	c.SetPolicy("restricted-flag", domain.GovernancePolicy{
		TransportPolicy: domain.TransportSSEOnly,
		ConsumePolicy:   domain.ConsumeAllowAll,
	})

	// Tentar acessar via HTTP GET Polling
	req := httptest.NewRequest("GET", "/api/v1/flags/restricted-flag", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden due to SSE_ONLY policy, got %d", w.Code)
	}
}
