# SkillMaker — Mapa del proyecto

> Plataforma interna de formacion en video (LMS corporativo peer-to-peer). Cualquier trabajador puede publicar cursos (videos + material + evaluaciones) y consumir los que publican otros. Los administradores aprueban y supervisan.

## Stack

- **Frontend**: Angular 21 (standalone + zoneless) + PrimeNG + Tailwind 3
- **Backend**: Go 1.25 + Gin + GORM — monolito modular (7 modulos de dominio)
- **DB**: PostgreSQL 16 + golang-migrate
- **Auth**: Google OAuth 2.0 + JWT propio + refresh tokens con rotacion
- **Storage**: MinIO (dev) / S3-GCS (prod)
- **Docs API**: OpenAPI/Swagger generado desde anotaciones Go
- **Containers**: Docker + Compose

## Layout del monorepo

```
skillmaker/
├── backend/                       # API Go + Gin (monolito modular)
│   ├── cmd/
│   │   ├── api/                   # entry point del servidor HTTP
│   │   └── seed/                  # seed de roles iniciales
│   ├── internal/
│   │   ├── modules/               # modulos de dominio (auth, users, ...)
│   │   ├── platform/              # infra: config, database, httpserver, storage, logger, httperr
│   │   └── middleware/            # JWT, RBAC, CORS, logger
│   ├── migrations/                # SQL de golang-migrate
│   ├── docs/                      # Swagger generado (swag init)
│   ├── GUIA-BACK.md               # patrones del backend
│   ├── SMOKE-TESTS.md             # smoke tests manuales del backend
│   └── go.mod
│
├── frontend/                      # Angular 21
│   ├── src/app/
│   │   ├── core/                  # services, guards, interceptors, constants
│   │   ├── pages/                 # auth/, platform/ (rutas top-level)
│   │   ├── shared/                # components y directives reutilizables
│   │   └── sass/                  # tokens y estilos globales
│   ├── GUIA-FRONT.md              # patrones del frontend
│   ├── SMOKE-TESTS.md             # smoke tests manuales del frontend
│   └── package.json
│
├── bases/                         # Fuente de verdad funcional + roadmap SDD
│   ├── documentacion/             # .docx — RFs, RTs, modelo de datos (NO editar a mano)
│   └── SDD-ROADMAP.md             # plan incremental de SDD changes hacia el MVP
│
├── GUIA-MONOREPO.md               # estructura, Makefile, Docker, deployment
├── SETUP.md                       # pre-requisitos + Google OAuth + primera ejecucion
├── README.md                      # quick start
├── Makefile                       # comandos canonicos (install, db-*, dev)
├── docker-compose.yml             # Postgres + MinIO + servicios
└── .env.example
```

## Donde mirar segun la tarea

| Si necesitas... | Ve a... |
|-----------------|---------|
| Entender QUE hace el sistema (negocio) | `bases/documentacion/` |
| Ver el plan de implementacion incremental | `bases/SDD-ROADMAP.md` |
| Setup inicial / OAuth / .env | `SETUP.md` |
| Estructura del monorepo + Docker + Make | `GUIA-MONOREPO.md` |
| Convenciones de codigo Go | `backend/GUIA-BACK.md` |
| Convenciones de codigo Angular | `frontend/GUIA-FRONT.md` |
| Agregar feature de dominio en backend | `backend/internal/modules/<modulo>/` |
| Infra (DB, HTTP, storage, logger) | `backend/internal/platform/` |
| Migraciones SQL | `backend/migrations/` |
| Nueva pagina frontend | `frontend/src/app/pages/` |
| Servicio / guard / interceptor frontend | `frontend/src/app/core/` |
| Componente reutilizable | `frontend/src/app/shared/components/` |

## Estado actual (snapshot 2026-06-01)

- Scaffold backend + frontend **completos y compilando** (16 commits, ~2700 LOC).
- Modulos de dominio: `auth` ✅ completo, `users` 🟡 stub minimo, resto ❌ pendientes.
- Frontend: login Google + layout de plataforma operativos; paginas de catalog / my-courses / profile son stubs.
- Direccion estetica: **Cyanotype Atelier** (midnight ink + cyan + Instrument Serif + Geist).

Para el detalle actualizado revisar `git log --oneline` y `bases/SDD-ROADMAP.md`.

## URLs en desarrollo

| Servicio | URL |
|----------|-----|
| Frontend | http://localhost:4200 |
| Backend | http://localhost:3000 |
| Swagger | http://localhost:3000/api/docs/index.html |
| MinIO | http://localhost:9001 (`minioadmin` / `minioadmin`) |

## Workflow SDD

Este proyecto usa **Spec-Driven Development** con `engram` como artifact store. El roadmap en `bases/SDD-ROADMAP.md` lista los changes pendientes. Para arrancar uno nuevo: `/sdd-new <change-name>`.
