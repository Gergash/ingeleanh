# 01 — Project Overview

## Visión

**Sistema C2 Blockchain-Blindado** es una plataforma de Command and Control (C2) diseñada para laboratorios de ciberseguridad y ejercicios de red team autorizados. Integra un servidor C2 tradicional con un **registro inmutable en blockchain** (Polygon Amoy testnet) que almacena configuración de descubrimiento, claves de operadores y metadatos de resiliencia.

La blockchain **no transporta comandos en claro**. Actúa como capa de **verificación y failover**: si el endpoint C2 primario cae o es comprometido, el agente puede leer configuración actualizada desde el contrato `C2Registry`, firmada por operadores autorizados on-chain.

## Contexto del reto

Este subproyecto responde al **Reto de Construcción de Command and Control (C2)** de la hackathon de desarrollo organizada por **Aligo Defensores Informáticos**. El reto busca que los equipos diseñen y construyan una infraestructura que permita a un servidor central coordinar uno o varios agentes para ejecutar comandos remotos, recibir resultados y orquestar la operación dentro de un **laboratorio cerrado y autorizado**.

El reto no exige un C2 ofensivo real contra terceros. Evalúa dominio de ingeniería, redes, criptografía, arquitectura distribuida, diseño de protocolos, resiliencia, claridad de documentación y capacidad de demostrar el sistema funcionando de extremo a extremo.

### Plataformas objetivo (agente / host comprometido en lab)

El reto permite desplegar el agente en **Linux o Windows** (VMs del laboratorio autorizado). El servidor C2 y la infraestructura de coordinación corren preferentemente en Linux (Docker); los agentes se compilan para el SO del host objetivo.

| Componente | SO soportado (lab) |
|------------|-------------------|
| C2 Server, Redis, indexer | Linux (Docker Compose) |
| Agente / implant | **Linux** (`linux-amd64`, `linux-arm64`) y **Windows** (`windows-amd64`) |
| Operador | Cualquier SO con cliente HTTP (curl, Postman, script) |

### Política: desarrollo propio vs frameworks existentes

Confirmación del reto: se **puede** usar frameworks ofensivos (ej. **Metasploit**) y herramientas del ecosistema, pero **no sustituyen** el C2 completo. Usar solo Metasploit (u otro C2/framework ya hecho) como solución entera **no cumple** el espíritu ni la evaluación del reto.

| Categoría | ¿Permitido? | Rol en este proyecto |
|-----------|-------------|----------------------|
| **Metasploit / frameworks similares** | Sí, como complemento | Módulos auxiliares en lab (ej. validar exploit, tarea `msf` opcional); **no** como servidor C2 principal |
| **C2 Server propio** | Obligatorio (desarrollo del equipo) | Go: API REST, WebSocket hub, task queue, operador |
| **Protocolo propio** | Obligatorio | Handshake ECDSA, beaconing, envelopes AES-GCM |
| **Agente propio** | Obligatorio | Go: beacon loop, executor, chain client |
| **Blockchain / registry** | Desarrollo del equipo | `C2Registry` + indexer (diferenciador Nivel Aligo) |
| Librerías estándar | Sí | `go-ethereum`, Redis, SQLite, Hardhat, TLS |

**Principio**: el jurado debe ver **ingeniería del equipo** en servidor, canal de control, agente y orquestación. Metasploit u otros frameworks pueden integrarse en **tareas específicas** o pruebas de lab, pero el **núcleo C2 es propio**.

### Retroalimentación del reto — innovación y realismo

Confirmaciones y recomendaciones de los retadores (Aligo):

| Tema | Confirmación / recomendación | Respuesta técnica del proyecto |
|------|------------------------------|--------------------------------|
| **Gateway IoT** | Modelar gateway residencial | `agent_role=iot_gateway`; un gateway por hogar/bloque |
| **Sensores** | Se pueden **simular** (no requiere hardware físico) | Scripts/VM que generan `iot_event` realistas (movimiento, zona, timestamp) |
| **Cerraduras** | **Sí** se pueden enviar comandos (lock/unlock) | `iot_command` con `action: lock\|unlock\|status` hacia `smart_lock` simulado |
| **Innovación** | Evitar “cosas ya hechas”; buscan propuestas originales | C2 propio + blockchain como registry + fusión residencial + camuflaje de canal |
| **No ser detectados (lab)** | Operación lo más **real** posible | Camuflaje de tráfico, metadata en chain, cifrado, jitter de beacon — ver [05_security_specs.md](./05_security_specs.md) |
| **Blockchain** | Utilizar blockchain como parte diferenciadora | `C2Registry`: identidades, config, failover sin exponer URLs en claro |

**Narrativa de innovación**: no es un C2 genérico ni un fork de herramienta existente. Es un **canal de control camuflado** (parece API IoT/cloud) con **coordinación verificable on-chain** y **escenario residencial** demostrable ante jurado.

### Mínimo esperado por el reto

