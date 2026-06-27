# Demo failover (MVP-05 / DEMO-006)

El agente lee `getConfig()` del contrato en Polygon Amoy y solo conecta URLs cuyo `SHA-256` coincide con `endpointHash` on-chain.

## Variables del agente

```bash
export C2_SERVER_URL=http://localhost:8443
export C2_URL_CANDIDATES=http://localhost:8443,http://127.0.0.1:8443
export C2_REGISTRY_ADDRESS=0x629238eD79c23267fe502AAd81E5AEfee3908750
export C2_RPC_URL=https://rpc-amoy.polygon.technology
```

**Importante:** el hash on-chain debe corresponder a la URL del lab. Si desplegaste con `https://localhost:8443` pero usas HTTP, incluye en candidatos la URL que coincida con el deploy o vuelve a desplegar con:

```bash
cd contracts
C2_SERVER_URL=http://localhost:8443 npx hardhat run scripts/deploy.js --network amoy
```

## Demo en vivo (≈ 2 min)

1. Server + agente corriendo (beacons OK).
2. **Detener el server** (`Ctrl+C` en terminal del server).
3. En el agente verás `beacon error` y tras 2 fallos: `failover: chain v1 verified endpoint → ...`
4. **Reiniciar el server** (`go run ./cmd/server`).
5. El agente reconecta: `failover: re-registered agent ...`

Decir al jurado: la URL no viaja en claro en la chain; el agente verifica el hash y reconecta al endpoint autorizado.

## Multi-OS (MVP-07)

```bash
# Windows PowerShell
./scripts/build-agents.ps1

# Linux / Git Bash
./scripts/build-agents.sh
```

Binarios en `dist/`: `c2-agent-linux-amd64`, `c2-agent-windows-amd64.exe`, etc.

En Linux VM o WSL:

```bash
export C2_SERVER_URL=http://<ip-lab>:8443
./dist/c2-agent-linux-amd64
```

El portal muestra `os` real (`linux-amd64` / `windows-amd64`) en la tabla de agentes.

## DEMO-011 — Flujo 3 capas

En el portal: botón **Flujo 3 capas** o:

```bash
curl -s -X POST http://localhost:8443/api/v1/demo/three-layer \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{}'
```

Orden: evento IoT Laureles → agentes C2 activos → `getConfig` Amoy.
