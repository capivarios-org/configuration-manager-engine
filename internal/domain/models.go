package domain

import (
	"encoding/json"
	"time"
)

// PolicyTransport define os protocolos de transporte permitidos pelo Control Plane
type PolicyTransport string

const (
	TransportAllowAll    PolicyTransport = "ALLOW_ALL"
	TransportSSEOnly     PolicyTransport = "SSE_ONLY"
	TransportPollingOnly PolicyTransport = "POLLING_ONLY"
)

// PolicyConsume define os alvos de consumo permitidos pelo Control Plane
type PolicyConsume string

const (
	ConsumeAllowAll   PolicyConsume = "ALLOW_ALL"
	ConsumeFlagOnly   PolicyConsume = "FLAG_ONLY"
	ConsumeFolderOnly PolicyConsume = "FOLDER_ONLY"
)

// GovernancePolicy representa as regras de governança aplicadas ao ambiente/cliente
type GovernancePolicy struct {
	TransportPolicy PolicyTransport `json:"transport_policy"`
	ConsumePolicy   PolicyConsume   `json:"consume_policy"`
}

// Flag representa uma Feature Flag individual
type Flag struct {
	Key         string          `json:"key"`
	Value       json.RawMessage `json:"value"`
	IsEnabled   bool            `json:"is_enabled"`
	Version     int64           `json:"version"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// Folder representa um agrupamento de flags para um serviço/domínio
type Folder struct {
	Key       string                     `json:"key"`
	Flags     map[string]json.RawMessage `json:"flags"`
	Version   int64                      `json:"version"`
	UpdatedAt time.Time                  `json:"updated_at"`
}

// CacheEntry armazena o valor processado e seu respectivo ETag em memória
type CacheEntry struct {
	Value     []byte    `json:"value"`
	ETag      string    `json:"etag"`
	Version   int64     `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}
