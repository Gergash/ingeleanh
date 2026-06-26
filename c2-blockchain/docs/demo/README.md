# Fase 3 — Demo y entrega (Hackathon Aligo)

Documentación operativa para **demo en vivo** y **video 3–7 min**.

## Estado del lab (26-06-2026)

| Ítem | Valor |
|------|--------|
| Contrato `C2Registry` (Polygon Amoy) | `0x629238eD79c23267fe502AAd81E5AEfee3908750` |
| Explorer | https://amoy.polygonscan.com/address/0x629238eD79c23267fe502AAd81E5AEfee3908750 |
| Config on-chain | `version=1`, `beacon_interval=30s` |
| Operador lab | `operator` / `lab` |
| Server lab | `http://localhost:8443` (`C2_INSECURE=true`) |

## Documentos

| Archivo | Uso |
|---------|-----|
| [GUION_DEMO_VIVO.md](./GUION_DEMO_VIVO.md) | Paso a paso ante jurado (15–20 min) |
| [GUION_VIDEO.md](./GUION_VIDEO.md) | Guion video 3–7 min con tiempos |
| [CHECKLIST_ENTREGABLES.md](./CHECKLIST_ENTREGABLES.md) | MIN-005: repo, docs, video, demo |
| [CLOUDFLARE_TUNNEL.md](./CLOUDFLARE_TUNNEL.md) | Publicar portal para jurados (HTTPS) |

## Portal operador (UI producto)

**URL lab:** http://localhost:8443/portal/

| Credencial | Valor |
|------------|--------|
| Usuario | `operator` |
| Contraseña | `lab` |

Con `C2_DEMO_MODE=true` se cargan **3 dispositivos IoT simulados** al arrancar el server.

```bash
docker start c2-redis || docker run -d --name c2-redis -p 6379:6379 redis:7-alpine
cd ingeleanh/c2-blockchain
go run ./cmd/server          # Terminal A
go run ./cmd/agent           # Terminal B (una sola instancia)
```

Guion detallado de pruebas: [README del proyecto](../README.md#prueba-de-lab--guion-completo-v010).

## Criterios DEMO cubiertos

| ID | Cubierto en demo | Notas |
|----|------------------|-------|
| DEMO-001 | ✅ | `/api/v1/health` |
| DEMO-002 | ✅ | Dashboard agentes `active` |
| DEMO-003 | ✅ | `whoami` → `stdout` |
| DEMO-004 | ✅ | Explicar envelope en beacon (ver guion) |
| DEMO-005 | ✅ | Polygonscan + `/chain/status` v1 |
| DEMO-006 | ⚠️ Stretch | Documentado; no obligatorio para mínimo |
| DEMO-007 | ⚠️ Opcional | IoT gateway + eventos simulados |
| DEMO-008 | ⚠️ Opcional | `iot_command` unlock |
| DEMO-009 | ✅ | Narrativa camuflaje + hash on-chain |
| DEMO-010 | ✅ | Dashboard agentes + chain footer |
| DEMO-011 | ⚠️ Stretch | IoT + chain en un flujo |

## Referencias SDD

- Criterios demo: [06_testing_strategy.md](../sdd/06_testing_strategy.md#criterios-de-demo)
- Fusión IoT: [07_iot_residential_fusion.md](../sdd/07_iot_residential_fusion.md)
