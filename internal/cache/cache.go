package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/capivarios-org/configuration-manager-engine/internal/domain"
)

// EngineCache gerencia o cache em memória RAM usando sync.Map para garantir latência < 0.5ms
type EngineCache struct {
	flags    sync.Map // key string -> *domain.CacheEntry
	folders  sync.Map // key string -> *domain.CacheEntry
	policies sync.Map // key string -> *domain.GovernancePolicy
}

// NewEngineCache inicializa uma nova instância de EngineCache
func NewEngineCache() *EngineCache {
	return &EngineCache{}
}

// generateETag gera uma hash SHA-256 rápida e única para o conteúdo
func generateETag(content []byte, version int64) string {
	h := sha256.New()
	h.Write(content)
	h.Write([]byte(fmt.Sprintf(":%d", version)))
	return `"` + hex.EncodeToString(h.Sum(nil))[:16] + `"`
}

// SetFlag armazena ou atualiza uma Flag no cache com ETag calculado
func (c *EngineCache) SetFlag(flag domain.Flag) (*domain.CacheEntry, error) {
	bytes, err := json.Marshal(flag)
	if err != nil {
		return nil, err
	}

	entry := &domain.CacheEntry{
		Value:     bytes,
		ETag:      generateETag(bytes, flag.Version),
		Version:   flag.Version,
		UpdatedAt: flag.UpdatedAt,
	}

	if entry.UpdatedAt.IsZero() {
		entry.UpdatedAt = time.Now()
	}

	c.flags.Store(flag.Key, entry)
	return entry, nil
}

// GetFlag recupera uma Flag do cache em RAM (< 0.5ms)
func (c *EngineCache) GetFlag(key string) (*domain.CacheEntry, bool) {
	val, ok := c.flags.Load(key)
	if !ok {
		return nil, false
	}
	return val.(*domain.CacheEntry), true
}

// SetFolder armazena ou atualiza um Folder no cache com ETag calculado
func (c *EngineCache) SetFolder(folder domain.Folder) (*domain.CacheEntry, error) {
	bytes, err := json.Marshal(folder.Flags)
	if err != nil {
		return nil, err
	}

	entry := &domain.CacheEntry{
		Value:     bytes,
		ETag:      generateETag(bytes, folder.Version),
		Version:   folder.Version,
		UpdatedAt: folder.UpdatedAt,
	}

	if entry.UpdatedAt.IsZero() {
		entry.UpdatedAt = time.Now()
	}

	c.folders.Store(folder.Key, entry)
	return entry, nil
}

// GetFolder recupera um Folder do cache em RAM (< 0.5ms)
func (c *EngineCache) GetFolder(key string) (*domain.CacheEntry, bool) {
	val, ok := c.folders.Load(key)
	if !ok {
		return nil, false
	}
	return val.(*domain.CacheEntry), true
}

// SetPolicy atualiza as políticas de governança para uma chave ou ambiente
func (c *EngineCache) SetPolicy(targetKey string, policy domain.GovernancePolicy) {
	c.policies.Store(targetKey, &policy)
}

// GetPolicy recupera as políticas de governança
func (c *EngineCache) GetPolicy(targetKey string) (*domain.GovernancePolicy, bool) {
	val, ok := c.policies.Load(targetKey)
	if !ok {
		return nil, false
	}
	return val.(*domain.GovernancePolicy), true
}
