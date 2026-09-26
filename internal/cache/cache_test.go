package cache

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/capivarios-org/configuration-manager-engine/internal/domain"
)

func TestEngineCache_FlagOperations(t *testing.T) {
	c := NewEngineCache()

	rawVal := json.RawMessage(`{"enabled": true, "rollout_percentage": 50}`)
	flag := domain.Flag{
		Key:       "payment-v2",
		Value:     rawVal,
		IsEnabled: true,
		Version:   1,
		UpdatedAt: time.Now(),
	}

	entry, err := c.SetFlag(flag)
	if err != nil {
		t.Fatalf("unexpected error setting flag: %v", err)
	}

	if entry.ETag == "" {
		t.Fatal("expected non-empty ETag")
	}

	// Recuperação rápida em RAM
	start := time.Now()
	retrieved, found := c.GetFlag("payment-v2")
	elapsed := time.Since(start)

	if !found {
		t.Fatal("expected flag to be found in cache")
	}

	if retrieved.ETag != entry.ETag {
		t.Fatalf("expected ETag %s, got %s", entry.ETag, retrieved.ETag)
	}

	if elapsed > 500*time.Microsecond {
		t.Logf("Warning: retrieval took %v, higher than 0.5ms target on cold run", elapsed)
	}
}

func TestEngineCache_FolderOperations(t *testing.T) {
	c := NewEngineCache()

	folder := domain.Folder{
		Key: "checkout-service",
		Flags: map[string]json.RawMessage{
			"one-click-buy": json.RawMessage(`true`),
			"discount-code": json.RawMessage(`"SPRING2026"`),
		},
		Version:   2,
		UpdatedAt: time.Now(),
	}

	entry, err := c.SetFolder(folder)
	if err != nil {
		t.Fatalf("unexpected error setting folder: %v", err)
	}

	retrieved, found := c.GetFolder("checkout-service")
	if !found {
		t.Fatal("expected folder to be found in cache")
	}

	if retrieved.ETag != entry.ETag {
		t.Fatalf("expected ETag %s, got %s", entry.ETag, retrieved.ETag)
	}
}
