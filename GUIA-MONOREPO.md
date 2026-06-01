# Guia del Monorepo — Arquitectura, Infraestructura y Deployment

Documentacion de la arquitectura del monorepo, su estructura polyglot (Go + Angular), estrategia Docker, comunicacion entre servicios, y workflow de desarrollo y deployment. Sirve como referencia para replicar este patron en futuros proyectos full-stack basados en Go + Angular.

> **Este documento es el "plano maestro" del monorepo.** Para patrones internos de cada workspace:
> - **Frontend (Angular):** ver [`frontend/GUIA-FRONT.md`](frontend/GUIA-FRONT.md)
> - **Backend (Go + Gin):** ver [`backend/GUIA-BACK.md`](backend/GUIA-BACK.md)
> - **Setup inicial:** ver [`SETUP.md`](SETUP.md)
> - **Requerimientos funcionales y tecnicos:** ver [`bases/documentacion/`](bases/documentacion/)

---

## Tabla de Contenidos

1. [Introduccion y Proposito](#1-introduccion-y-proposito)
2. [Vision General de la Arquitectura](#2-vision-general-de-la-arquitectura)
3. [Estructura de Directorios](#3-estructura-de-directorios)
4. [Orquestacion del Monorepo (Makefile)](#4-orquestacion-del-monorepo-makefile)
5. [Gestion de Dependencias](#5-gestion-de-dependencias)
6. [Variables de Entorno y Configuracion](#6-variables-de-entorno-y-configuracion)
7. [Estrategia Docker](#7-estrategia-docker)
8. [Comunicacion entre Servicios](#8-comunicacion-entre-servicios)
9. [Workflow de Desarrollo](#9-workflow-de-desarrollo)
10. [Deployment a Produccion](#10-deployment-a-produccion)
11. [Testing](#11-testing)
12. [Linting y Formateo](#12-linting-y-formateo)
13. [Agregar Nuevas Features End-to-End](#13-agregar-nuevas-features-end-to-end)
14. [Directorios Auxiliares](#14-directorios-auxiliares)
15. [Referencias Cruzadas](#15-referencias-cruzadas)

---

## 1. Introduccion y Proposito

Este documento describe **como esta construido el monorepo** y como replicar este patron para nuevos proyectos full-stack en **Go + Angular**. No cubre los patrones internos de cada workspace — eso lo hacen las guias especificas.

El monorepo es **polyglot**: convive una toolchain Go (modulo unico) con una toolchain Node/npm (Angular). Por eso NO se usa `npm workspaces` — cada workspace gestiona su toolchain de forma independiente, y un **Makefile** en la raiz orquesta todos los comandos.

### Sistema de 3 Guias

| Guia | Alcance | Analogia |
|------|---------|----------|
| **GUIA-MONOREPO.md** (este archivo) | Estructura del monorepo, Makefile, Docker, deployment, comunicacion entre servicios | La estructura de la casa: cimientos, paredes, plomeria |
| **GUIA-FRONT.md** | Patrones internos del frontend Angular | Como amueblar el living |
| **GUIA-BACK.md** | Patrones internos del backend Go + Gin (monolito modular) | Como amueblar la cocina |

**Lectura recomendada:** Este documento primero (entender la casa), luego la guia del workspace donde vayas a trabajar.

### Fuente de verdad del dominio

Los requerimientos funcionales y tecnicos viven en [`bases/documentacion/`](bases/documentacion/):
- `Requerimientos_Plataforma_Formacion.docx` — RFs y RNFs del producto
- `Requerimientos_Tecnicos_Plataforma_Formacion.docx` — RTs (Angular + Go + Gin + GORM + PostgreSQL)
- `Modelo_Datos_Plataforma_Formacion.docx` — diccionario de datos y DDL de referencia

Cualquier decision arquitectonica documentada en esta guia debe ser trazable a un RT o RNF de esa documentacion.

---

## 2. Vision General de la Arquitectura

### Stack Tecnologico

| Capa | Tecnologia | Version | Rol |
|------|-----------|---------|-----|
| Frontend | Angular | 21 | SPA (Standalone, Zoneless) |
| UI | PrimeNG + TailwindCSS | 21 / 4 | Componentes + utilidades CSS |
| Backend | Go + Gin | 1.23 / 1.10 | API REST (monolito modular) |
| ORM | GORM | 1.25+ | Acceso a PostgreSQL |
| Migraciones | golang-migrate | v4 | Migraciones reproducibles (RT-18) |
| Base de datos | PostgreSQL | 16 | Persistencia |
| Auth | Google OAuth 2.0 + JWT | - | Login corporativo + sesion propia |
| Object Storage | MinIO (dev) / S3-GCS (prod) | - | Material adjunto de cursos |
| Hot reload backend | Air | v1.52+ | Recompilacion automatica en dev |
| Contenedores | Docker + Compose | - | Desarrollo y produccion |
| API Docs | swaggo/swag | v1.16+ | Genera OpenAPI desde anotaciones Go |
| Testing backend | go test + testify + testcontainers-go | - | Unit + integracion contra Postgres real |
| Testing frontend | Vitest | 2 | Testing unitario de Angular |
| Orquestacion | Makefile | - | Comandos del monorepo |

### Diagrama de Alto Nivel

```
┌──────────────────────────────────────────────────────────────────┐
│                    MONOREPO polyglot (Makefile)                   │
│                                                                  │
│  ┌──────────────────┐         ┌────────────────────────────────┐ │
│  │    frontend/     │         │           backend/             │ │
│  │  (Node/npm)      │         │           (Go module)          │ │
│  │                  │         │                                │ │
│  │  Angular 21      │  HTTPS  │  Go 1.23 + Gin 1.10           │ │
│  │  PrimeNG 21      │ ──────> │  GORM 1.25                     │ │
│  │  TailwindCSS 4   │  /api/* │  Google OAuth 2.0 + JWT        │ │
│  │                  │         │  Monolito modular (7 modulos)  │ │
│  │  Dev: :4200      │         │                                │ │
│  │  Prod: nginx:80  │         │  Dev: :3000 (Air)              │ │
│  └──────────────────┘         │  Prod: distroless:3000         │ │
│                               └────────┬────────────────┬──────┘ │
│                                        │                │        │
│                              ┌─────────▼─────────┐  ┌───▼─────┐  │
│                              │  PostgreSQL 16    │  │ MinIO   │  │
│                              │  Dev: :5432       │  │ Dev:9000│  │
│                              │  Prod: internal   │  │ S3 prod │  │
│                              └───────────────────┘  └─────────┘  │
│                                                                  │
│  Makefile (root) ── targets para backend + frontend + docker     │
│  docker-compose.yml ── orquestacion produccion (4 servicios)     │
└──────────────────────────────────────────────────────────────────┘
```

### Modelo C4 — Resumen

La documentacion base define 3 niveles del modelo C4:

- **Nivel 1 (Contexto):** trabajadores (alumnos/creadores), administradores y supervisores interactuan con el sistema. Dependencias externas: Google Workspace (auth) y YouTube/Vimeo (videos referenciados, no almacenados).
- **Nivel 2 (Contenedores):** SPA Angular + backend Go + PostgreSQL + Object Storage. La SPA embebe videos desde YouTube/Vimeo directo en el navegador.
- **Nivel 3 (Componentes del backend):** router con middleware JWT y RBAC delegando en los 7 modulos de dominio. La capa `platform` concentra acceso a Postgres y object storage. Cada modulo es propietario de sus tablas y se comunica con otros solo via interfaces.

Esta separacion permite extraer un modulo como microservicio sin reescribir su logica de dominio (RNFT-02).

---

## 3. Estructura de Directorios

```
skillmaker/
├── backend/                          # Workspace: Go monolito modular
│   ├── cmd/
│   │   └── api/
│   │       └── main.go               # Entry point — wiring + bootstrap del server
│   ├── internal/
│   │   ├── modules/                  # Modulos de dominio (frontera de propiedad)
│   │   │   ├── auth/                 # Google OAuth + emision de JWT
│   │   │   ├── users/                # Usuarios, roles, supervision
│   │   │   ├── courses/              # Cursos, secciones, videos, material, inscripcion
│   │   │   ├── evaluations/          # Evaluaciones, preguntas, intentos, respuestas
│   │   │   ├── approvals/            # Revision/aprobacion de cursos
│   │   │   ├── certificates/         # Certificados, badges, ranking
│   │   │   └── reporting/            # Reportes globales y de supervision
│   │   ├── platform/                 # Infra compartida (no es un modulo de dominio)
│   │   │   ├── config/               # Carga de variables de entorno
│   │   │   ├── database/             # Conexion GORM + healthcheck
│   │   │   ├── storage/              # Cliente S3/MinIO + URLs prefirmadas
│   │   │   ├── httpserver/           # Setup Gin + middlewares globales
│   │   │   └── logger/               # Logger estructurado (zap o slog)
│   │   ├── middleware/               # JWT, RBAC, CORS, request-id, recovery
│   │   └── router/                   # Registro de rutas por modulo
│   ├── migrations/                   # SQL versionado para golang-migrate
│   │   ├── 0001_init.up.sql
│   │   └── 0001_init.down.sql
│   ├── docs/                         # OpenAPI generado por swaggo/swag
│   ├── tests/                        # Tests de integracion compartidos (testcontainers)
│   ├── .air.toml                     # Config de hot reload (Air)
│   ├── .env                          # Variables de entorno (gitignored)
│   ├── .env.example                  # Template de variables
│   ├── .golangci.yml                 # Config del linter
│   ├── Dockerfile                    # Build de produccion (multi-stage Go + distroless)
│   ├── docker-compose.dev.yml        # PostgreSQL + MinIO para desarrollo local
│   ├── docker-compose.yml            # Backend + PostgreSQL + MinIO (testing local del container)
│   ├── go.mod                        # Modulo Go raiz del backend
│   ├── go.sum                        # Lockfile de dependencias Go
│   ├── Makefile                      # Targets locales del backend (opcional)
│   └── GUIA-BACK.md     # Guia de patrones del backend
│
├── frontend/                         # Workspace: Angular SPA
│   ├── src/
│   │   ├── app/
│   │   │   ├── core/                 # Services, guards, interceptors (auth, JWT, http)
│   │   │   ├── pages/                # Feature modules (lazy loaded)
│   │   │   └── shared/               # Componentes reutilizables
│   │   └── environments/             # environment.ts / environment.prod.ts
│   ├── Dockerfile                    # Build de produccion (multi-stage Angular + nginx)
│   ├── nginx.conf                    # Reverse proxy + SPA fallback
│   ├── angular.json                  # Config de Angular CLI
│   ├── tsconfig.json                 # TypeScript config (con path aliases)
│   ├── package.json                  # Dependencias npm del frontend
│   ├── package-lock.json             # Lockfile npm (solo del frontend)
│   └── GUIA-FRONT.md         # Guia de patrones del frontend
│
├── bases/                            # Documentacion funcional/tecnica del dominio
│   └── documentacion/                # Fuente de verdad (RFs, RTs, modelo de datos)
│
├── docker-compose.yml                # Orquestacion produccion (4 servicios)
├── .env                              # Variables de produccion (gitignored)
├── .env.example                      # Template de produccion
├── Makefile                          # Orquestador del monorepo (raiz)
├── CLAUDE.md                         # Contexto para agentes IA
├── SETUP.md                          # Guia de instalacion rapida
└── GUIA-MONOREPO.md                  # Este documento
```

**Puntos clave:**
- **Polyglot:** el backend tiene su propio `go.mod` y el frontend su propio `package.json` con `package-lock.json` independiente. No hay lockfile compartido.
- **Boundary de modulos:** cada modulo bajo `backend/internal/modules/` es propietario de sus tablas. No hay joins SQL cross-modulo — el acceso pasa por la interfaz del modulo dueño (RT-19).
- **`cmd/` vs `internal/`:** convencion estandar de Go. `cmd/api/main.go` es solo wiring (composition root); toda la logica vive en `internal/`. `internal/` impide que paquetes externos importen el codigo del proyecto.
- **Migraciones SQL versionadas:** golang-migrate exige pares `up.sql` / `down.sql` numerados. GORM se usa para queries y mapeo, no para definir el schema en produccion (AutoMigrate solo se permite en dev opcionalmente).
- **Object storage local con MinIO:** evita depender de credenciales AWS/GCS durante desarrollo (RT-21 acepta S3, GCS o MinIO).
- **`docker-compose.dev.yml` vive en `backend/`** para que el desarrollador frontend pueda levantar la dependencia que necesita sin enredarse con la del backend.

---

## 4. Orquestacion del Monorepo (Makefile)

A diferencia de un monorepo Node puro (que usa npm workspaces), este monorepo es **polyglot**: Go y npm coexisten. Por eso se usa un **Makefile** en la raiz como orquestador unico de comandos.

### 4.1 Por que Makefile y no npm workspaces

| Razon | Detalle |
|-------|---------|
| Polyglot | `npm workspaces` solo entiende `package.json`; no puede orquestar `go build`, `go test`, `air`, `golang-migrate`, etc. |
| Familiaridad | `make <target>` es el patron estandar en proyectos Go open source. |
| Cero dependencias extra | `make` viene preinstalado en Linux/macOS. En Windows: usar WSL o Git Bash. |
| Discoverabilidad | `make help` lista todos los targets disponibles con descripcion. |

**Alternativas validas si el equipo lo prefiere:** [Task](https://taskfile.dev) (YAML, multiplataforma sin WSL) o [justfile](https://github.com/casey/just). El patron descrito aqui se traduce 1:1.

### 4.2 Makefile de Referencia (raiz)

```makefile
# Makefile — orquestador del monorepo Angular + Go
.DEFAULT_GOAL := help
SHELL := /bin/bash

## ---------- Help ----------
.PHONY: help
help: ## Muestra esta ayuda
	@grep -E '^[a-zA-Z_:-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2}'

## ---------- Setup ----------
.PHONY: install
install: backend-install frontend-install ## Instala dependencias de ambos workspaces

.PHONY: backend-install
backend-install: ## Descarga dependencias Go
	cd backend && go mod download

.PHONY: frontend-install
frontend-install: ## Instala dependencias npm del frontend
	cd frontend && npm install --legacy-peer-deps

## ---------- Dev ----------
.PHONY: dev
dev: ## Levanta backend (Air) + frontend (ng serve) en paralelo
	@command -v concurrently >/dev/null || npm i -g concurrently
	concurrently --names "BACKEND,FRONTEND" --prefix-colors "cyan,magenta" \
	  "make backend-dev" "make frontend-dev"

.PHONY: backend-dev
backend-dev: ## Backend con hot reload (Air)
	cd backend && air

.PHONY: frontend-dev
frontend-dev: ## Frontend en modo desarrollo (ng serve :4200)
	cd frontend && npm run start

## ---------- Build ----------
.PHONY: build
build: backend-build frontend-build ## Compila ambos workspaces

.PHONY: backend-build
backend-build: ## Compila el binario de produccion
	cd backend && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o dist/api ./cmd/api

.PHONY: frontend-build
frontend-build: ## Build de produccion del frontend
	cd frontend && npm run build:prod

## ---------- Tests ----------
.PHONY: test
test: backend-test frontend-test ## Ejecuta tests de ambos workspaces

.PHONY: backend-test
backend-test: ## Tests del backend con cobertura
	cd backend && go test ./... -race -coverprofile=coverage.out

.PHONY: backend-test-integration
backend-test-integration: ## Tests de integracion (testcontainers, mas lentos)
	cd backend && go test ./... -tags=integration

.PHONY: frontend-test
frontend-test: ## Tests del frontend (Vitest)
	cd frontend && npm run test

## ---------- Lint ----------
.PHONY: lint
lint: backend-lint frontend-lint ## Lintea ambos workspaces

.PHONY: backend-lint
backend-lint: ## golangci-lint sobre todo el backend
	cd backend && golangci-lint run

.PHONY: frontend-lint
frontend-lint: ## Angular ESLint
	cd frontend && npm run lint

## ---------- DB / Migraciones ----------
DB_URL ?= postgres://skillmaker:skillmaker@localhost:5432/skillmaker?sslmode=disable

.PHONY: db-up
db-up: ## Levanta PostgreSQL + MinIO de desarrollo
	cd backend && docker compose -f docker-compose.dev.yml up -d

.PHONY: db-down
db-down: ## Detiene PostgreSQL + MinIO de desarrollo
	cd backend && docker compose -f docker-compose.dev.yml down

.PHONY: db-reset
db-reset: ## Borra volumen de Postgres y vuelve a levantar
	cd backend && docker compose -f docker-compose.dev.yml down -v && \
	  docker compose -f docker-compose.dev.yml up -d

.PHONY: db-migrate
db-migrate: ## Aplica migraciones pendientes
	migrate -path backend/migrations -database "$(DB_URL)" up

.PHONY: db-migrate-down
db-migrate-down: ## Revierte la ultima migracion
	migrate -path backend/migrations -database "$(DB_URL)" down 1

.PHONY: db-migrate-new
db-migrate-new: ## Crea archivos .up.sql/.down.sql para una nueva migracion (name=...)
	@test -n "$(name)" || (echo "Uso: make db-migrate-new name=add_user_table"; exit 1)
	migrate create -ext sql -dir backend/migrations -seq $(name)

.PHONY: db-seed
db-seed: ## Pobla la BD con datos iniciales (ej: roles)
	cd backend && go run ./cmd/seed

## ---------- Docs ----------
.PHONY: swagger
swagger: ## Regenera el OpenAPI desde anotaciones (swaggo/swag)
	cd backend && swag init -g cmd/api/main.go -o docs

## ---------- Docker (produccion) ----------
.PHONY: docker-up
docker-up: ## Levanta los 4 servicios de produccion
	docker compose up -d --build

.PHONY: docker-down
docker-down: ## Detiene los 4 servicios de produccion
	docker compose down

.PHONY: docker-logs
docker-logs: ## Tail de logs de produccion
	docker compose logs -f
```

### 4.3 Convencion de Nombres de Targets

| Patron | Ejemplo | Descripcion |
|--------|---------|-------------|
| `backend-*` | `backend-dev`, `backend-test` | Operacion sobre el workspace `backend` |
| `frontend-*` | `frontend-dev`, `frontend-lint` | Operacion sobre el workspace `frontend` |
| Sin prefijo (agregado) | `dev`, `build`, `test`, `lint` | Ejecuta el target equivalente en ambos workspaces |
| `db-*` | `db-up`, `db-migrate` | Operaciones de base de datos |
| `docker-*` | `docker-up`, `docker-down` | Operaciones de produccion / orquestacion completa |
| `swagger` | `make swagger` | Generacion del OpenAPI |

**Regla:** los targets agregados (`build`, `test`, etc.) **delegan** a los targets de workspace via dependencias de make. No contienen logica propia.

### 4.4 Agregar un Nuevo Workspace

Para agregar un tercer workspace (ej: un CLI auxiliar `tools/`):

1. Crear el directorio con su toolchain propia (otro `go.mod`, o un `package.json`).
2. Agregar targets al Makefile siguiendo la convencion (`tools-build`, `tools-test`).
3. Agregar el workspace a los targets agregados si corresponde.
4. Si el workspace es Go y vive dentro del mismo modulo, mejor usar un sub-paquete en `backend/cmd/` en vez de un workspace separado.

---

## 5. Gestion de Dependencias

Cada workspace gestiona sus dependencias por separado, con lockfiles independientes. **No hay un node_modules ni un GOPATH compartido.**

### 5.1 Dependencias del Backend (Go)

Gestionadas por `go mod`. El archivo `backend/go.mod` declara el modulo y `backend/go.sum` fija los hashes (lockfile).

```bash
# Agregar una dependencia
cd backend
go get github.com/golang-jwt/jwt/v5

# Actualizar dependencias
go get -u ./...
go mod tidy   # limpia imports no usados

# Auditoria de vulnerabilidades
govulncheck ./...
```

**Dependencias core del backend (referencia):**

| Dependencia | Proposito |
|-------------|-----------|
| `github.com/gin-gonic/gin` | Framework HTTP (RT-06) |
| `gorm.io/gorm` + `gorm.io/driver/postgres` | ORM + driver Postgres (RT-17) |
| `github.com/golang-migrate/migrate/v4` | Migraciones reproducibles (RT-18) |
| `github.com/golang-jwt/jwt/v5` | Emision/validacion de JWT (RT-14) |
| `google.golang.org/api/idtoken` | Validacion del ID token de Google OAuth (RT-12, RT-13) |
| `github.com/minio/minio-go/v7` | Cliente S3-compatible (RT-21) |
| `github.com/swaggo/swag` + `swaggo/gin-swagger` | OpenAPI desde anotaciones (RT-09) |
| `github.com/spf13/viper` o `github.com/caarlos0/env` | Carga de variables de entorno (RT-11) |
| `go.uber.org/zap` o `log/slog` | Logging estructurado (RT-28) |
| `github.com/stretchr/testify` | Aserciones de tests |
| `github.com/testcontainers/testcontainers-go` | Tests de integracion contra Postgres real |
| `github.com/cosmtrek/air` (dev) | Hot reload local |

### 5.2 Dependencias del Frontend (npm)

Gestionadas por `npm` con su propio `package-lock.json` dentro de `frontend/`.

```bash
# Agregar una dependencia
cd frontend
npm install primeng

# Auditoria
npm audit
```

### 5.3 Por que NO Compartir Dependencias entre Workspaces

A diferencia de un monorepo Node puro, aqui no hay forma natural de compartir dependencias entre Go y npm. Cualquier "logica compartida" (ej: contratos de API) se materializa via:

- **OpenAPI/Swagger**: el backend genera el spec y el frontend genera sus tipos TypeScript desde ese spec (con `openapi-typescript` o similar). Asi se mantiene un unico contrato sin duplicar definiciones a mano.
- **No mas que eso.** Si se necesita reutilizar logica de dominio, debe vivir en uno de los dos lados; cruzar la frontera con codigo compartido va contra el principio de bajo acoplamiento (RT-08).

---

## 6. Variables de Entorno y Configuracion

### 6.1 Archivos .env y su Alcance

| Archivo | Ubicacion | Usado por | Entorno |
|---------|-----------|-----------|---------|
| `backend/.env` | `backend/` | Backend Go (cargado via viper/env) | Desarrollo local |
| `backend/.env.example` | `backend/` | Template — copiar a `.env` | — |
| `.env` (root) | Raiz | `docker-compose.yml` (produccion) | Produccion |
| `.env.example` (root) | Raiz | Template de produccion | — |

Ambos `.env` estan en `.gitignore`. **Nunca se commitean secrets** (RNFT-03).

### 6.2 Flujo de Secrets por Entorno

```
DESARROLLO:
  backend/.env.example ──(copiar)──> backend/.env ──> Config loader (Go)
  frontend/src/environments/environment.ts ──> Angular (compilado en build)

PRODUCCION:
  .env (root) ──> docker-compose.yml (${VAR} substitution) ──> containers
  frontend/src/environments/environment.prod.ts ──> Angular (compilado en build:prod)
```

### 6.3 Variables del Backend

| Variable | Categoria | Descripcion | Ejemplo |
|----------|-----------|-------------|---------|
| `APP_ENV` | Server | `development` \| `production` | `development` |
| `PORT` | Server | Puerto del servidor Gin | `3000` |
| `LOG_LEVEL` | Server | Nivel de logging | `debug` \| `info` \| `warn` \| `error` |
| `ALLOWED_ORIGINS` | Server | Origins permitidos para CORS (coma-separados) | `http://localhost:4200` |
| `DATABASE_URL` | DB | Connection string PostgreSQL | `postgres://user:pass@localhost:5432/skillmaker?sslmode=disable` |
| `DB_MAX_OPEN_CONNS` | DB | Pool de conexiones (opcional) | `25` |
| `DB_MAX_IDLE_CONNS` | DB | Pool de conexiones idle (opcional) | `5` |
| `JWT_SECRET` | Auth | Secret HS256 para firmar el access JWT | String aleatorio largo |
| `JWT_EXPIRES_IN` | Auth | Duracion del access JWT (Go duration) | `1h` |
| `REFRESH_TOKEN_EXPIRES_IN` | Auth | Duracion del refresh token | `168h` (7 dias) |
| `GOOGLE_CLIENT_ID` | Auth | OAuth 2.0 client ID de Google | `xxxxx.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET` | Auth | OAuth 2.0 client secret (si se usa code flow server-side) | — |
| `GOOGLE_REDIRECT_URI` | Auth | Callback OAuth | `http://localhost:3000/api/auth/google/callback` |
| `GOOGLE_HOSTED_DOMAIN` | Auth | Dominio corporativo permitido (RT-13) | `tuempresa.com` |
| `STORAGE_ENDPOINT` | Storage | Endpoint S3/MinIO | `http://localhost:9000` (dev) / `s3.amazonaws.com` (prod) |
| `STORAGE_REGION` | Storage | Region S3 | `us-east-1` |
| `STORAGE_BUCKET` | Storage | Bucket para material adjunto | `skillmaker-materials` |
| `STORAGE_ACCESS_KEY` | Storage | Access key ID | — |
| `STORAGE_SECRET_KEY` | Storage | Secret access key | — |
| `STORAGE_USE_SSL` | Storage | Habilita TLS al cliente | `true` (prod) / `false` (dev MinIO) |
| `STORAGE_PRESIGN_TTL` | Storage | TTL de URLs prefirmadas | `15m` |
| `MAX_UPLOAD_BYTES` | Storage | Limite de tamano para material adjunto (RT-24b) | `52428800` (50 MB) |

**Diferencia con la guia anterior (NestJS):**
- **Eliminadas:** `REFRESH_TOKEN_EXPIRES_IN_DAYS`, `MAIL_SERVICE`, `SMTP_USER`, `SMTP_PASS`, `MAIL_FROM`, `PASSWORD_RESET_URL`, `RESET_PASSWORD_TOKEN_EXPIRATION`. El sistema NO maneja contrasenas propias ni envia emails de reset (RT-12, RNF-03). Toda la auth se delega a Google Workspace.
- **Agregadas:** `GOOGLE_*` (RT-12, RT-13) y `STORAGE_*` (RT-21 a RT-24b).

### 6.4 Variables del Frontend (environments/)

El frontend usa archivos TypeScript en vez de `.env`. Se compilan en build time.

**Desarrollo** (`frontend/src/environments/environment.ts`):
```typescript
export const environment = {
  production: false,
  apiBaseUrl: 'http://localhost:3000/api',
  googleClientId: 'xxxxx.apps.googleusercontent.com',
  googleHostedDomain: 'tuempresa.com',
};
```

**Produccion** (`frontend/src/environments/environment.prod.ts`):
```typescript
export const environment = {
  production: true,
  apiBaseUrl: '/api',
  googleClientId: 'xxxxx.apps.googleusercontent.com',
  googleHostedDomain: 'tuempresa.com',
};
```

**Diferencia clave:** En desarrollo, `apiBaseUrl` es absoluto (`http://localhost:3000/api`) porque el frontend (`:4200`) y el backend (`:3000`) estan en origenes distintos. En produccion es relativo (`/api`) porque nginx hace el proxying y todo vive bajo el mismo origen.

**Nota:** `googleClientId` es publico por diseno del flujo OAuth; no es un secret. El `googleClientSecret` vive solo en el backend.

---

## 7. Estrategia Docker

### 7.1 Vision General: 3 Archivos Docker Compose

| Archivo | Ubicacion | Servicios | Uso |
|---------|-----------|-----------|-----|
| `docker-compose.dev.yml` | `backend/` | PostgreSQL + MinIO | Desarrollo local diario |
| `docker-compose.yml` | `backend/` | PostgreSQL + MinIO + Backend | Testing local del container backend |
| `docker-compose.yml` | Raiz | PostgreSQL + MinIO + Backend + Frontend | Produccion (orquestacion completa) |

**Flujo tipico:**
- Desarrollo diario → `backend/docker-compose.dev.yml` (solo deps, app corre nativa con Air/ng serve)
- Probar el build del backend → `backend/docker-compose.yml` (backend containerizado + deps)
- Deployment completo → `docker-compose.yml` en la raiz (los 4 servicios)

### 7.2 Desarrollo: docker-compose.dev.yml (PostgreSQL + MinIO)

```yaml
# backend/docker-compose.dev.yml
services:
  postgres-dev:
    image: postgres:16
    restart: unless-stopped
    environment:
      POSTGRES_USER: skillmaker
      POSTGRES_PASSWORD: skillmaker
      POSTGRES_DB: skillmaker
    ports:
      - '5432:5432'              # Expuesto al host: backend nativo se conecta directo
    volumes:
      - postgres_dev_data:/var/lib/postgresql/data
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U skillmaker -d skillmaker']
      interval: 10s
      timeout: 5s
      retries: 5

  minio-dev:
    image: minio/minio:latest
    restart: unless-stopped
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    ports:
      - '9000:9000'              # API S3-compatible
      - '9001:9001'              # Consola web (http://localhost:9001)
    volumes:
      - minio_dev_data:/data
    healthcheck:
      test: ['CMD', 'curl', '-f', 'http://localhost:9000/minio/health/live']
      interval: 10s
      timeout: 5s
      retries: 5

  minio-init:
    image: minio/mc:latest
    depends_on:
      minio-dev:
        condition: service_healthy
    entrypoint: >
      /bin/sh -c "
      mc alias set local http://minio-dev:9000 minioadmin minioadmin;
      mc mb --ignore-existing local/skillmaker-materials;
      mc anonymous set download local/skillmaker-materials;
      exit 0;
      "

volumes:
  postgres_dev_data:
  minio_dev_data:
```

**Puntos clave:**
- Puerto `5432` (Postgres) y `9000` (MinIO API) expuestos al host — el backend Go nativo se conecta directo.
- Consola MinIO en `:9001` para inspeccionar buckets durante desarrollo.
- `minio-init` es un job efimero que crea el bucket inicial. Equivalente a `prisma seed` para storage.
- Healthchecks de Postgres (`pg_isready`) y MinIO (`/minio/health/live`).

**Comandos desde la raiz:**
```bash
make db-up        # Levanta PostgreSQL + MinIO
make db-down      # Detiene ambos
make db-reset     # Borra volumenes y vuelve a levantar (reset completo)
```

### 7.3 Testing Local: backend/docker-compose.yml (Backend + Deps)

Levanta el backend containerizado junto con sus dependencias. Util para verificar que el Dockerfile funciona antes del deployment completo.

```yaml
# backend/docker-compose.yml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: skillmaker
      POSTGRES_PASSWORD: skillmaker
      POSTGRES_DB: skillmaker
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U skillmaker -d skillmaker']
      interval: 10s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio:latest
    command: server /data
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    volumes:
      - minio_data:/data
    healthcheck:
      test: ['CMD', 'curl', '-f', 'http://localhost:9000/minio/health/live']
      interval: 10s
      timeout: 5s
      retries: 5

  backend:
    build: .
    ports:
      - '3000:3000'
    depends_on:
      postgres:
        condition: service_healthy
      minio:
        condition: service_healthy
    environment:
      APP_ENV: development
      PORT: 3000
      DATABASE_URL: postgres://skillmaker:skillmaker@postgres:5432/skillmaker?sslmode=disable
      JWT_SECRET: super_secret_key_dev_only
      JWT_EXPIRES_IN: 1h
      REFRESH_TOKEN_EXPIRES_IN: 168h
      GOOGLE_CLIENT_ID: changeme.apps.googleusercontent.com
      GOOGLE_HOSTED_DOMAIN: example.com
      STORAGE_ENDPOINT: http://minio:9000
      STORAGE_REGION: us-east-1
      STORAGE_BUCKET: skillmaker-materials
      STORAGE_ACCESS_KEY: minioadmin
      STORAGE_SECRET_KEY: minioadmin
      STORAGE_USE_SSL: 'false'
      ALLOWED_ORIGINS: 'http://localhost:4200'

volumes:
  postgres_data:
  minio_data:
```

**Diferencias con el compose de produccion (raiz):**
- Credenciales hardcodeadas (no usa `.env`)
- Solo 3 servicios (sin frontend)
- Backend en puerto `3000:3000` (no `3002:3000`)
- Sin `restart: unless-stopped`

### 7.4 Produccion: docker-compose.yml (Orquestacion Completa)

Orquesta los 4 servicios para produccion. Todas las variables vienen del archivo `.env` en la raiz.

```yaml
# docker-compose.yml (raiz)
services:
  postgres:
    image: postgres:16
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}']
      interval: 10s
      timeout: 5s
      retries: 5
    # Sin "ports:" — NO expuesto al host, solo accesible por otros containers

  minio:
    image: minio/minio:latest
    restart: unless-stopped
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - minio_data:/data
    healthcheck:
      test: ['CMD', 'curl', '-f', 'http://localhost:9000/minio/health/live']
      interval: 10s
      timeout: 5s
      retries: 5
    # Si se usa S3/GCS en produccion, este servicio puede eliminarse.
    # Mantenerlo permite deploys autocontenidos en VPS / on-premise.

  migrate:
    image: migrate/migrate:latest
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - ./backend/migrations:/migrations
    command: >
      -path=/migrations
      -database=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
      up
    restart: 'no'                # Job de un solo uso

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - '3002:3000'                               # Host:3002 → Container:3000
    depends_on:
      postgres:
        condition: service_healthy
      minio:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully  # Espera a que migraciones pasen
    environment:
      APP_ENV: production
      PORT: 3000
      LOG_LEVEL: info
      ALLOWED_ORIGINS: ${ALLOWED_ORIGINS}
      DATABASE_URL: ${DATABASE_URL}
      JWT_SECRET: ${JWT_SECRET}
      JWT_EXPIRES_IN: ${JWT_EXPIRES_IN}
      REFRESH_TOKEN_EXPIRES_IN: ${REFRESH_TOKEN_EXPIRES_IN}
      GOOGLE_CLIENT_ID: ${GOOGLE_CLIENT_ID}
      GOOGLE_CLIENT_SECRET: ${GOOGLE_CLIENT_SECRET}
      GOOGLE_REDIRECT_URI: ${GOOGLE_REDIRECT_URI}
      GOOGLE_HOSTED_DOMAIN: ${GOOGLE_HOSTED_DOMAIN}
      STORAGE_ENDPOINT: ${STORAGE_ENDPOINT}
      STORAGE_REGION: ${STORAGE_REGION}
      STORAGE_BUCKET: ${STORAGE_BUCKET}
      STORAGE_ACCESS_KEY: ${STORAGE_ACCESS_KEY}
      STORAGE_SECRET_KEY: ${STORAGE_SECRET_KEY}
      STORAGE_USE_SSL: ${STORAGE_USE_SSL}
      STORAGE_PRESIGN_TTL: ${STORAGE_PRESIGN_TTL}
      MAX_UPLOAD_BYTES: ${MAX_UPLOAD_BYTES}
    restart: unless-stopped

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - '3700:80'                                  # Host:3700 → nginx:80
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  postgres_data:
  minio_data:
```

**Patrones clave:**

1. **Migraciones como job separado:** el servicio `migrate` corre `golang-migrate` una sola vez y termina (`restart: 'no'`). El backend solo arranca cuando el job termina exitosamente (`service_completed_successfully`). Esto es mas robusto que correr migraciones dentro del entrypoint del binario porque:
   - El backend no necesita el cliente `migrate` embebido.
   - Si la migracion falla, el backend ni arranca (el container `migrate` queda con exit != 0 y `service_completed_successfully` no se satisface).
   - Permite escalar el backend horizontalmente en el futuro sin que cada replica corra migraciones.

2. **Dependencias con healthcheck:** `condition: service_healthy` asegura que Postgres y MinIO esten aceptando conexiones antes de migrar o levantar el backend.

3. **PostgreSQL sin puerto expuesto:** solo accesible desde la red interna de Docker. Igual con MinIO si todo el trafico de archivos pasa por el backend (recomendado para usar URLs prefirmadas con el endpoint publico del backend).

4. **Puertos de produccion:** Backend en `3002`, Frontend en `3700`. Distintos a los de desarrollo para evitar conflictos si se prueba en la misma maquina.

### 7.5 Dockerfile del Backend (Multi-stage Go + Distroless)

```dockerfile
# backend/Dockerfile

# Stage 1: Build — compila el binario estatico
FROM golang:1.23-alpine AS build
WORKDIR /src

# Cache de dependencias: copiar go.mod/go.sum antes que el codigo
COPY go.mod go.sum ./
RUN go mod download

# Copiar codigo fuente y compilar
COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Stage 2: Runtime — imagen minima sin shell ni libc completa
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=build /out/api /app/api

EXPOSE 3000
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
```

**Patron de cache:** `go mod download` se ejecuta ANTES de copiar el codigo fuente. Si el codigo cambia pero las dependencias no, Docker reutiliza la capa cacheada.

**Por que distroless:**
- La imagen final pesa ~20 MB (vs ~300 MB de `golang:alpine`).
- No tiene shell (`sh`/`bash`), reduciendo superficie de ataque.
- `nonroot` user por defecto — el binario corre sin privilegios.
- `CGO_ENABLED=0` produce un binario estatico sin dependencias dinamicas, asi `distroless/static` alcanza.

**Flags de build:**
- `-trimpath`: elimina rutas absolutas del host del build (mejor reproducibilidad).
- `-ldflags="-s -w"`: strip de simbolos para reducir tamano (~30% menos).

**Comparacion con el Dockerfile anterior (NestJS):**
- NestJS necesita Node.js en runtime (~150 MB) + node_modules (~200 MB). Total: ~400 MB.
- Go con distroless: ~20 MB. **20x mas chico.**

### 7.6 Dockerfile del Frontend (Multi-stage Angular + Nginx)

```dockerfile
# frontend/Dockerfile

# Stage 1: Build — compila la app Angular con AOT
FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci --legacy-peer-deps
COPY . .
RUN npm run build:prod

# Stage 2: Runtime — sirve los archivos estaticos con nginx
FROM nginx:alpine AS runtime
COPY --from=build /app/dist/skillmaker-frontend/browser /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

**Sin cambios significativos vs la guia anterior** — el frontend sigue siendo Angular. Solo se actualiza la ruta del bundle de salida segun el `outputPath` del proyecto.

**`--legacy-peer-deps`:** necesario por conflictos de peer dependencies entre librerias del ecosistema Angular.

### 7.7 Nginx: Reverse Proxy y SPA Fallback

```nginx
# frontend/nginx.conf
server {
    listen 80;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    # Proxy API requests al backend Go
    location /api/ {
        proxy_pass http://backend:3000/api/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;

        # Tamano maximo de upload (material adjunto)
        client_max_body_size 50M;
    }

    # Swagger UI (solo expuesto al frontend si se desea)
    location /api/docs {
        proxy_pass http://backend:3000/api/docs;
        proxy_set_header Host $host;
    }

    # Angular SPA fallback
    location / {
        try_files $uri $uri/ /index.html;

        # Cache largo para assets versionados (Angular les agrega hash)
        location ~* \.(js|css|png|jpg|jpeg|gif|svg|woff|woff2)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }
    }

    # No cachear index.html para que los deploys se vean inmediatamente
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        expires 0;
    }
}
```

**Dos funciones criticas:**

1. **Reverse proxy para API:** toda peticion a `/api/*` se reenvia al container `backend:3000`. El nombre `backend` lo resuelve el DNS interno de Docker Compose.

2. **SPA fallback:** toda ruta que no sea un archivo estatico ni `/api/*` devuelve `index.html`. Esto permite que Angular Router maneje las rutas del lado del cliente.

**`client_max_body_size 50M`** alinea el limite de nginx con `MAX_UPLOAD_BYTES` del backend (RT-24b).

---

## 8. Comunicacion entre Servicios

### 8.1 En Desarrollo (Puertos Directos)

```
┌────────────┐    HTTP directa     ┌────────────┐    TCP directo    ┌────────────┐
│  Browser   │ ──────────────────> │  Gin       │ ──────────────── │ PostgreSQL │
│            │                     │  :3000     │                  │  :5432     │
│  ng serve  │ <── apiBaseUrl ──── │  (Air)     │                  │  (Docker)  │
│  :4200     │   http://...:3000   │            │                  └────────────┘
└────────────┘                     │            │ ───── S3 API ──> ┌────────────┐
                                   └────────────┘                  │  MinIO     │
                                                                   │  :9000     │
                                                                   │  (Docker)  │
                                                                   └────────────┘
```

- Frontend Angular en `:4200` (`ng serve`)
- Backend Go en `:3000` con Air (hot reload)
- PostgreSQL en `:5432` (Docker)
- MinIO en `:9000` (Docker) — el backend habla S3 contra el
- El frontend llama directo al backend via `http://localhost:3000/api`
- **CORS habilitado en el backend** para permitir requests desde `:4200` (RT-04, RT-05)

### 8.2 En Produccion (Red Docker + Nginx Proxy)

```
                        Docker Network
┌────────────┐    ┌──────────────────────────────────────────────────┐
│            │    │                                                  │
│  Browser   │    │  ┌────────────┐         ┌────────────┐          │
│            │ ───┼─>│  nginx     │──/api/──>│  Gin       │          │
│            │    │  │  :80 :3700 │         │  :3000      │          │
│            │    │  │            │         │  :3002      │          │
│            │    │  │  Angular   │         │            │          │
│            │    │  │  static    │         └─────┬──────┘          │
│            │    │  └────────────┘               │                 │
│            │    │                        ┌──────▼────────┐        │
│            │    │                        │ PostgreSQL    │        │
│            │    │                        │  :5432        │        │
│            │    │                        │  (internal)   │        │
│            │    │                        └───────────────┘        │
│            │    │                        ┌───────────────┐        │
│            │    │                        │  MinIO/S3     │        │
│            │    │                        │  (internal)   │        │
└────────────┘    └────────────────────────┴───────────────┴────────┘
```

- El browser accede a nginx en el puerto `:3700`
- nginx sirve los archivos estaticos de Angular
- Las peticiones a `/api/*` se proxyean a `backend:3000` por DNS interno de Docker
- PostgreSQL y MinIO NO tienen puertos expuestos al host (solo accesibles dentro de la red Docker)
- `apiBaseUrl: '/api'` — ruta relativa; nginx hace el routing

### 8.3 Diferencia Clave: CORS

| Entorno | CORS necesario | Razon |
|---------|---------------|-------|
| Desarrollo | Si | Frontend (`:4200`) y Backend (`:3000`) estan en origenes distintos |
| Produccion | No | Todo pasa por nginx (`:80`), mismo origen |

En Go + Gin, el middleware CORS se configura via `github.com/gin-contrib/cors` consumiendo `ALLOWED_ORIGINS` desde el environment. En produccion, esa variable se deja vacia o con el origen unico del frontend.

### 8.4 Comunicacion entre Modulos del Backend (Crucial)

Este es uno de los puntos mas importantes del diseno (RT-08, RT-19, RNFT-02). **No es comunicacion entre servicios** — es comunicacion **dentro** del binario, pero gobernada como si fuera entre servicios.

**Reglas:**

1. **Cada modulo expone una interface** (en Go: un `interface` declarado en `internal/modules/<modulo>/<modulo>.go` o `api.go`).
2. **Los demas modulos dependen de esa interface**, nunca de structs concretos ni de las tablas internas del otro modulo.
3. **La capa de wiring (`cmd/api/main.go`)** construye las implementaciones y las inyecta — patron Composition Root.
4. **Sin queries SQL cross-modulo**. Si `evaluations` necesita el nombre del usuario, llama a `users.GetByID(...)`, no hace `JOIN` contra la tabla de users.
5. **Sin transacciones que abarquen tablas de varios modulos** — serian imposibles de mantener tras la separacion a microservicios (recomendacion 5.6 del doc tecnico).
6. **Eventos internos opcionales** para flujos asincronos naturales (ej: `curso aprobado` → generar certificado / notificar). Hoy son llamadas a funciones; manana pueden moverse a una cola sin cambiar la logica de dominio.

Esta disciplina es lo que permite extraer un modulo como microservicio (RNFT-02) sin reescribirlo. Pasa de:

```go
// Hoy (monolito modular)
certs := certificates.New(...)
courses := courses.New(certs, ...)   // courses depende de la interface Certificates
```

A:

```go
// Manana (microservicio)
certs := certificatesclient.NewHTTP("http://certificates-svc")
courses := courses.New(certs, ...)   // misma interface, otra implementacion
```

---

## 9. Workflow de Desarrollo

### 9.1 Pre-requisitos

Instalar una vez en la maquina:

- **Go** 1.23+
- **Node.js** 20+ y **npm** 10+
- **Docker** y **Docker Compose**
- **Make** (preinstalado en Linux/macOS; en Windows usar WSL2)
- **golang-migrate** CLI: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
- **Air** (hot reload): `go install github.com/air-verse/air@latest`
- **swag** (Swagger): `go install github.com/swaggo/swag/cmd/swag@latest`
- **golangci-lint**: `brew install golangci-lint` o ver instalacion oficial

### 9.2 Setup Inicial (Primera Vez)

```bash
# 1. Clonar e instalar dependencias de ambos workspaces
git clone <repo-url>
cd skillmaker
make install

# 2. Configurar variables de entorno del backend
cp backend/.env.example backend/.env
# Editar backend/.env con credenciales reales:
#   - GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET (consola de Google Cloud)
#   - GOOGLE_HOSTED_DOMAIN (dominio corporativo)
#   - JWT_SECRET (string aleatorio largo)
#   - resto: dejar valores por defecto para dev

# 3. Levantar PostgreSQL y MinIO
make db-up

# 4. Aplicar migraciones y poblar datos iniciales
make db-migrate
make db-seed

# 5. Generar el spec OpenAPI (primera vez)
make swagger

# 6. Iniciar desarrollo (backend con Air + frontend con ng serve)
make dev
```

### 9.3 Desarrollo Diario

```bash
# Levantar dependencias (si no estan corriendo)
make db-up

# Iniciar ambos servicios en paralelo
make dev
# Output:
# [BACKEND]  watching .
# [BACKEND]  building...
# [BACKEND]  running...
# [FRONTEND] Angular Live Development Server is listening on localhost:4200
```

El target `dev` usa `concurrently` para correr backend (Air) y frontend (`ng serve`) en paralelo con prefijos coloreados.

**Hot reload:**
- **Backend (Air):** detecta cambios en `*.go`, recompila y reinicia el binario. Configurable en `backend/.air.toml`.
- **Frontend (ng serve):** HMR de Angular CLI.

### 9.4 Puertos y URLs

| Servicio | Entorno | Puerto | URL |
|----------|---------|--------|-----|
| Frontend | dev (ng serve) | 4200 | `http://localhost:4200` |
| Backend | dev (Air) | 3000 | `http://localhost:3000` |
| Swagger UI | dev | 3000 | `http://localhost:3000/api/docs/index.html` |
| PostgreSQL | dev (docker-compose.dev) | 5432 | `postgres://skillmaker:skillmaker@localhost:5432/skillmaker` |
| MinIO API | dev | 9000 | `http://localhost:9000` |
| MinIO Console | dev | 9001 | `http://localhost:9001` (user: minioadmin / pass: minioadmin) |
| Backend | test local (backend/docker-compose) | 3000 | `http://localhost:3000` |
| Backend | produccion (root docker-compose) | 3002 | `http://localhost:3002` |
| Frontend | produccion (root docker-compose) | 3700 | `http://localhost:3700` |

### 9.5 Base de Datos (golang-migrate)

Todos los comandos se ejecutan desde la raiz via Makefile:

| Comando | Descripcion |
|---------|-------------|
| `make db-up` | Levanta PostgreSQL + MinIO de desarrollo |
| `make db-down` | Detiene PostgreSQL + MinIO |
| `make db-reset` | Borra volumen de Postgres y vuelve a levantar (reset total) |
| `make db-migrate` | Aplica migraciones pendientes (`migrate up`) |
| `make db-migrate-down` | Revierte la ultima migracion (`migrate down 1`) |
| `make db-migrate-new name=add_x_table` | Crea archivos `NNNN_add_x_table.up.sql` y `.down.sql` |
| `make db-seed` | Pobla la BD con datos iniciales (ej: roles del catalogo) |

**Flujo tipico al agregar una entidad:**
1. `make db-migrate-new name=add_courses_table` — crea los SQL vacios
2. Escribir el DDL en `migrations/NNNN_add_courses_table.up.sql` (CREATE TABLE)
3. Escribir el rollback en `migrations/NNNN_add_courses_table.down.sql` (DROP TABLE)
4. `make db-migrate` — aplica al esquema local
5. Crear el modelo GORM correspondiente en `backend/internal/modules/courses/`

**Por que SQL plano y no AutoMigrate de GORM:**
- **Reproducible** (RT-18): el mismo SQL corre identico en dev/staging/prod.
- **Auditable**: cada cambio queda en git como un archivo separado.
- **Rollback explicito**: `down.sql` permite revertir migraciones problematicas.
- **GORM AutoMigrate es destructivo en evoluciones complejas** (no maneja renames, drops, ni transformaciones de datos).
- AutoMigrate puede usarse opcionalmente en tests con SQLite efimero o en arranque local muy temprano, pero NUNCA en produccion.

### 9.6 Documentacion de API (Swagger)

El spec OpenAPI se genera desde anotaciones en los handlers Go via `swaggo/swag`. Despues de agregar o modificar endpoints:

```bash
make swagger        # Regenera backend/docs/swagger.json y .yaml
```

El backend sirve la UI en `http://localhost:3000/api/docs/index.html` durante desarrollo. En produccion, se puede dejar accesible o bloquear via middleware segun la politica del equipo.

---

## 10. Deployment a Produccion

### 10.1 Build de Imagenes Docker

Docker Compose construye las imagenes automaticamente desde los Dockerfiles de cada workspace:

```bash
# Construir y levantar todos los servicios
docker compose up -d --build
# o via Makefile
make docker-up
```

**Flujo de build:**
1. PostgreSQL: imagen oficial `postgres:16` (no se construye).
2. MinIO: imagen oficial `minio/minio:latest` (puede omitirse si se usa S3/GCS).
3. Backend: `backend/Dockerfile` — descarga deps Go, compila binario estatico, copia a distroless.
4. Frontend: `frontend/Dockerfile` — instala deps npm, compila Angular AOT, copia a nginx.

### 10.2 Migraciones en Produccion

Las migraciones se ejecutan via un servicio `migrate` separado en docker-compose:

```yaml
migrate:
  image: migrate/migrate:latest
  command: -path=/migrations -database=$DATABASE_URL up
  restart: 'no'

backend:
  depends_on:
    migrate:
      condition: service_completed_successfully
```

**Por que no embebido en el binario backend:**
- Permite escalar `backend` horizontalmente sin race conditions en migraciones.
- El binario backend queda mas chico (no embebe el cliente `migrate`).
- Si una migracion falla, queda visible en `docker compose ps` y el backend no arranca.

**Importante:** `migrate up` solo aplica migraciones nuevas; nunca recrea ni dropea sin un `down` explicito. Es seguro para produccion.

### 10.3 Orquestacion con Docker Compose

```bash
# Levantar en produccion
make docker-up         # equivale a: docker compose up -d --build

# Ver logs de todos los servicios
make docker-logs       # equivale a: docker compose logs -f

# Ver logs de un servicio especifico
docker compose logs -f backend

# Detener todo
make docker-down       # equivale a: docker compose down

# Detener y borrar volumenes (CUIDADO: borra la base de datos y MinIO)
docker compose down -v

# Rebuild y redeploy
docker compose up -d --build
```

### 10.4 Healthchecks y Dependencias entre Servicios

```
postgres (healthcheck: pg_isready)         minio (healthcheck: /minio/health/live)
    │                                          │
    └──────── condition: service_healthy ──────┘
                       │
                       ▼
                  migrate (job: migrate up)
                       │
                       ▼ condition: service_completed_successfully
                   backend
                       │
                       ▼ (frontend depende para evitar 502 en arranque)
                   frontend
```

PostgreSQL tiene un healthcheck que ejecuta `pg_isready` cada 10s. MinIO usa el endpoint `/minio/health/live`. El backend no inicia hasta que:
1. Postgres este saludable
2. MinIO este saludable
3. El job `migrate` haya terminado con exit 0

### 10.5 Health Endpoint del Backend (RT-27)

El backend expone `GET /api/health` que retorna `200 OK` si el servidor esta vivo, y `GET /api/health/ready` que verifica que Postgres y MinIO sean alcanzables (readiness para load balancers o k8s).

### 10.6 Logging Estructurado (RT-28)

El backend emite logs JSON estructurados (zap o `log/slog`) a stdout. Docker recolecta automaticamente y el orquestador (compose, swarm, k8s) los expone via `docker logs` o agregadores externos (Loki, Datadog, etc.).

---

## 11. Testing

### 11.1 Backend: go test + testify + testcontainers-go

| Comando | Descripcion |
|---------|-------------|
| `make backend-test` | Tests unitarios con `-race` y cobertura |
| `make backend-test-integration` | Tests de integracion (tag `integration`, levantan Postgres con testcontainers) |

**Estructura de tests:**
- **Unit tests:** conviven con el codigo (`courses_service_test.go` al lado de `courses_service.go`). Mock de repos/clientes via interfaces.
- **Integration tests:** archivos con la primera linea `//go:build integration`. Levantan un Postgres efimero via `testcontainers-go` y corren queries reales.
- **Tabla-driven tests:** patron idiomatico de Go (`for _, tc := range tests`). Documentar cada caso con un nombre claro.

**Cobertura objetivo (RNFT-05):** al menos la capa de servicios de cada modulo.

Para patrones detallados ver `backend/GUIA-BACK.md`.

### 11.2 Frontend: Vitest

| Comando | Descripcion |
|---------|-------------|
| `make frontend-test` | Ejecutar tests una vez |
| `cd frontend && npm run test:watch` | Ejecutar en modo watch |

Usa `@analogjs/vitest-angular` como bridge entre Vitest y Angular TestBed.

### 11.3 Ejecutar Tests Combinados

```bash
make test
# Equivale a: make backend-test && make frontend-test
```

Los tests se ejecutan secuencialmente para evitar conflictos de recursos (Postgres, MinIO efimeros) y hacer el output mas legible.

### 11.4 CI/CD (Recomendacion)

Pipeline tipico (GitHub Actions / GitLab CI):

```
1. lint        → make lint
2. test-unit   → make backend-test && make frontend-test
3. test-integ  → make backend-test-integration  (con Postgres en service container)
4. build       → make build
5. docker      → docker build (backend + frontend)
6. deploy      → docker compose up -d (en server)
```

**Tip:** los tests con testcontainers requieren Docker disponible en el runner. En GitHub Actions usar `runs-on: ubuntu-latest` (ya trae Docker).

---

## 12. Linting y Formateo

Cada workspace gestiona su propio linter. No hay configuracion compartida.

| Workspace | Herramienta | Comando | Config |
|-----------|------------|---------|--------|
| Backend | golangci-lint + gofmt | `make backend-lint` | `backend/.golangci.yml` |
| Frontend | Angular ESLint | `make frontend-lint` | `angular.json` (lint target) |

**Configuracion del Backend (`.golangci.yml`):**
Linters recomendados activos: `govet`, `errcheck`, `staticcheck`, `gosec`, `gofmt`, `goimports`, `revive`, `unused`, `ineffassign`. Ajustar segun preferencias del equipo.

**Formateo:**
- Go: `gofmt -w .` (idempotente, parte del estandar del lenguaje). VS Code / Goland aplican on save por defecto.
- Frontend: Prettier integrado en Angular CLI.

**Pre-commit hook (opcional):** instalar `lefthook` o `husky` para correr `make lint` y `make test` antes de cada commit.

---

## 13. Agregar Nuevas Features End-to-End

### 13.1 Nuevo Modulo de Dominio (Backend Go)

Dado que el dominio ya esta cerrado (7 modulos definidos en el doc tecnico), agregar un modulo nuevo deberia ser raro. Si igualmente se requiere:

1. **Crear la estructura del modulo:**
```
backend/internal/modules/mifeature/
├── mifeature.go          # Define la interface publica (lo que otros modulos consumen)
├── handler/              # Handlers HTTP (capa de entrega)
│   └── handler.go
├── service/              # Logica de dominio (capa de negocio)
│   └── service.go
├── repository/           # Acceso a datos (GORM)
│   └── repository.go
├── domain/               # Entidades de dominio (modelos GORM, value objects)
│   └── models.go
└── dto/                  # Request/response DTOs
    ├── request.go
    └── response.go
```

2. **Definir la interface publica** en `mifeature.go`:
```go
package mifeature

import "context"

type Service interface {
    GetByID(ctx context.Context, id string) (*MyResponse, error)
    // ...
}
```

3. **Crear la migracion:** `make db-migrate-new name=add_mifeature_table` + escribir el DDL.

4. **Registrar las rutas** en `backend/internal/router/router.go` o en un `Register(r *gin.RouterGroup)` del propio modulo.

5. **Cablear las dependencias** en `cmd/api/main.go`:
```go
mifeatureRepo := mifeatureRepository.New(db)
mifeatureSvc := mifeatureService.New(mifeatureRepo, otherModule)
mifeatureHandler := mifeatureHandler.New(mifeatureSvc)
mifeatureHandler.Register(apiRouterGroup)
```

6. **Asegurar que el seed** cree los 4 roles fijos (`alumno`, `creador`, `supervisor`, `administrador`) en `backend/cmd/seed/main.go` si aun no existe. El control de acceso del modulo se hace contra estos roles, no contra permisos granulares.

7. **Documentar endpoints** con anotaciones swaggo:
```go
// CreateMiFeature godoc
// @Summary Crea un mifeature
// @Tags mifeature
// @Accept json
// @Produce json
// @Param body body dto.CreateRequest true "Body"
// @Success 201 {object} dto.Response
// @Router /mifeature [post]
// @Security BearerAuth
func (h *Handler) Create(c *gin.Context) { ... }
```

8. `make swagger` para regenerar el spec.

### 13.2 Nuevo Endpoint en un Modulo Existente

Mucho mas comun. Pasos:

1. Agregar el metodo a la interface `Service` del modulo.
2. Implementarlo en `service/service.go` (logica de dominio) y `repository/repository.go` (queries).
3. Crear el handler en `handler/handler.go` y registrarlo en el `Register(...)` del modulo.
4. Agregar el DTO en `dto/`.
5. Anotar con swaggo y correr `make swagger`.
6. Si toca el schema, generar migracion: `make db-migrate-new name=...`.

Para patrones detallados ver `backend/GUIA-BACK.md`.

### 13.3 Nueva Pagina Frontend + Ruta

1. Crear componente en `frontend/src/app/pages/`:
```
frontend/src/app/pages/mi-feature/
├── mi-feature.component.ts
├── mi-feature.component.html
└── mi-feature.component.sass
```

2. Agregar ruta lazy-loaded en `frontend/src/app/pages/pages.routes.ts`:
```typescript
{
  path: 'mi-feature',
  loadComponent: () =>
    import('./mi-feature/mi-feature.component')
      .then((m) => m.MiFeatureComponent),
  canActivate: [authGuard, roleGuard],
  data: { roles: ['administrador'] },
}
```

3. Agregar item de menu en el sidebar del layout.

Para patrones detallados ver `frontend/GUIA-FRONT.md`.

### 13.4 Conectar Frontend con Backend

1. **Generar tipos TypeScript desde el OpenAPI** (opcional, recomendado):
```bash
npx openapi-typescript backend/docs/swagger.json -o frontend/src/app/api/types.ts
```
Asi el contrato del backend se refleja como tipos TS sin duplicar definiciones a mano.

2. **Crear servicio en `frontend/src/app/core/services/`:**
```typescript
@Injectable({ providedIn: 'root' })
export class MiFeatureService {
  private readonly baseUrl = `${environment.apiBaseUrl}/mi-feature`;
  private readonly http = inject(HttpClient);

  getAll(): Observable<MiFeatureResponse[]> {
    return this.http.get<MiFeatureResponse[]>(this.baseUrl);
  }
}
```

3. **Inyectar en el componente con `inject()`:**
```typescript
private readonly miFeatureService = inject(MiFeatureService);
```

### 13.5 Flujo de Autorizacion End-to-End

El sistema usa **solo roles** (RF-02, RNF-04). No hay permisos granulares: el modelo de datos define unicamente la tabla `role`. Toda decision de autorizacion se basa en el conjunto de roles del usuario; las reglas de propiedad (ownership) las aplica el service.

```
1. SEED (backend, cmd/seed/main.go)
   └─ Crear los 4 roles fijos: alumno, creador, supervisor, administrador (RF-02)

2. BACKEND
   └─ Middleware RBAC en el router lee los roles del claim JWT
   └─ Cada handler protege con: r.GET("/x", middleware.RequireRole("administrador"), handler.X)
   └─ Reglas de ownership (ej: "solo el creador edita SU curso") viven en el service

3. FRONTEND - Rutas
   └─ pages.routes.ts: data: { roles: ['administrador'] }
   └─ roleGuard valida antes de cargar el componente

4. FRONTEND - UI
   └─ Template: *hasRole="'creador'" para mostrar/ocultar botones
   └─ Logica: roleCheckService.hasRole('administrador')
```

**Formato de roles:** los 4 roles fijos del dominio (`alumno`, `creador`, `supervisor`, `administrador`). No introducir permisos granulares hasta que un RF concreto lo exija.

### 13.6 Subir Material Adjunto (S3/MinIO)

Patron recomendado (RT-23): URLs prefirmadas.

```
Frontend                           Backend                          MinIO/S3
   │                                  │                                │
   ├──── POST /materials/presign ────>│                                │
   │     {filename, mime, size}       │                                │
   │                                  ├──── putObject presign ────────>│
   │                                  │<──── presigned URL ────────────┤
   │<──── {uploadUrl, key} ───────────┤                                │
   │                                  │                                │
   ├──── PUT uploadUrl (file) ────────┼───────────────────────────────>│
   │                                  │                                │
   ├──── POST /materials ────────────>│                                │
   │     {courseId, key, name, ...}   ├──── INSERT material ──────────>│ (Postgres)
   │<──── 201 created ────────────────┤                                │
```

El backend solo persiste la referencia (`storage_key`), nunca el archivo (RT-22). El frontend sube directo a MinIO/S3 via la URL prefirmada — esto evita pasar el archivo dos veces por la red y descarga al backend.

---

## 14. Directorios Auxiliares

### 14.1 bases/ — Documentacion del Dominio

Contiene la **fuente de verdad** del proyecto (RFs, RTs, modelo de datos). No es un workspace de codigo; es material de consulta que alimenta tanto el diseno como la implementacion.

Estructura actual:
```
bases/
└── documentacion/
    ├── Requerimientos_Plataforma_Formacion.docx           # RFs y RNFs
    ├── Requerimientos_Tecnicos_Plataforma_Formacion.docx  # RTs y arquitectura C4
    └── Modelo_Datos_Plataforma_Formacion.docx             # Diccionario + DDL
```

**Importante:** ante una decision arquitectonica, esta documentacion gana sobre cualquier convencion de la guia. Si encontras una discrepancia, abrir un issue para resolverla.

### 14.2 docs/ — Especificaciones de Diseno (opcional)

Cuando se trabaje en features grandes, agregar documentos de diseno:
```
docs/
└── features/
    └── YYYY-MM-DD-nombre-feature-design.md
```

**Convencion de nombres:** Fecha ISO + nombre descriptivo + `-design.md`.

### 14.3 .claude/ y .agents/ — Configuracion de IA

- **`.claude/`**: configuracion de Claude Code (settings, skills, MCP servers)
- **`.agents/`**: definiciones de skills para agentes IA
- **`.mcp.json`**: configuracion de MCP servers para herramientas de desarrollo

No son parte de la aplicacion — son herramientas del entorno de desarrollo.

---

## 15. Referencias Cruzadas

| Tema | Guia | Ruta |
|------|------|------|
| Estructura del monorepo, Makefile, Docker | Este documento | `GUIA-MONOREPO.md` |
| Patrones Angular: componentes, servicios, routing, auth, roles | Guia Frontend | `frontend/GUIA-FRONT.md` |
| Patrones Go + Gin: modulos, handlers, GORM, RBAC, testing | Guia Backend | `backend/GUIA-BACK.md` |
| Requerimientos funcionales y no funcionales | Fuente de verdad | `bases/documentacion/Requerimientos_Plataforma_Formacion.docx` |
| Requerimientos tecnicos y arquitectura C4 | Fuente de verdad | `bases/documentacion/Requerimientos_Tecnicos_Plataforma_Formacion.docx` |
| Modelo de datos y DDL de referencia | Fuente de verdad | `bases/documentacion/Modelo_Datos_Plataforma_Formacion.docx` |
| Instalacion y primera ejecucion | Setup | `SETUP.md` |
| Contexto general del proyecto para IA | Claude | `CLAUDE.md` |
| Migraciones SQL versionadas | golang-migrate | `backend/migrations/` |
| Documentacion de API (runtime, dev) | Swagger | `http://localhost:3000/api/docs/index.html` |
| Spec OpenAPI versionado | Swag generado | `backend/docs/swagger.json` |

### Orden de Lectura Recomendado para Agentes IA y Nuevos Desarrolladores

1. **`bases/documentacion/`** — Entender el dominio, RFs, RTs y modelo de datos (fuente de verdad)
2. **`CLAUDE.md`** — Contexto del proyecto (stack, convenciones de equipo)
3. **`GUIA-MONOREPO.md`** (este archivo) — Entender la infraestructura (workspaces, Makefile, Docker, deployment)
4. **`frontend/GUIA-FRONT.md`** o **`backend/GUIA-BACK.md`** — Segun donde vayas a trabajar
5. **`SETUP.md`** — Solo si necesitas levantar el entorno desde cero
