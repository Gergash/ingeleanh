# Checklist entregables — MIN-005 (cierre hackathon)

Marcar antes de enviar al jurado.

## 1. Repositorio (código)

- [ ] Rama `master` / `main` con Fase 2 implementada
- [ ] `go test ./...` verde localmente
- [ ] `cd contracts && npx hardhat test` verde
- [ ] Sin `.env` ni claves privadas commiteadas
- [ ] `package-lock.json` de contracts sincronizado (`npm ci` funciona)

## 2. Documentación

- [ ] SDD completo en `docs/sdd/` (Fase 1)
- [ ] README operativo con guion de lab
- [ ] Fase 3: `docs/demo/` (este folder)
- [ ] Contrato Amoy documentado con address y explorer

## 3. Video (3–7 min)

- [ ] Grabado según [GUION_VIDEO.md](./GUION_VIDEO.md)
- [ ] Muestra: server, agente, whoami, blockchain Amoy
- [ ] Sin secretos en pantalla
- [ ] Enlace agregado al README o entrega hackathon

## 4. Demo en vivo

- [ ] Guion [GUION_DEMO_VIVO.md](./GUION_DEMO_VIVO.md) ensayado
- [ ] Redis + server + agente probados el mismo día
- [ ] Token operador y un `agent_id` activo anotados

## MVPs Aligo (mínimo reto)

- [ ] MVP-01 Registro agente
- [ ] MVP-02 Beacon
- [ ] MVP-03 Comando remoto + resultado
- [ ] MVP-04 Lectura on-chain (`getConfig` v1)

## Stretch (Nivel Aligo)

- [ ] MVP-06 / MVP-08 IoT gateway + unlock
- [ ] MVP-05 Failover documentado
- [ ] DEMO-011 flujo 3 capas

## Enlaces útiles

| Recurso | URL |
|---------|-----|
| Contrato Amoy | https://amoy.polygonscan.com/address/0x629238eD79c23267fe502AAd81E5AEfee3908750 |
| Dashboard lab | http://localhost:8443/dashboard/ |
| SDD índice | [docs/sdd/README.md](../sdd/README.md) |
