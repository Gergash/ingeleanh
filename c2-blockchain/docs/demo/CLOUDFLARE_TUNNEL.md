# Publicar el portal para jurados (Cloudflare Tunnel)

Guía para que el equipo Aligo acceda al portal **sin instalar nada**, vía HTTPS público.

## Arquitectura

```
Jurado (browser) → https://c2-demo.tudominio.com/portal/
                 → Cloudflare Tunnel (cloudflared)
                 → tu PC: localhost:8443 (C2 server Go)
```

El server Go sirve el portal estático y la API en el mismo origen (sin CORS extra).

## Requisitos

1. Cuenta [Cloudflare](https://dash.cloudflare.com) (gratis).
2. Dominio en Cloudflare (o subdominio de un dominio que controles).
3. En tu PC: Redis + C2 server corriendo con `C2_DEMO_MODE=true`.

## 1. Preparar el server local

```bash
docker start c2-redis || docker run -d --name c2-redis -p 6379:6379 redis:7-alpine
cd ingeleanh/c2-blockchain
# .env debe incluir C2_DEMO_MODE=true y C2_INSECURE=true
go run ./cmd/server
```

Verifica: http://localhost:8443/portal/ → login `operator` / `lab`.

Opcional (datos IoT en vivo además del seed):

```bash
C2_IOT_GATEWAY=true go run ./cmd/agent
```

## 2. Instalar cloudflared

Windows (winget):

```bash
winget install Cloudflare.cloudflared
```

O descarga desde: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/

## 3. Autenticar cloudflared

```bash
cloudflared tunnel login
```

Abre el navegador y autoriza tu dominio en Cloudflare.

## 4. Crear tunnel

```bash
cloudflared tunnel create c2-hackathon-aligo
```

Anota el **Tunnel ID** que imprime.

## 5. DNS — subdominio para jurados

```bash
cloudflared tunnel route dns c2-hackathon-aligo c2-demo.tudominio.com
```

Reemplaza `tudominio.com` por tu dominio real.

## 6. Archivo de configuración

Crea `%USERPROFILE%\.cloudflared\config.yml` (Windows) o `~/.cloudflared/config.yml`:

```yaml
tunnel: <TUNNEL_ID>
credentials-file: C:\Users\<TU_USUARIO>\.cloudflared\<TUNNEL_ID>.json

ingress:
  - hostname: c2-demo.tudominio.com
    service: http://localhost:8443
  - service: http_status:404
```

## 7. Ejecutar tunnel

```bash
cloudflared tunnel run c2-hackathon-aligo
```

Mantén esta terminal abierta durante la evaluación.

## 8. Link para jurados

Entregar en la hackathon:

```
https://c2-demo.tudominio.com/portal/
Usuario: operator
Contraseña: lab
```

Incluir en README y video:

- Contrato Amoy: https://amoy.polygonscan.com/address/0x629238eD79c23267fe502AAd81E5AEfee3908750

## Qué verán los jurados

| Sección | Contenido |
|---------|-----------|
| Resumen | Agentes, dispositivos, eventos, chain v1 |
| 3 IoT | Sensor movimiento, medidor, cerradura (abrir/cerrar) |
| Blockchain | Barra con link a Polygonscan |
| C2 | Tabla agentes + botón whoami |
| Eventos | Feed simulado + live si hay gateway |

## Seguridad lab

- **No** commitear `.env` ni claves MetaMask.
- Credenciales `operator/lab` son solo para demo; cambiar si el link es público prolongado.
- Polygon Amoy es **testnet** (sin valor real).
- Apagar tunnel y server al terminar.

## Alternativa sin dominio propio

Cloudflare ofrece URLs temporales con **Quick Tunnel** (sin cuenta DNS):

```bash
cloudflared tunnel --url http://localhost:8443
```

Genera una URL `https://xxxx.trycloudflare.com` — útil para prueba rápida; la URL cambia cada vez.

## Troubleshooting

| Problema | Solución |
|----------|----------|
| 502 en portal | Server no corre en `:8443` |
| Login falla | Verificar `C2_OPERATOR_USER/PASS` en `.env` |
| Sin dispositivos IoT | `C2_DEMO_MODE=true` y reiniciar server, o botón "Recargar demo" |
| Chain v0 | Verificar `C2_REGISTRY_ADDRESS` y RPC Amoy |
| whoami pending | Iniciar `go run ./cmd/agent` en otra terminal |

## Referencias

- [Cloudflare Tunnel docs](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/)
- [Guion demo en vivo](./GUION_DEMO_VIVO.md)
- [Guion video](./GUION_VIDEO.md)
