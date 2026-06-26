# Guion — Demo en vivo (jurado Aligo)

Duración estimada: **15–20 minutos** (incluye preguntas cortas).

## Preparación (5 min antes)

- [ ] Redis: `docker start c2-redis`
- [ ] Solo **una** terminal con agente (cerrar instancias duplicadas)
- [ ] Server: `go run ./cmd/server` desde `ingeleanh/c2-blockchain`
- [ ] Agent: `go run ./cmd/agent`
- [ ] Browser: `http://localhost:8443/dashboard/` abierto
- [ ] Polygonscan Amoy abierto en otra pestaña
- [ ] Token JWT fresco (login) o guardado en dashboard

---

## 1. Contexto (2 min) — Narrativa Aligo

Decir en voz alta:

1. **C2 propio** en Go: servidor, protocolo handshake ECDSA, agente, no Metasploit-as-C2.
2. **Blockchain** como registry de confianza en Polygon Amoy: operadores y config de failover (URL hasheada, no en claro).
3. **Fusión residencial**: gateway IoT simulado (sensores, cerradura) — stretch opcional en vivo.
4. **Lab autorizado**: red local, sin terceros.

---

## 2. DEMO-001 — Servidor y salud (2 min)

**Terminal (mostrar):**

```bash
curl -s http://localhost:8443/api/v1/health
```

Destacar: `status: ok`, `registry_address` con contrato Amoy.

**Decir:** SQLite + Redis + lectura RPC Amoy configurada.

---

## 3. DEMO-002 — Agente conectado (2 min)

**Dashboard:** Guardar JWT → **Actualizar**.

Mostrar tabla **Agentes**: uno con `last_beacon` reciente, `windows-amd64`, `active`.

**Decir:** Handshake ECDSA + sesión AES-GCM establecida; beacon cada ~30 s.

---

## 4. DEMO-005 — Blockchain (3 min)

**Dashboard footer:** Chain **v1**, Registry `0x6292...`, red **polygon-amoy (80002)**.

**Terminal:**

```bash
curl -s http://localhost:8443/api/v1/chain/status -H "Authorization: Bearer TOKEN"
```

Mostrar: `config_version: 1`, `endpoint_hash` (sin URL en claro).

**Browser:** https://amoy.polygonscan.com/address/0x629238eD79c23267fe502AAd81E5AEfee3908750

Mostrar transacciones de deploy y `updateConfig`.

**Decir:** El agente puede leer `getConfig()` para failover si el endpoint primario cae.

---

## 5. DEMO-003 — Comando remoto whoami (3 min)

**Terminal (una línea, `AGENT_ID` del dashboard):**

```bash
curl -s http://localhost:8443/api/v1/tasks \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"AGENT_ID","command_type":"shell","payload":{"argv":["whoami"]}}'
```

Anotar `task_id`. Esperar ~15 s.

```bash
curl -s http://localhost:8443/api/v1/tasks/TASK_ID -H "Authorization: Bearer TOKEN"
```

Mostrar `status: completed`, `stdout` con usuario del host.

**Decir:** Operador → API REST → Redis queue → beacon → executor en Windows.

---

## 6. DEMO-004 + DEMO-009 — Cifrado y camuflaje (2 min)

Sin mostrar secretos:

- Beacons usan **envelope AES-GCM** (`encrypted` en JSON).
- Tráfico parece API REST/IoT normal; la URL del C2 no viaja en claro on-chain (solo hash SHA-256).
- JWT solo para operador humano; agente usa handshake propio.

Opcional: logs del server (`middleware.Logger`) con POST beacon.

---

## 7. Stretch — IoT (DEMO-007 / DEMO-008) (3 min, opcional)

Reiniciar agente en modo gateway:

```bash
C2_IOT_GATEWAY=true go run ./cmd/agent
```

Eventos simulados en beacon. Dashboard → eventos (si hay).

Unlock cerradura:

```bash
curl -s http://localhost:8443/api/v1/tasks \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"AGENT_ID","command_type":"iot_command","payload":{"target_device":"lock-main","action":"unlock","duration_sec":5}}'
```

---

## 8. Cierre (1 min)

Recapitular MVPs:

| MVP | Demostrado |
|-----|------------|
| Registro + beacon | ✅ |
| Comando remoto + resultado | ✅ |
| Lectura on-chain Amoy | ✅ |
| C2 propio + innovación blockchain | ✅ |

Invitar a ver repo: `docs/sdd/`, tests `go test ./...`, contratos Hardhat.

---

## Si algo falla

| Problema | Acción rápida |
|----------|----------------|
| Agent sin beacon | Reiniciar un solo `go run ./cmd/agent` |
| Tarea pending | Esperar 30 s; verificar agente activo |
| Chain v0 | Reiniciar server; verificar `C2_REGISTRY_ADDRESS` en `.env` |
| JWT inválido | Login de nuevo; pegar solo `eyJ...` en dashboard |
