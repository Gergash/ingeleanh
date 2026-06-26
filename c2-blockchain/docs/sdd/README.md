# SDD — Sistema C2 Blockchain-Blindado

**Single Source of Truth (SSOT)** para el subproyecto `ingeleanh/c2-blockchain/`.

> **GATE DE APROBACIÓN**: No se escribe código fuente (`*.go`, `*.sol`, `package.json`, etc.) hasta que esta documentación esté revisada y aprobada explícitamente por el equipo. La Fase 2 (TDD) solo inicia tras marcar el checklist final de este archivo.

## Orden de lectura

| # | Documento | Descripción |
|---|-----------|-------------|
| 1 | [01_project_overview.md](./01_project_overview.md) | Objetivos, stack, restricciones, glosario |
| 2 | [02_system_architecture.md](./02_system_architecture.md) | Componentes, flujos, integración blockchain |
| 3 | [03_api_design.md](./03_api_design.md) | REST, WebSocket, payloads, encriptación |
| 4 | [04_data_models.md](./04_data_models.md) | SQLite, Redis, smart contract |
| 5 | [05_security_specs.md](./05_security_specs.md) | Criptografía, claves, modelo de amenazas |
| 6 | [06_testing_strategy.md](./06_testing_strategy.md) | TDD, casos de prueba, CI |
| 7 | [07_iot_residential_fusion.md](./07_iot_residential_fusion.md) | Fusión IoT Centro de Inteligencia Residencial + C2 |

### Dependencias entre documentos

```
01_overview → 02_architecture → 03_api + 04_models
01_overview → 05_security
01_overview → 07_iot_fusion
03_api + 04_models + 05_security + 07_iot_fusion → 06_testing
```

## Contexto del proyecto

- **Equipo**: Inge Lean — Hackathon Talento Tech
- **Empresa retadora**: Aligo Defensores Informáticos
- **Reto**: Construcción de Command and Control (C2)
- **Ubicación**: `ingeleanh/c2-blockchain/` (subproyecto aislado del landing estático)
- **Red blockchain**: Polygon Amoy testnet (sucesor de Mumbai, deprecada en 2024)
- **Uso**: Laboratorio de ciberseguridad autorizado / red team educativo
- **Nivel objetivo**: Nivel 4 — Aligo (arquitectura original, blockchain, IoT residencial, resiliencia, cifrado y demostración extremo a extremo)
- **Plataformas agente**: Linux y Windows (VMs lab autorizadas)
- **Innovación**: C2 propio + blockchain + camuflaje de canal + IoT (no réplica de frameworks conocidos)
- **Fusión aplicativa**: Centro de Inteligencia Residencial (IoT) + C2 como motor de seguridad — ver [07_iot_residential_fusion.md](./07_iot_residential_fusion.md)
- **Capas de plataforma**: Integración (Go/SQLite/Redis + API + Dashboard) → Seguridad (AES-GCM, ECDSA/ECDH, blockchain) → Pruebas y validación

## Criterios del reto Aligo

| Criterio | Peso | Evidencia esperada |
|----------|------|--------------------|
| Innovación técnica | 35% | Blockchain + camuflaje de canal + IoT gateway + C2 propio (sensores/cerraduras simulados) |
| Funcionalidad / que sirva | 25% | Servidor y agente funcionando, comando remoto ejecutado, resultado retornado |
| Robustez y diseño de arquitectura | 20% | Reconexión, múltiples agentes, modularidad, canal cifrado, anti-replay |
| Calidad de código y arquitectura | 10% | Código limpio, separación de responsabilidades, tests, README operativo |
| Presentación y documentación | 10% | SDD, video claro, demo en vivo y explicación de decisiones |

## Entregables oficiales del reto

| # | Entregable | Estado esperado al cierre |
|---|------------|---------------------------|
| 1 | Código fuente | Repositorio completo con servidor, agente, interfaz/API y README de ejecución |
| 2 | Documentación técnica | Arquitectura, protocolo de comunicación, cifrado, decisiones de diseño y limitaciones |
| 3 | Video | Grabación corta sugerida de 3 a 7 minutos mostrando el C2 en funcionamiento |
| 4 | Demostración en vivo | Operación del C2 ante jurado en entorno controlado |

## Mínimo para entrar a evaluación

- Servidor C2 acepta la conexión de al menos un agente.
- Agente se conecta al servidor y ejecuta comandos remotos enviados por el operador.
- El operador recibe resultados de los comandos.
- El canal de comunicación funciona de extremo a extremo durante la demostración.
- Los cuatro entregables oficiales están disponibles al cierre.

## Checklist de aceptación — Fase SDD

Marcar cada ítem antes de iniciar Fase 2 (desarrollo TDD):

- [ ] Los 7 documentos SDD existen con contenido completo (sin placeholders)
- [ ] Fusión IoT residencial documentada en `07_iot_residential_fusion.md`
- [ ] Retroalimentación retadores (gateway, sensores simulados, cerraduras, camuflaje) en `01`, `05`, `07`
- [ ] El contexto del reto Aligo está reflejado en `01_project_overview.md`
- [ ] Los criterios de evaluación están trazados contra capacidades del MVP
- [ ] Stack tecnológico y red Polygon Amoy documentados (con nota Mumbai→Amoy)
- [ ] Handshake y beaconing descritos con secuencias y payloads JSON
- [ ] Smart contract `C2Registry` especificado (structs, funciones, eventos)
- [ ] Modelo de amenazas con mitigaciones para MITM, replay y spoofing
- [ ] ≥15 casos de prueba identificados con IDs en `06_testing_strategy.md`
- [ ] Tres capas (plataforma, seguridad, validación) documentadas en `02` y `06`
- [ ] Dashboard de estado definido (vistas, endpoints) en `02_system_architecture.md`
- [ ] Escenarios de validación por capas con flujos end-to-end en `06_testing_strategy.md`
- [ ] No existe código fuente fuera de `docs/` en `c2-blockchain/`
- [ ] Equipo aprueba explícitamente el SDD

## Aprobación

| Rol | Nombre | Fecha | Aprobado |
|-----|--------|-------|----------|
| Tech Lead | | | ☐ |
| Security Lead | | | ☐ |
| Product / Hackathon | | | ☐ |

**Firma de gate**: Al marcar las checkboxes y completar la tabla, el equipo autoriza la Fase 2 (TDD).

## Fase 2 — Referencia (no iniciar sin gate)

Orden de desarrollo previsto tras aprobación:

1. `contracts/` + Hardhat tests
2. `internal/crypto` + tests
3. `internal/handshake` + tests
4. C2 server API/WebSocket + integration tests
5. Agent beacon loop + e2e
6. Gateway IoT residencial + eventos simulados (ver `07_iot_residential_fusion.md`)
7. Dashboard de estado (HTML/JS estático, endpoints nuevos)
8. Escenarios de validación por capas (E2E-INTEG-001)