| Requisito mínimo | Cobertura propuesta |
|------------------|---------------------|
| Servidor C2 que acepte conexión de al menos un agente | `C2 Server` en Go con REST/WebSocket |
| Agente conectado al servidor que ejecute comandos remotos | `Agent` en Go con beacon loop y executor controlado |
| Retorno de resultados al operador | `task_result` cifrado y persistido en SQLite |
| Canal funcionando de extremo a extremo durante la demostración | Handshake + WebSocket beacon + tarea `whoami` |
| Entregables completos: código, documentación, video y demo | Fase 1 SDD, Fase 2 TDD, Fase 3 demo/video |

### Nivel de ambición

La propuesta apunta al **Nivel 4 — Aligo** de la curva de complejidad: además de cumplir el C2 funcional, incorpora una arquitectura original con blockchain, mecanismos de coordinación verificables, cifrado robusto, failover, auditoría y diseño modular.

## Fusión con Centro de Inteligencia Residencial (IoT)

El SDD no solo cubre un C2 de laboratorio abstracto. Se fusiona con el **Centro de Inteligencia Residencial** como capa de aplicación:

| Capa | Responsabilidad |
|------|-----------------|
| Residencial (PDF / contexto) | Usuarios, comunidad, sensores, cámaras, cerraduras, medidores, gateways |
| C2 SDD | Cifrado E2E (AES-256-GCM, ECDH), handshake, beaconing, operador |
| Blockchain | Registry de identidades, config verificable, failover en Polygon Amoy |

Cada **gateway residencial** actúa como **agente C2**; dispositivos IoT reportan eventos y telemetría a través del gateway con envelopes cifrados. Detalle completo: [07_iot_residential_fusion.md](./07_iot_residential_fusion.md).

## Objetivos del MVP (Hackathon)

| ID | Objetivo | Criterio de éxito |
|----|----------|-------------------|
| MVP-01 | Registro de agente | 1 agente completa handshake ECDSA y queda registrado en SQLite |
| MVP-02 | Beacon periódico | Agente envía beacon cada `N` segundos (configurable, default 30s) vía WebSocket |
| MVP-03 | Comando remoto | Operador crea tarea; agente ejecuta `whoami` y devuelve resultado cifrado |
| MVP-04 | Lectura on-chain | Agente o servidor lee `getConfig()` del contrato y valida endpoint/beacon interval |
| MVP-05 | Failover documentado | Simulación de caída del servidor primario; agente usa config on-chain para reconectar |
| MVP-06 | Gateway IoT residencial | Gateway modelado; sensores **simulados** envían `iot_event` cifrado |
| MVP-07 | Agente multi-OS | Linux + Windows en lab VM |
| MVP-08 | Comando a cerradura | Operador envía `iot_command` lock/unlock; gateway simula respuesta de `smart_lock` |
| MVP-09 | Camuflaje de canal | Tráfico beacon/API indistinguible de API IoT legítima; config crítica vía blockchain |

## Alineación con criterios de evaluación

| Criterio Aligo | Peso | Cómo lo cubre el proyecto |
|----------------|------|---------------------------|
| Innovación técnica | 35% | Blockchain + IoT gateway + camuflaje de canal + C2 propio (no réplica de frameworks conocidos) |
| Funcionalidad / que sirva | 25% | Flujo extremo a extremo: servidor levanta, agente conecta, ejecuta comando y retorna resultado |
| Robustez y diseño de arquitectura | 20% | Cifrado, anti-replay, Redis queue, SQLite audit log, reconexión y separación por módulos |
| Calidad de código y arquitectura | 10% | C2 core propio (no Metasploit-as-C2), TDD, Go modular, contrato Hardhat, CI |
| Presentación y documentación | 10% | SDD completo, diagramas, estrategia de pruebas, guion de demo y trazabilidad contra el reto |

## Non-goals (fuera de alcance)

- Usar Metasploit, Cobalt Strike, Sliver u otro C2 completo **como sustituto** del servidor/agente/protocolo propios
- Presentar un proyecto donde el “C2” sea únicamente configuración de herramienta existente sin desarrollo propio
- Despliegue en mainnet o redes de producción con valor real
- Botnet masiva o escalado a miles de implantes
- **Obfuscación de binarios** para evadir antivirus (fuera de alcance MVP)
- Uso contra sistemas de terceros sin autorización explícita
- Transporte de payloads de comando vía transacciones blockchain (costo y latencia prohibidos)
- Dashboard web completo (MVP: API + logs; UI opcional fase posterior)

## Stack tecnológico

| Capa | Tecnología | Versión / Notas |
|------|------------|-----------------|
| **C2 Server** | Go | 1.22+ |
| **HTTP router** | `go-chi/chi` o `net/http` std | REST + middleware |
| **WebSocket** | `gorilla/websocket` o `nhooyr.io/websocket` | Beacon en tiempo real |
| **Agente** | Go | `cmd/agent`; cross-compile `linux/amd64`, `linux/arm64`, `windows/amd64` |
| **Integración lab (opcional)** | Metasploit Framework | Solo módulos/tareas auxiliares; no reemplaza C2 Server ni protocolo |
| **Base de datos** | SQLite | Registro persistente (agentes, tareas, audit) |
| **Cache / queue** | Redis | Sesiones, beacon queue, rate limit, pub/sub |
| **Smart contracts** | Solidity | 0.8.20+ |
| **Toolchain chain** | Hardhat | Deploy, tests, scripts |
| **Cliente chain (Go)** | `go-ethereum` | Lectura de contrato y eventos |
| **Cliente chain (scripts)** | `ethers.js` v6 | Deploy y scripts Hardhat |
| **Criptografía** | std `crypto/` Go | AES-256-GCM, ECDSA secp256k1, ECDH P-256, HKDF-SHA256 |
| **TLS** | TLS 1.3 | Obligatorio en producción/lab |
| **Contenedores** | Docker Compose | Server + Redis (+ Hardhat node opcional local) |

