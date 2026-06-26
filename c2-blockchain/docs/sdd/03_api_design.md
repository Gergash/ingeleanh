# 03 — API Design

## Convenciones generales

| Parámetro | Valor |
|-----------|-------|
| Base URL REST | `https://{host}:8443/api/v1` |
| WebSocket | `wss://{host}:8443/api/v1/ws/agent` |
| Content-Type | `application/json` |
| TLS | 1.3 obligatorio en lab/producción |
| Versionado | Prefijo `/api/v1`; header opcional `X-API-Version: 1` |

## Tipos de tarea (`command_type`)

El **canal C2 es propio** (handshake, beacon, envelopes). Los tipos de tarea definen qué ejecuta el agente en el host (Linux o Windows).

| `command_type` | Origen | Descripción |
|----------------|--------|-------------|
| `shell` | C2 propio (obligatorio MVP) | Ejecuta argv en shell del SO (`whoami`, `id`, etc.) |
| `iot_command` | C2 propio | Comando hacia dispositivo vía gateway — **cerraduras, sensores simulados** |
| `msf_module` | Framework auxiliar (opcional) | Invoca módulo Metasploit en VM lab; **resultado retorna por el canal C2 propio** |

### `iot_command` — payload (cerraduras y dispositivos simulados)

Confirmado por retadores: **sí** se pueden enviar comandos a cerraduras (simuladas en lab).

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `target_device` | string | ID dispositivo (`lock-main`, `sensor-motion-01`) |
| `action` | string | `lock`, `unlock`, `status`, `read_telemetry` |
| `duration_sec` | int | Opcional; auto-relock tras unlock temporal |
| `zone` | string | Opcional; zona residencial |
| `reason` | string | Auditoría (`operator_authorized_lab`) |

**Ejemplo unlock cerradura**

```json
{
  "agent_id": "gateway-uuid",
  "command_type": "iot_command",
  "payload": {
    "target_device": "lock-main",
    "action": "unlock",
    "duration_sec": 5,
    "reason": "demo_jurado"
  }
}
```

**Respuesta simulada (`task_result` plaintext)**

```json
{
  "type": "task_result",
  "task_id": "uuid",
  "device_id": "lock-main",
  "state": "unlocked",
  "previous_state": "locked",
  "auto_relock_at": 1719000005
}
```

### Camuflaje HTTP (headers opcionales agente)

Para alinear tráfico con API IoT legítima (ver [05_security_specs.md](./05_security_specs.md)):

| Header | Valor ejemplo | Uso |
|--------|---------------|-----|
| `User-Agent` | `ResidentialHub/1.0 (gateway)` | Fingerprint menos obvio |
| `X-Client-Type` | `iot-gateway` | Identificación cover story |
| `X-Device-Firmware` | `2.1.0-lab` | Consistencia narrativa demo |

**Regla**: Metasploit no reemplaza el servidor ni el protocolo.

### Valores `os` en handshake (agente)

| Valor | Plataforma |
|-------|------------|
| `linux-amd64` | Agente Linux 64-bit |
| `linux-arm64` | Agente Linux ARM (gateway IoT) |
| `windows-amd64` | Agente Windows 64-bit |

## Headers comunes

### Agente

| Header | Obligatorio | Descripción |
|--------|-------------|-------------|
| `X-Agent-Id` | Tras handshake | UUID del agente registrado |
| `X-Timestamp` | Sí | Unix epoch segundos (anti-replay) |
| `X-Nonce` | Sí (beacon/handshake) | 32 bytes hex, único por request |
| `X-Signature` | Sí | ECDSA secp256k1 hex de `timestamp + nonce + body_hash` |
| `X-Session-Key-Id` | Tras handshake | ID de sesión activa |

### Operador

| Header | Obligatorio | Descripción |
|--------|-------------|-------------|
| `Authorization` | Sí | `Bearer {jwt}` — JWT emitido tras login operador |
| `X-Timestamp` | Sí | Unix epoch segundos |
| `X-Request-Id` | Opcional | UUID para trazabilidad |

