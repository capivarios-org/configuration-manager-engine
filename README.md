# 🚀 Configuration Manager Engine (Data Plane)

Motor de leitura em tempo real e altíssima performance (< 0.5ms) para Feature Flags e configurações distribuídas, parte do ecossistema [Capivarios](https://github.com/capivarios-org).

Projetado para ser **autossuficiente e ultraleve**, o Engine pode rodar isolado ou integrado a qualquer interface de gestão.

---

## ⚡ Principais Recursos

- **Sub-millisecond Latency (< 0.5ms):** Cache em memória RAM (`sync.Map`) alimentado diretamente pelo banco de dados.
- **Server-Sent Events (SSE):** Stream HTTP persistente com push em tempo real para SDKs e microsserviços.
- **Smart Polling (ETag / HTTP 304):** Geração instantânea de hashes ETag em memória com respostas `304 Not Modified` com custo zero de CPU/DB.
- **Multi-Database Support (Matrix DB):** Suporte nativo aos 4 principais bancos de dados de mercado:
  - PostgreSQL
  - MySQL
  - MariaDB
  - Oracle Database
- **Autossuficiente:** Não depende de painel administrativo (Control Plane) para operar.

---

## 🛠️ Stack Tecnológica

- **Linguagem:** Go 1.25+
- **Acesso a Banco:** `database/sql` nativo com drivers otimizados (`pgx/v5`, `go-sql-driver/mysql`, `go-ora/v2`)
- **Cache Local:** In-Memory `sync.Map` concorrente e thread-safe
- **Containerização:** Docker & Docker Compose com suporte a Matrix Testing

---

## 📡 Endpoints da API

| Método | Rota | Descrição |
| :--- | :--- | :--- |
| `GET` | `/health` | Healthcheck do serviço |
| `GET` | `/api/v1/flags/{key}` | Consulta de flag individual (suporta `If-None-Match` / ETag) |
| `GET` | `/api/v1/folders/{key}` | Consulta de folder com agrupamento de flags |
| `GET` | `/api/v1/stream` | Canal persistente SSE para eventos em tempo real |

---

## ⚙️ Variáveis de Ambiente

Crie um arquivo `.env` baseado no `.env.example`:

```env
PORT=8080

# Seleção do banco ativo (postgresql | mysql | mariadb | oracle)
DB_DRIVER=postgresql

# Dados de conexão
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=config_manager
DB_USER=root
DB_PASSWORD=secret_password
```

---

## 🚀 Executando Localmente

### 1. Iniciar os bancos de dados (Docker)
```bash
docker compose up -d postgres
```

### 2. Executar o Engine
```bash
go run ./cmd/server
```

### 3. Rodar os testes
```bash
go test -v ./...
```

---

## 📄 Licença

Distribuído sob a licença open source. Veja [LICENSE](LICENSE) para mais detalhes.
