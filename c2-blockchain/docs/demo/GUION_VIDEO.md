# Guion — Video demo (3–7 min)

Formato: grabación de pantalla + voz. Objetivo: jurado entiende **C2 propio + blockchain + flujo E2E** sin leer el SDD.

## Checklist pre-grabación

- [ ] Ventanas ordenadas: terminal, dashboard, Polygonscan
- [ ] Sin tokens/claves privadas visibles en pantalla
- [ ] Un solo agente corriendo
- [ ] Micrófono probado

---

## Estructura (≈ 5 min)

| Min | Sección | Pantalla | Qué decir |
|-----|---------|----------|-----------|
| 0:00–0:45 | **Hook** | Slide o terminal | "C2 Blockchain-Blindado: command and control propio para lab Aligo, con registry en Polygon Amoy. No es Metasploit; es ingeniería del equipo." |
| 0:45–1:15 | **Arquitectura** | Diagrama SDD o dibujo simple | Server Go + Redis + SQLite; agente Windows; contrato `C2Registry` en Amoy guarda config hasheada." |
| 1:15–1:45 | **DEMO-001** | `curl health` | Server OK, registry address visible. |
| 1:45–2:30 | **DEMO-002** | Dashboard agentes | Agente activo, beacon reciente, handshake completado. |
| 2:30–3:30 | **DEMO-005** | Dashboard footer + Polygonscan | Chain v1, contrato en Amoy, transacciones on-chain. URL del C2 no está en claro, solo hash. |
| 3:30–4:45 | **DEMO-003** | Terminal curl whoami | Operador crea tarea; resultado `completed` con stdout del host. |
| 4:45–5:15 | **DEMO-009** | Logs o JSON beacon | Payload cifrado; canal camuflado como API IoT. |
| 5:15–5:45 | **Cierre** | Repo / SDD | Tests verdes, SDD en `docs/sdd`, contrato desplegado. Invitación a demo en vivo. |

**Stretch (+2 min):** IoT unlock con `C2_IOT_GATEWAY=true` y comando `iot_command`.

---

## Texto guion (voz) — versión corta

> Hola, somos [equipo]. Presentamos C2 Blockchain-Blindado para el reto Aligo.
>
> Construimos un servidor C2 propio en Go: API REST, handshake ECDSA, beacons cifrados con AES-GCM y un agente para Windows.
>
> La innovación es el contrato C2Registry en Polygon Amoy testnet. No mandamos comandos por blockchain — eso sería lento y caro. Usamos la chain como registry de confianza: operadores autorizados y configuración de failover con la URL del servidor guardada como hash SHA-256.
>
> [Mostrar health y dashboard] El servidor está vivo. El agente se registró y beacon cada treinta segundos.
>
> [Mostrar Polygonscan] Aquí está nuestro contrato en Amoy. Versión uno de la configuración, intervalo de beacon treinta segundos.
>
> [Mostrar curl whoami] El operador envía una tarea remota. El agente ejecuta whoami y devuelve el resultado al operador. Flujo completo sin intervención manual interna.
>
> El tráfico parece una API normal; los payloads van cifrados. Documentación completa en docs/sdd, tests automatizados, y demo en vivo disponible.
>
> Gracias.

---

## Datos para overlay (texto en video)

```
C2Registry (Amoy): 0x629238eD79c23267fe502AAd81E5AEfee3908750
Chain: Polygon Amoy (80002)
Config version: 1 | Beacon: 30s
Repo: ingeleanh/c2-blockchain
```

---

## Publicación

- Duración objetivo: **5 min** (aceptable 3–7)
- Subir a plataforma indicada por hackathon (YouTube / Drive / Moodle)
- En README del repo: enlace al video cuando esté listo
