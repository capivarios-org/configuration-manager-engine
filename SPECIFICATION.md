# 📐 Especificação Técnica: Configuration Manager Engine

> **Repositório:** `capivarios-org/configuration-manager-engine`  
> **Papel:** Data Plane (Motor de Leitura de Alta Performance)  
> **Status:** Autônomo / Standalone

---

## 1. Princípio Fundamental de Autonomia

O **Configuration Manager Engine** foi rigorosamente arquitetado para ser **100% independente do painel administrativo (Admin / Control Plane)**.

```plaintext
+-------------------------------------------------------------------------+
|                  CENÁRIO 1: ENGINE TOTALMENTE AUTÔNOMO                  |
|                                                                         |
|   [Engenheiro / Script / CI]                                            |
|                |                                                        |
|                v (INSERT / UPDATE direto via SQL, DBeaver, Script)      |
|   +---------------------------------------+                             |
|   |         BANCO DE DADOS                |                             |
|   |   (Apenas o Core Schema Mínimo)       |                             |
|   +-------------------+-------------------+                             |
|                       | (Leitura & Polling)                             |
|                       v                                                 |
|   +---------------------------------------+                             |
|   |      CONFIGURATION MANAGER ENGINE     |                             |
|   |    - In-Memory RAM Cache (sync.Map)   |                             |
|   |    - Resposta Sub-millisecond < 0.5ms |                             |
|   |    - ETag Generator / HTTP 304        |                             |
|   |    - SSE Streamer em tempo real       |                             |
|   +-------------------+-------------------+                             |
|                       ^                                                 |
|                       | (HTTP / SSE)                                    |
|              +--------+--------+                                        |
|              |                 |                                        |
|        [SDK Go Client]   [SDK PHP Client]                               |
+-------------------------------------------------------------------------+
```

### O que isso significa na prática?
1. **Zero dependência de runtime:** O Engine não requer PHP, Symfony, Node ou qualquer outro componente do Admin para subir ou executar.
2. **Gestão agnóstica de dados:** Se uma equipe preferir não utilizar o Admin oficial, pode gerenciar flags e folders via migrações SQL no seu próprio pipeline, scripts Python/Bash, interfaces de banco (DBeaver, DataGrip) ou UIs proprietárias.
3. **Resiliência Máxima:** Mesmo se o Admin cair ou for desativado para manutenção, a Engine continua servindo flags aos SDKs sem nenhuma interrupção.
4. **Tolerância a Extensões de Schema:** O Engine seleciona estritamente as colunas do Core Schema. Tabelas adicionais criadas pelo Admin (como `users`, `audit_logs`, `metrics`) são solenemente ignoradas pelo Engine e não causam nenhum impacto na performance.

---

## 2. Core Schema (Schema Mínimo do Banco de Dados)

O Engine requer exclusivamente as tabelas fundamentais de configuração.

### 2.1. Tabelas do Core Schema

#### `environments`
Define os ambientes de execução (ex: `production`, `staging`, `development`).
- `id` (VARCHAR / INT PK)
- `key` (VARCHAR UNIQUE, ex: `production`)
- `name` (VARCHAR)
- `created_at` (TIMESTAMP)

#### `folders`
Agrupamento lógico de configurações para consumo consolidado em lote.
- `id` (VARCHAR / INT PK)
- `key` (VARCHAR UNIQUE, ex: `checkout-service`)
- `name` (VARCHAR)
- `environment_key` (VARCHAR FK -> `environments.key`)
- `version` (INT ou VARCHAR de versão / ETag)
- `updated_at` (TIMESTAMP)

#### `flags`
Feature flags individuais.
- `id` (VARCHAR / INT PK)
- `key` (VARCHAR UNIQUE, ex: `new-checkout-v2`)
- `folder_key` (VARCHAR NULL FK -> `folders.key`)
- `environment_key` (VARCHAR FK -> `environments.key`)
- `type` (VARCHAR: `boolean`, `string`, `json`, `number`)
- `value` (TEXT / JSONB)
- `enabled` (BOOLEAN)
- `rollout_percentage` (INT: 0 a 100)
- `version` (INT ou ETag)
- `updated_at` (TIMESTAMP)

---

## 3. Arquitetura de Leitura & Cache

```plaintext
      Requisição SDK (GET /api/v1/flags/my-flag)
                     │
                     ▼
          Existe no In-Memory Cache?
          ├── SIM:
          │    ├── Header "If-None-Match" == ETag em memória?
          │    │    ├── SIM  ──> Retorna HTTP 304 Not Modified (< 0.1ms)
          │    │    └── NÃO  ──> Retorna HTTP 200 com Payload & ETag (< 0.5ms)
          └── NÃO:
               Carrega do Banco de Dados, calcula ETag, armazena no Cache e retorna.
```

- **Sincronização com o Banco:** Polling de intervalo curto ou trigger para invalidação de cache local.
- **Server-Sent Events (SSE):** Hub de eventos em tempo real (`sse.Hub`) que notifica clientes conectados instantaneamente quando o valor de uma flag ou folder é alterado.

---

## 4. Matriz de Compatibilidade de Banco de Dados

O Engine suporta conexão direta com 4 SGBDs através da variável `DB_DRIVER`:
- `postgresql` (driver: `pgx/v5`)
- `mysql` (driver: `go-sql-driver/mysql`)
- `mariadb` (driver: `go-sql-driver/mysql`)
- `oracle` (driver: `sijms/go-ora/v2`)