## Envelope cifrado (AES-256-GCM)

Todos los payloads sensibles (post-handshake) se envían dentro de un envelope JSON. El plaintext interno se serializa como JSON UTF-8 antes de cifrar.

### Estructura

```json
{
  "v": 1,
  "alg": "AES-256-GCM",
  "iv": "base64-encoded-12-bytes",
  "ct": "base64-encoded-ciphertext",
  "tag": "base64-encoded-16-bytes"
}
```

| Campo | Tipo | Descripción |
|-------|------|-------------|
| `v` | int | Versión del envelope (actual: `1`) |
| `alg` | string | Algoritmo (`AES-256-GCM`) |
| `iv` | string | Nonce 12 bytes, base64 |
| `ct` | string | Ciphertext, base64 |
| `tag` | string | Auth tag 128-bit, base64 |

### Clave

Derivada en handshake vía ECDH P-256 + HKDF-SHA256 (ver [05_security_specs.md](./05_security_specs.md)). Rotación: nueva sesión tras `expires_at` o comando explícito `session_rotate`.

### Request con envelope

```json
{
  "encrypted": {
    "v": 1,
    "alg": "AES-256-GCM",
    "iv": "...",
    "ct": "...",
    "tag": "..."
  }
}
```

---

## REST Endpoints

### `GET /health`

Healthcheck sin autenticación.

**Response 200**

```json
{
  "status": "ok",
  "version": "0.1.0",
  "chain_connected": true,
  "registry_address": "0x..."
}
```

---

### `POST /agents/handshake`

Handshake en dos pasos. **No requiere** envelope cifrado (aún no hay clave de sesión).

#### Step 1: `challenge_request`

**Request**

```json
{
  "step": "challenge_request"
}
```

**Response 200**

```json
{
  "step": "challenge",
  "nonce": "hex-64-chars",
  "server_ecdh_pub": "base64-spki",
  "expires_at": 1719000060
}
```

#### Step 2: `challenge_response`

**Request**

```json
{
  "step": "challenge_response",
  "nonce": "hex-64-chars",
  "agent_ecdsa_pub": "hex-compressed-secp256k1",
  "agent_ecdh_pub": "base64-spki",
  "signature": "hex-ecdsa",
  "hostname": "lab-vm-01",
  "os": "linux-amd64",
  "timestamp": 1719000000
}
```

**Response 200**

```json
{
  "step": "complete",
  "agent_id": "550e8400-e29b-41d4-a716-446655440000",
  "session_key_id": "sess_abc123",
  "session_expires_at": 1719086400,
  "status": "registered"
}
```

**Errors**

| Código | `error_code` | Descripción |
|--------|--------------|-------------|
| 400 | `INVALID_STEP` | Step desconocido |
| 400 | `NONCE_EXPIRED` | Nonce expirado o inválido |
| 400 | `SIGNATURE_INVALID` | Firma ECDSA inválida |
| 409 | `NONCE_REUSED` | Replay de nonce |
| 429 | `RATE_LIMITED` | Demasiados intentos |

---

### `POST /agents/{agent_id}/beacon`

Fallback REST cuando WebSocket no está disponible.

**Headers**: `X-Agent-Id`, `X-Timestamp`, `X-Nonce`, `X-Signature`, `X-Session-Key-Id`

**Request**

```json
{
  "encrypted": { "v": 1, "alg": "AES-256-GCM", "iv": "...", "ct": "...", "tag": "..." }
}
```

**Plaintext interno (beacon)**

```json
{
  "type": "beacon",
  "agent_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": 1719000030,
  "nonce": "hex-32-bytes",
  "status": "idle"
}
```

**Response 200 — ACK sin tarea**

Envelope cifrado con plaintext:

