# C2 Blockchain-Blindado — Setup

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.22+ | `go version` to check |
| Node.js | 20+ | Required for Hardhat contract tests |
| Redis | 7.x | Can run via Docker (see below) |
| Docker | 24+ | Optional — used for Redis + testcontainers in integration tests |

### Quick install check

```bash
go version          # go1.22.x or higher
node --version      # v20.x or higher
npm --version       # bundled with Node
redis-cli --version # 7.x
docker --version    # 24.x
```

## Environment variables

Copy and edit before running:

```bash
cp .env.example .env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `C2_DB_PATH` | `./c2.db` | SQLite database path |
| `C2_REDIS_ADDR` | `localhost:6379` | Redis address |
| `C2_PORT` | `8443` | HTTP/HTTPS port |
| `C2_INSECURE` | `true` | Lab: HTTP plano sin TLS (`http://localhost:8443`) |
| `C2_TLS_CERT` | `./certs/server.crt` | TLS certificate (lab self-signed ok) |
| `C2_TLS_KEY` | `./certs/server.key` | TLS private key |
| `C2_MASTER_KEY` | — | 32-byte hex; session key encryption at-rest |
| `C2_JWT_SECRET` | — | Operator JWT signing secret |
| `C2_OPERATOR_USER` | `operator` | Login operador (lab) |
| `C2_OPERATOR_PASS` | `lab` | Password operador (lab) |
| `C2_REGISTRY_ADDRESS` | — | `C2Registry` contract address on Polygon Amoy |
| `C2_RPC_URL` | — | Polygon Amoy RPC endpoint |
| `C2_OPERATOR_WALLET_KEY` | — | Private key for on-chain operator calls (lab only) |
| `C2_SERVER_URL` | `http://localhost:8443` | URL del server para el agente (usar `http` si `C2_INSECURE=true`) |

> **Never commit `.env`** — it contains secrets.
>
> El server y el agente **no cargan `.env` automáticamente**. Exporta variables antes de `go run` o usa `set -a && source .env && set +a` (bash).

## Redis via Docker (dev)

```bash
docker run -d --name c2-redis -p 6379:6379 redis:7-alpine
```

## Smart contracts (Hardhat)

```bash
cd contracts
npm ci
npx hardhat test              # run contract tests
npx hardhat run scripts/deploy.js --network amoy   # deploy to Polygon Amoy
```

Required in `contracts/.env`:
```
AMOY_RPC_URL=https://rpc-amoy.polygon.technology
OPERATOR_PRIVATE_KEY=0x...
```

## Database migrations

Migrations are applied automatically on server start via `migrations/001_initial.sql`. To apply manually:

```bash
sqlite3 c2.db < migrations/001_initial.sql
```

## Run the server (Fase 2)

Desde `ingeleanh/c2-blockchain/`:

```bash
export C2_INSECURE=true
# opcional: set -a && source .env && set +a
go run ./cmd/server
```

Con `C2_INSECURE=true` el server usa **HTTP** en `:8443`. Healthcheck:

```bash
curl -s http://localhost:8443/api/v1/health
```

## Run the agent (Fase 2)

Segunda terminal, misma carpeta:

```bash
export C2_SERVER_URL=http://localhost:8443
go run ./cmd/agent
```

Esperado en logs: `agent registered: <uuid>`.

Modo gateway IoT (opcional):

```bash
C2_IOT_GATEWAY=true C2_SERVER_URL=http://localhost:8443 go run ./cmd/agent
```

---

## Prueba de lab — guion completo (v0.1.0)

Historial de comandos validado en Windows (Git Bash) para demostrar el **mínimo Aligo**: servidor, agente, operador, comando remoto `whoami`, dashboard.

### 0. Prerrequisitos

```bash
cd ingeleanh/c2-blockchain
cp .env.example .env   # editar si hace falta; no commitear .env
go version             # >= 1.22
docker run -d --name c2-redis -p 6379:6379 redis:7-alpine
```

### 1. Tests automatizados (opcional, antes de demo)

```bash
go test -race -cover ./...
cd contracts && npm ci && npx hardhat test && cd ..
```

### 2. Terminal A — C2 Server

```bash
cd ingeleanh/c2-blockchain
export C2_INSECURE=true
go run ./cmd/server
```

### 3. Terminal B — Agente (una sola instancia)

```bash
cd ingeleanh/c2-blockchain
export C2_SERVER_URL=http://localhost:8443
go run ./cmd/agent
```

Anota el `agent_id` del agente activo (o usa el que tenga `last_beacon` reciente en el dashboard).

### 4. Terminal C — Verificación API (operador)

**Health**

