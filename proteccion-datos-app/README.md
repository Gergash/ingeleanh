# Plataforma SaaS de Autodiagnóstico de Protección de Datos

Implementación inicial basada en:

- `SDD_Arquitectura_Sistema_y_Negocio.md` — dominio legal, accountability, UI/UX
- `TDD_Arquitectura_Tecnica_y_RAG.md` — stack, RAG MRHR, esquema PostgreSQL

## Stack

| Capa | Tecnología |
|------|------------|
| Frontend | Next.js 14, Tailwind CSS |
| API Gateway | Node.js + Express |
| AI / RAG | FastAPI, PostgreSQL full-text (pgvector listo) |
| Datos | PostgreSQL 16 + pgvector, Redis, MinIO |

## Arranque rápido

```bash
cd ingeleanh/proteccion-datos-app
cp .env.example .env
docker compose up --build
```

Servicios:

| URL | Descripción |
|-----|-------------|
| http://localhost:3000 | Frontend |
| http://localhost:4000 | API Gateway |
| http://localhost:8000/v1/health | AI Service |
| http://localhost:5432 | PostgreSQL |
| http://localhost:9001 | MinIO Console |

Demo: http://localhost:3000/app/empresa-demo/dashboard

## Estructura del monorepo

```
proteccion-datos-app/
├── db/migrations/          # Esquema + seed Ley 1581
├── ai_service/             # FastAPI — RAG y análisis
├── api_gateway/            # Proxy + endpoints dashboard
├── frontend/               # Next.js App Router
└── docker-compose.yml
```

## Fase actual (v0.1)

- Esquema core: tenants, usuarios, diagnósticos, remediación, corpus legal
- Seed: tenant `empresa-demo`, artículos 4/9/15 Ley 1581
- RAG MVP: búsqueda full-text con Filtering Wall multitenant
- Dashboard: score por dimensiones, tareas de remediación, panel RAG

## Próximos pasos (según TDD)

1. Embeddings pgvector + RRF híbrido denso/disperso
2. LangGraph — agentes legal, evidencia, gaps, citas
3. Wizard de diagnóstico multi-paso (SDD §2.3)
4. Kanban de remediación con evidencias (SDD §5.2)
5. Auth OAuth2 + RBAC + RLS PostgreSQL

## Desarrollo local sin Docker (solo AI)

```bash
# PostgreSQL debe estar corriendo (docker compose up postgres -d)
cd ai_service
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8000
```