## Blockchain — Polygon Amoy testnet

### Red seleccionada

| Parámetro | Valor |
|-----------|-------|
| Red | Polygon Amoy (testnet) |
| Chain ID | `80002` |
| RPC público (ejemplo) | `https://rpc-amoy.polygon.technology` |
| Explorer | `https://amoy.polygonscan.com` |
| Faucet | Polygon faucet (Amoy) |

### Migración Mumbai → Amoy

Polygon **Mumbai** fue deprecada y descontinuada (2024). En narrativa del hackathon puede aparecer "Mumbai" como referencia histórica; **toda implementación y documentación técnica apunta a Amoy**.

| Mumbai (deprecada) | Amoy (activa) |
|--------------------|---------------|
| Chain ID `80001` | Chain ID `80002` |
| RPC Mumbai | RPC Amoy |
| Contratos Mumbai | Re-deploy en Amoy |

Variables de entorno documentadas:

```bash
CHAIN_ID=80002
POLYGON_RPC_URL=https://rpc-amoy.polygon.technology
C2_REGISTRY_ADDRESS=<deployed_contract>
OPERATOR_PRIVATE_KEY=<never_commit>
```

## Restricciones de seguridad y legales

### Uso autorizado

Este sistema está diseñado **únicamente** para:

- Laboratorios aislados (VMs propias, redes virtuales sin salida a internet pública no controlada)
- Ejercicios de red team con **consentimiento documentado** de todos los participantes
- Investigación educativa en contexto de hackathon / bootcamp

### Prohibiciones

- Instalar el agente en sistemas sin autorización explícita
- Desplegar en infraestructura de terceros
- Commitear claves privadas, seeds o credenciales al repositorio
- Usar mainnet Polygon u otras redes con activos reales

### Requisitos operativos

- Red lab aislada o VLAN dedicada
- Logs de auditoría habilitados (`audit_log` en SQLite)
- Rotación de claves de operador tras cada ejercicio
- Destrucción de datos de lab al finalizar el hackathon

## Entregables por fase

| Fase | Entregable | Estado |
|------|------------|--------|
| **Fase 1 — SDD** | 6 documentos Markdown en `docs/sdd/` alineados al reto Aligo | En curso |
| **Fase 2 — TDD** | Código + tests (Red → Verde → Refactor) | Pendiente gate |
| **Fase 3 — Demo** | Contrato en Amoy + server + agent en lab VM + video 3-7 min + demo en vivo | Pendiente |

## Glosario

| Termino | Definición |
|---------|------------|
| **C2** | Command and Control — infraestructura que controla sistemas comprometidos |
| **Agente / Implant** | Software en el host objetivo que beacon y ejecuta tareas |
| **Operador** | Usuario autorizado que envía comandos vía API/dashboard |
| **Beacon / Beaconing** | Heartbeat periódico del agente al servidor C2 |
| **Handshake** | Protocolo inicial de autenticación y establecimiento de sesión |
| **Registry** | Contrato `C2Registry` on-chain con config y operadores |
| **Envelope cifrado** | Payload JSON envuelto en AES-256-GCM (iv, ct, tag) |
| **Chain indexer** | Componente del servidor que observa eventos del contrato |
| **Failover** | Reconexión del agente usando config leída desde blockchain |
| **Gateway IoT** | Agente C2 que agrega dispositivos residenciales (sensores, medidores, cerraduras) |
| **Telemetría IoT** | Lecturas periódicas cifradas (energía, agua) vía beacon |
| **C2 propio** | Servidor, protocolo y agente desarrollados por el equipo; no un framework C2 empaquetado |
| **Framework auxiliar** | Herramienta externa (ej. Metasploit) usada solo en tareas puntuales del lab |
| **Camuflaje de canal** | Tráfico C2 disimulado como API/IoT legítima; payloads cifrados; metadata sensible on-chain |
| **Sensor simulado** | Script o proceso en VM que emula lecturas/eventos sin hardware físico |
| **Cerradura simulada** | Estado lock/unlock en gateway; comando remoto con auditoría |

## Referencias cruzadas

- Fusión IoT residencial: [07_iot_residential_fusion.md](./07_iot_residential_fusion.md)
- Arquitectura: [02_system_architecture.md](./02_system_architecture.md)
- API: [03_api_design.md](./03_api_design.md)
- Modelos: [04_data_models.md](./04_data_models.md)
- Seguridad: [05_security_specs.md](./05_security_specs.md)
- Testing: [06_testing_strategy.md](./06_testing_strategy.md)
