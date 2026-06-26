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
| `C2_PORT` | `8443` | HTTPS/WSS port |
| `C2_TLS_CERT` | `./certs/server.crt` | TLS certificate (lab self-signed ok) |
| `C2_TLS_KEY` | `./certs/server.key` | TLS private key |
| `C2_MASTER_KEY` | — | 32-byte hex; session key encryption at-rest |
| `C2_JWT_SECRET` | — | Operator JWT signing secret |
| `C2_REGISTRY_ADDRESS` | — | `C2Registry` contract address on Polygon Amoy |
| `C2_RPC_URL` | — | Polygon Amoy RPC endpoint |
| `C2_OPERATOR_WALLET_KEY` | — | Private key for on-chain operator calls (lab only) |

> **Never commit `.env`** — it contains secrets.

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

```bash
go run ./cmd/server
```

Server starts on `https://localhost:8443`. Healthcheck:

```bash
curl -k https://localhost:8443/api/v1/health
```

## Run the agent (Fase 2)

```bash
C2_SERVER_URL=https://localhost:8443 go run ./cmd/agent
```

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