```bash
curl -s http://localhost:8443/api/v1/health
```

**Login operador → copiar solo el valor de `token` (cadena `eyJ...`)**

```bash
curl -s http://localhost:8443/api/v1/operator/login \
  -H "Content-Type: application/json" \
  -d '{"username":"operator","password":"lab"}'
```

**Listar agentes**

```bash
curl -s http://localhost:8443/api/v1/agents \
  -H "Authorization: Bearer TU_TOKEN_AQUI"
```

**Crear tarea `whoami` (JSON en una sola línea; reemplazar `AGENT_ID`)**

```bash
curl -s http://localhost:8443/api/v1/tasks \
  -H "Authorization: Bearer TU_TOKEN_AQUI" \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"AGENT_ID","command_type":"shell","payload":{"argv":["whoami"]}}'
```

Respuesta esperada: `{"status":"pending","task_id":"<uuid>"}`

**Consultar resultado (esperar 10–15 s; reemplazar `TASK_ID`)**

```bash
curl -s http://localhost:8443/api/v1/tasks/TASK_ID \
  -H "Authorization: Bearer TU_TOKEN_AQUI"
```

Respuesta esperada: `status: completed`, `exit_code: 0`, `stdout` con el usuario del host.

**Estado blockchain (lab sin deploy)**

```bash
curl -s http://localhost:8443/api/v1/chain/status \
  -H "Authorization: Bearer TU_TOKEN_AQUI"
```

### 5. Dashboard

1. Abrir `http://localhost:8443/dashboard/`
2. Pegar en el campo JWT **solo** el token (`eyJ...`), no el JSON completo
3. **Guardar token** → **Actualizar**
4. Verificar: tabla **Agentes** con `active` y `last_beacon` reciente; barra inferior sin `undefined`

Consultar resultado de tarea desde consola del navegador (F12):

```javascript
fetch('/api/v1/tasks/TASK_ID', {
  headers: { Authorization: 'Bearer ' + localStorage.getItem('c2_jwt') }
}).then(r => r.json()).then(console.log);
```

### 6. Comando IoT cerradura (opcional)

Con agente en modo gateway (`C2_IOT_GATEWAY=true`):

```bash
curl -s http://localhost:8443/api/v1/tasks \
  -H "Authorization: Bearer TU_TOKEN_AQUI" \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"AGENT_ID","command_type":"iot_command","payload":{"target_device":"lock-main","action":"unlock","duration_sec":5}}'
```

### Problemas frecuentes

| Síntoma | Causa | Solución |
|---------|--------|----------|
| `handshake: unexpected end of JSON input` | Bug agente leyendo respuesta incorrecta | Usar versión actual de `cmd/agent` |
| Agent no conecta | `C2_SERVER_URL` con `https` y server en HTTP | `export C2_SERVER_URL=http://localhost:8443` |
| Handshake falla | Redis no corre | `docker start c2-redis` o crear contenedor |
| `NOT_FOUND` en tarea | `agent_id` con salto de línea en JSON multilínea | Usar JSON en **una línea** |
| Dashboard vacío | JWT vacío o JSON completo pegado | Pegar solo `eyJ...` y Guardar token |
| 6 agentes en dashboard | Varias instancias de agente | Dejar una sola terminal con `go run ./cmd/agent` |

---

## Run tests

```bash
# All Go tests (unit + integration)
go test -race -cover ./...

# Integration only (requires Docker for testcontainers-go Redis)
go test -race -tags integration ./internal/api/...

# Contract tests
cd contracts && npx hardhat test
```

## Project structure (Fase 2 target)

```
c2-blockchain/
├── cmd/
│   ├── server/         # C2 server entrypoint
│   └── agent/          # Agent entrypoint
├── contracts/          # Solidity + Hardhat
│   └── C2Registry.sol
├── internal/
│   ├── api/            # HTTP handlers, WebSocket hub
│   ├── chain/          # Blockchain indexer + eth_call
│   ├── crypto/         # AES-GCM, ECDSA, ECDH, HKDF
│   ├── executor/       # Shell command runner (Linux + Windows)
│   └── handshake/      # Nonce, challenge-response, session
├── migrations/
│   └── 001_initial.sql
├── tests/fixtures/     # Test keys (never production)
├── docs/sdd/           # SDD documents (SSOT)
└── frontend/           # Dashboard HTML/JS (static, served by server)
```

## References

- [SDD documentation](./docs/sdd/README.md)
- [API design](./docs/sdd/03_api_design.md)
- [Data models](./docs/sdd/04_data_models.md)
- [Testing strategy](./docs/sdd/06_testing_strategy.md)