```json
{
  "type": "ack",
  "timestamp": 1719000031
}
```

**Response 200 — Con tarea**

Envelope cifrado con plaintext:

```json
{
  "type": "task",
  "task_id": "task-uuid",
  "command_type": "shell",
  "payload": { "argv": ["whoami"] }
}
```

**Errors**: 401 `SESSION_INVALID`, 403 `AGENT_MISMATCH`, 409 `REPLAY_DETECTED`, 429 `RATE_LIMITED`

---

### `POST /agents/{agent_id}/task_result`

Entrega resultado de tarea (REST fallback).

**Request**: envelope cifrado; plaintext:

```json
{
  "type": "task_result",
  "task_id": "task-uuid",
  "exit_code": 0,
  "stdout": "lab-user",
  "stderr": "",
  "timestamp": 1719000040,
  "nonce": "hex-32-bytes"
}
```

**Response 200**

```json
{
  "status": "accepted",
  "task_id": "task-uuid"
}
```

---

### `GET /agents`

Lista agentes registrados. **Requiere JWT operador**.

**Response 200**

```json
{
  "agents": [
    {
      "agent_id": "550e8400-e29b-41d4-a716-446655440000",
      "hostname": "lab-vm-01",
      "os": "linux-amd64",
      "status": "active",
      "first_seen": "2024-06-20T10:00:00Z",
      "last_beacon": "2024-06-20T12:30:00Z"
    }
  ],
  "total": 1
}
```

---

### `POST /tasks`

Crear tarea para un agente. **Requiere JWT operador**.

**Request**

```json
{
  "agent_id": "550e8400-e29b-41d4-a716-446655440000",
  "command_type": "shell",
  "payload": { "argv": ["whoami"] },
  "priority": 0
}
```

**Response 201**

```json
{
  "task_id": "task-uuid",
  "agent_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "created_at": "2024-06-20T12:30:05Z"
}
```

**Errors**: 400 `INVALID_COMMAND`, 404 `AGENT_NOT_FOUND`, 401 `UNAUTHORIZED`

---

### `GET /tasks/{task_id}`

Estado de tarea. **Requiere JWT operador**.

**Response 200**

```json
{
  "task_id": "task-uuid",
  "agent_id": "550e8400-e29b-41d4-a716-446655440000",
  "command_type": "shell",
  "status": "completed",
  "exit_code": 0,
  "stdout": "lab-user",
  "stderr": "",
  "created_at": "2024-06-20T12:30:05Z",
  "completed_at": "2024-06-20T12:30:12Z",
  "chain_tx_hash": null
}
```

---

### `POST /operator/login` (MVP operador)

**Request**

```json
{
  "username": "operator",
  "password": "..."
}
```

**Response 200**

```json
{
  "access_token": "jwt...",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

---

## WebSocket — `ws/agent`

### Conexión

```
wss://host:8443/api/v1/ws/agent
Headers:
  X-Agent-Id: {uuid}
  X-Timestamp: {epoch}
  X-Signature: {hex}
  X-Session-Key-Id: {sess_id}
```

Tras upgrade 101, el servidor valida sesión en Redis. Conexión permanece abierta para beacon bidireccional.

### Mensajes

Todos los frames son texto JSON. Estructura:

```json
{
  "encrypted": { "v": 1, "alg": "AES-256-GCM", "iv": "...", "ct": "...", "tag": "..." }
}
```

### Tipos de plaintext (agente → servidor)

| `type` | Descripción |
|--------|-------------|
| `beacon` | Heartbeat periódico |
| `task_result` | Resultado de comando |

### Tipos de plaintext (servidor → agente)

| `type` | Descripción |
|--------|-------------|
| `ack` | Beacon recibido, sin tareas |
| `task` | Tarea pendiente |
| `session_rotate` | Forzar re-handshake |

### Heartbeat WebSocket

- Ping/pong a nivel protocolo cada 30s
- Si no beacon en `2 * beacon_interval_sec`, servidor cierra conexión

---

## Dashboard & estado (nuevos endpoints MVP)

Endpoints consumidos por el panel de estado (dashboard HTML/JS estático servido por el propio servidor Go).

### `GET /api/v1/events`

Lista eventos IoT recientes (paginados). **Persistencia**: lectura desde `audit_log` (actions `iot_event`, `iot_telemetry`, `iot_command_result`) — ver [04_data_models.md](./04_data_models.md#endpoint-events--implementación-vía-audit_log).

| Header | Requerido | Valor |
|--------|-----------|-------|
| Authorization | Sí | `Bearer {jwt}` |

**Query params**

| Param | Tipo | Default | Descripción |
|-------|------|---------|-------------|
| `limit` | int | 50 | Máximo eventos |
| `offset` | int | 0 | Paginación |
| `gateway_id` | string | — | Filtro por gateway |

**Response 200**

```json
{
  "events": [
    {
      "id": "evt-uuid",
      "agent_id": "gateway-uuid",
      "event_type": "iot_event",
      "device_id": "sensor-motion-01",
      "payload_summary": "motion detected zone-1",
      "created_at": "2025-06-26T12:00:00Z"
    }
  ],
  "total": 142,
  "limit": 50,
  "offset": 0
}
```

### `GET /api/v1/devices/{id}/state`

Estado actual de un dispositivo IoT simulado.

**Response 200**

```json
{
  "device_id": "lock-main",
  "device_type": "smart_lock",
  "state": "locked",
  "last_action": "lock",
  "last_action_by": "operator-uuid",
  "updated_at": "2025-06-26T12:05:00Z"
}
```

### `GET /api/v1/chain/status`

Estado del indexer blockchain y config activa.

**Response 200**

```json
{
  "contract_address": "0xabc...def",
  "network": "polygon-amoy",
  "chain_id": 80002,
  "config_version": 3,
  "beacon_interval_sec": 30,
  "endpoint_hash": "0x...",
  "last_indexed_block": 42018,
  "indexer_lag_blocks": 2
}
```

---

## Rate limits

| Endpoint / recurso | Límite | Ventana |
|--------------------|--------|---------|
| `POST /agents/handshake` | 10 req | por IP / 1 min |
| `POST /agents/{id}/beacon` | 120 req | por agent_id / 1 min |
| WebSocket beacons | 120 msg | por agent_id / 1 min |
| `POST /tasks` | 60 req | por operador / 1 min |
| Global por IP | 300 req | por IP / 1 min |

Implementación: Redis counters `rate:{scope}:{id}`. Response 429 con `Retry-After`.

---

## Códigos de error JSON

```json
{
  "error": {
    "code": "SIGNATURE_INVALID",
    "message": "ECDSA signature verification failed",
    "request_id": "req-uuid"
  }
}
```

| `code` | HTTP |
|--------|------|
| `INVALID_STEP` | 400 |
| `NONCE_EXPIRED` | 400 |
| `SIGNATURE_INVALID` | 400 |
| `NONCE_REUSED` | 409 |
| `REPLAY_DETECTED` | 409 |
| `SESSION_INVALID` | 401 |
| `UNAUTHORIZED` | 401 |
| `AGENT_MISMATCH` | 403 |
| `AGENT_NOT_FOUND` | 404 |
| `RATE_LIMITED` | 429 |

---

## Versionado y compatibilidad

- `v` en envelope: incrementar solo si cambia formato de cifrado
- `/api/v2` si se cambian paths o auth; v1 soportada mínimo 6 meses en lab
- Agentes deben enviar `X-API-Version: 1` en handshake

## Referencias cruzadas

- Modelos: [04_data_models.md](./04_data_models.md)
- Seguridad: [05_security_specs.md](./05_security_specs.md)
- Arquitectura: [02_system_architecture.md](./02_system_architecture.md)
