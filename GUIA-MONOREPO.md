# Guia del Monorepo — Arquitectura, Infraestructura y Deployment

Documentacion de la arquitectura del monorepo, su configuracion de workspaces, estrategia Docker, comunicacion entre servicios, y workflow de desarrollo y deployment. Sirve como referencia para replicar este patron en futuros proyectos full-stack.

> **Este documento es el "plano maestro" del monorepo.** Para patrones internos de cada workspace:
> - **Frontend (Angular):** ver [`frontend/GUIA-PROYECTO-BASE.md`](frontend/GUIA-PROYECTO-BASE.md)
> - **Backend (NestJS):** ver [`backend/PROJECT-PATTERNS-GUIDE.md`](backend/PROJECT-PATTERNS-GUIDE.md)
> - **Setup inicial:** ver [`SETUP.md`](SETUP.md)

---

## Tabla de Contenidos

1. [Introduccion y Proposito](#1-introduccion-y-proposito)
2. [Vision General de la Arquitectura](#2-vision-general-de-la-arquitectura)
3. [Estructura de Directorios](#3-estructura-de-directorios)
4. [npm Workspaces](#4-npm-workspaces)
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

Este documento describe **como esta construido el monorepo** y como replicar este patron para nuevos proyectos full-stack. No cubre los patrones internos de cada workspace (Angular o NestJS) — eso lo hacen las guias especificas.

### Sistema de 3 Guias

| Guia | Alcance | Analogia |
|------|---------|----------|
| **GUIA-MONOREPO.md** (este archivo) | Estructura del monorepo, workspaces, Docker, deployment, comunicacion entre servicios | La estructura de la casa: cimientos, paredes, plomeria |
| **GUIA-PROYECTO-BASE.md** | Patrones internos del frontend Angular | Como amueblar el living |
| **PROJECT-PATTERNS-GUIDE.md** | Patrones internos del backend NestJS | Como amueblar la cocina |

**Lectura recomendada:** Este documento primero (entender la casa), luego la guia del workspace donde vayas a trabajar.

---

## 2. Vision General de la Arquitectura

### Stack Tecnologico

| Capa | Tecnologia | Version | Rol |
|------|-----------|---------|-----|
| Frontend | Angular | 21 | SPA (Standalone, Zoneless) |
| UI | PrimeNG + TailwindCSS | 21 / 4 | Componentes + utilidades CSS |
| Backend | NestJS | 10 | API REST |
| ORM | Prisma | 6 | Acceso a base de datos |
| Base de datos | PostgreSQL | 16 | Persistencia |
| Contenedores | Docker + Compose | - | Desarrollo y produccion |
| Testing | Vitest (front) + Jest (back) | 2 / 29 | Testing unitario y E2E |
| Monorepo | npm Workspaces | - | Gestion de paquetes |

### Diagrama de Alto Nivel

```
┌─────────────────────────────────────────────────────────┐
│                    MONOREPO (npm workspaces)             │
│                                                         │
│  ┌──────────────────┐         ┌──────────────────────┐  │
│  │    frontend/      │         │      backend/         │  │
│  │                  │         │                      │  │
│  │  Angular 21      │  HTTP   │  NestJS 10           │  │
│  │  PrimeNG 21      │ ──────> │  Prisma 6            │  │
│  │  TailwindCSS 4   │  /api/* │  JWT + RBAC          │  │
│  │                  │         │                      │  │
│  │  Dev: :4200      │         │  Dev: :3000          │  │
│  │  Prod: nginx:80  │         │  Prod: node:3000     │  │
│  └──────────────────┘         └──────────┬───────────┘  │
│                                          │              │
│                                ┌─────────▼──────────┐   │
│                                │   PostgreSQL 16     │   │
│                                │   Dev: :5432       │   │
│                                └────────────────────┘   │
│                                                         │
│  package.json (root) ── workspaces + scripts + deps     │
│  docker-compose.yml  ── orquestacion produccion         │
└─────────────────────────────────────────────────────────┘
```

---

## 3. Estructura de Directorios

```
mi-proyecto/
├── backend/                        # Workspace: NestJS API
│   ├── src/                        # Codigo fuente (modulos, servicios, controllers)
│   ├── prisma/                     # Schema, migraciones, seeds
│   ├── test/                       # Tests E2E
│   ├── docker-compose.dev.yml      # PostgreSQL para desarrollo local
│   ├── docker-compose.yml          # Backend + PostgreSQL (testing local del container)
│   ├── Dockerfile                  # Build de produccion (single-stage)
│   ├── .env                        # Variables de entorno (gitignored)
│   ├── .env.example                # Template de variables
│   ├── nest-cli.json               # Config del CLI de NestJS
│   ├── tsconfig.json               # TypeScript config
│   ├── .eslintrc.js                # ESLint config
│   ├── .prettierrc                 # Prettier config
│   ├── PROJECT-PATTERNS-GUIDE.md   # Guia de patrones del backend
│   └── package.json                # Dependencias del workspace
│
├── frontend/                       # Workspace: Angular SPA
│   ├── src/                        # Codigo fuente (app/, environments/)
│   │   ├── app/
│   │   │   ├── core/               # Services, guards, interceptors
│   │   │   ├── Pages/              # Feature modules (lazy loaded)
│   │   │   └── shared/             # Componentes reutilizables
│   │   └── environments/           # environment.ts / environment.prod.ts
│   ├── Dockerfile                  # Build de produccion (multi-stage + nginx)
│   ├── nginx.conf                  # Reverse proxy + SPA fallback
│   ├── angular.json                # Config de Angular CLI
│   ├── tsconfig.json               # TypeScript config (con path aliases)
│   ├── GUIA-PROYECTO-BASE.md       # Guia de patrones del frontend
│   └── package.json                # Dependencias del workspace
│
├── bases/                          # Contenido y documentacion del dominio
├── docs/                           # Especificaciones de diseno
│
├── docker-compose.yml              # Orquestacion produccion (3 servicios)
├── package.json                    # Root: workspaces + scripts + deps compartidas
├── package-lock.json               # Lockfile unico para todo el monorepo
├── CLAUDE.md                       # Contexto para agentes IA
├── SETUP.md                        # Guia de instalacion rapida
└── GUIA-MONOREPO.md                # Este documento
```

**Puntos clave:**
- Cada workspace tiene su propio `package.json`, `tsconfig.json`, y herramientas de linting
- Un unico `package-lock.json` en la raiz (gestionado por npm workspaces)
- Cada workspace tiene su propio `Dockerfile` para builds independientes
- Docker Compose de desarrollo vive en `backend/`, el de produccion en la raiz

---

## 4. npm Workspaces

### 4.1 Configuracion de Workspaces

El root `package.json` define los workspaces:

```json
{
  "name": "mi-proyecto",
  "version": "0.0.1",
  "private": true,
  "workspaces": [
    "backend",
    "frontend"
  ]
}
```

**Como funciona:**
- `npm install` en la raiz instala las dependencias de TODOS los workspaces
- Se genera un unico `package-lock.json` en la raiz
- npm crea symlinks en `node_modules/` para cada workspace
- Las dependencias compartidas se "hoistean" al `node_modules/` de la raiz

### 4.2 Scripts Delegados (Root → Workspace)

Todos los comandos se ejecutan desde la raiz. El root `package.json` delega a cada workspace usando `--workspace=`:

```json
{
  "scripts": {
    "dev": "concurrently --names \"BACKEND,FRONTEND\" --prefix-colors \"cyan,magenta\" \"npm run backend:dev\" \"npm run frontend:start\"",
    "build": "npm run backend:build && npm run frontend:build",
    "test": "npm run backend:test && npm run frontend:test",

    "backend:dev":          "npm run start:dev       --workspace=backend",
    "backend:start":        "npm run start           --workspace=backend",
    "backend:build":        "npm run build           --workspace=backend",
    "backend:test":         "npm run test            --workspace=backend",
    "backend:test:e2e":     "npm run test:e2e        --workspace=backend",
    "backend:test:cov":     "npm run test:cov        --workspace=backend",
    "backend:lint":         "npm run lint            --workspace=backend",
    "backend:format":       "npm run format          --workspace=backend",
    "backend:db:generate":  "npm run db:generate     --workspace=backend",
    "backend:db:migrate":   "npm run db:migrate      --workspace=backend",
    "backend:db:seed":      "npm run db:seed         --workspace=backend",
    "backend:db:reset":     "npm run db:reset        --workspace=backend",
    "backend:docker:up":    "npm run docker:dev:up   --workspace=backend",
    "backend:docker:down":  "npm run docker:dev:down --workspace=backend",
    "backend:docker:reset": "npm run docker:dev:reset --workspace=backend",

    "frontend:start":      "npm run start            --workspace=frontend",
    "frontend:build":      "npm run build            --workspace=frontend",
    "frontend:build:prod": "npm run build:prod       --workspace=frontend",
    "frontend:test":       "npm run test             --workspace=frontend",
    "frontend:test:watch": "npm run test:watch       --workspace=frontend",
    "frontend:lint":       "npm run lint             --workspace=frontend"
  }
}
```

### 4.3 Convencion de Nombres de Scripts

| Patron | Ejemplo | Descripcion |
|--------|---------|-------------|
| `backend:*` | `backend:dev`, `backend:test` | Delega al workspace `backend` |
| `frontend:*` | `frontend:start`, `frontend:lint` | Delega al workspace `frontend` |
| Sin prefijo | `dev`, `build`, `test` | Ejecuta ambos workspaces (secuencial o paralelo) |
| `backend:db:*` | `backend:db:migrate` | Operaciones de base de datos (Prisma) |
| `backend:docker:*` | `backend:docker:up` | Operaciones Docker del desarrollo |

**Regla:** El script del root NO contiene logica — solo delega. La logica real vive en el `package.json` de cada workspace.

### 4.4 Agregar un Nuevo Workspace

Para agregar un tercer workspace (ej: una libreria compartida `shared`):

1. Crear el directorio y su `package.json`:
```bash
mkdir shared
cd shared
npm init -y
# Ajustar el name a algo como "@mi-proyecto/shared"
```

2. Agregar al array de workspaces en el root `package.json`:
```json
{
  "workspaces": [
    "backend",
    "frontend",
    "shared"
  ]
}
```

3. Agregar scripts delegados en el root:
```json
{
  "scripts": {
    "shared:build": "npm run build --workspace=shared",
    "shared:test": "npm run test --workspace=shared"
  }
}
```

4. Ejecutar `npm install` desde la raiz para que npm reconozca el nuevo workspace.

5. Para usar el workspace desde otro, importar por nombre de paquete (npm lo resuelve via symlink).

---

## 5. Gestion de Dependencias

### 5.1 Dependencias Compartidas (Root)

El root `package.json` contiene dependencias que se comparten entre workspaces:

```json
{
  "devDependencies": {
    "concurrently": "^9.0.0"
  },
  "dependencies": {
    // Aqui van las dependencias compartidas entre workspaces.
    // Ejemplo: si ambos workspaces usan la misma libreria de
    // utilidades, editor de texto, o SDK, se instala a nivel root
    // para garantizar la misma version en ambos.
  }
}
```

**`concurrently`** es el unico devDependency obligatorio del root — se usa para ejecutar backend y frontend en paralelo durante desarrollo.

**Dependencias compartidas:** Si una libreria se usa en multiples workspaces (ej: un editor de texto, un SDK, utilidades comunes), se instala a nivel root para garantizar consistencia de versiones. Al hoistear al root, npm resuelve la misma version para todos los workspaces.

### 5.2 Dependencias de Workspace

Cada workspace gestiona sus propias dependencias especificas:

- **Backend:** Framework (NestJS), ORM (Prisma), autenticacion (Passport, JWT), encriptacion (bcrypt), email, cloud SDKs
- **Frontend:** Framework (Angular), componentes UI (PrimeNG), utilidades CSS (TailwindCSS), librerias de UI especificas

Para instalar una dependencia en un workspace especifico:

```bash
# Instalar en el backend
npm install @nestjs/swagger --workspace=backend

# Instalar en el frontend
npm install primeng --workspace=frontend

# Instalar en el root (compartida)
npm install lodash
```

### 5.3 Cuando Hoistear vs Mantener en Workspace

| Criterio | Root (compartida) | Workspace (especifica) |
|----------|-------------------|------------------------|
| Usada por multiples workspaces | ✅ | — |
| Usada por un solo workspace | — | ✅ |
| Versionado debe ser identico entre workspaces | ✅ | — |
| Framework o herramienta del workspace | — | ✅ |
| Herramienta del monorepo (concurrently, lerna) | ✅ | — |

**Regla general:** En caso de duda, mantener en el workspace. Solo hoistear cuando hay una razon concreta (compartir version, evitar duplicacion).

---

## 6. Variables de Entorno y Configuracion

### 6.1 Archivos .env y su Alcance

| Archivo | Ubicacion | Usado por | Entorno |
|---------|-----------|-----------|---------|
| `backend/.env` | `backend/` | NestJS (ConfigModule) | Desarrollo local |
| `backend/.env.example` | `backend/` | Template — copiar a `.env` | — |
| `.env` (root) | Raiz | `docker-compose.yml` (produccion) | Produccion |

Ambos `.env` estan en `.gitignore`. Nunca se commitean secrets.

### 6.2 Flujo de Secrets por Entorno

```
DESARROLLO:
  backend/.env.example ──(copiar)──> backend/.env ──> NestJS ConfigModule
  frontend/src/environments/environment.ts ──> Angular (compilado en build)

PRODUCCION:
  .env (root) ──> docker-compose.yml (${VAR} substitution) ──> containers
  frontend/src/environments/environment.prod.ts ──> Angular (compilado en build:prod)
```

### 6.3 Variables del Backend

| Variable | Categoria | Descripcion | Ejemplo |
|----------|-----------|-------------|---------|
| `DATABASE_URL` | DB | Connection string PostgreSQL | `postgresql://user:pass@localhost:5432/db` |
| `JWT_SECRET` | Auth | Secret para firmar access tokens | String aleatorio largo |
| `JWT_EXPIRES_IN` | Auth | Duracion del access token | `1h` |
| `REFRESH_TOKEN_EXPIRES_IN_DAYS` | Auth | Duracion del refresh token | `7` |
| `PORT` | Server | Puerto del servidor NestJS | `3000` |
| `MAIL_SERVICE` | Email | Servicio de email | `gmail` |
| `SMTP_USER` | Email | Usuario SMTP | `correo@gmail.com` |
| `SMTP_PASS` | Email | Password SMTP / App Password | — |
| `MAIL_FROM` | Email | Remitente de emails | `"App" <correo@gmail.com>` |
| `PASSWORD_RESET_URL` | Auth | URL del frontend para reset de password | `http://localhost:4200/auth/new-password` |
| `RESET_PASSWORD_TOKEN_EXPIRATION` | Auth | Duracion del token de reset | `3600` (segundos) |
| `*_API_KEY` | Externo | API keys de servicios externos (IA, storage, etc.) | Segun el servicio |

**Nota:** Agregar variables segun los servicios externos que el proyecto necesite (IA, cloud storage, notificaciones push, etc.). Cada una debe estar en `.env.example` con un valor placeholder.

### 6.4 Variables del Frontend (environments/)

El frontend usa archivos TypeScript en vez de `.env`. Se compilan en build time.

**Desarrollo** (`frontend/src/environments/environment.ts`):
```typescript
export const environment = {
  production: false,
  apiBaseUrl: 'http://localhost:3000/api',
  devCredentials: {
    email: 'admin@example.com',
    password: 'password123',
  },
};
```

**Produccion** (`frontend/src/environments/environment.prod.ts`):
```typescript
export const environment = {
  production: true,
  apiBaseUrl: '/api',
};
```

**Diferencia clave:** En desarrollo, `apiBaseUrl` es una URL absoluta (`http://localhost:3000/api`) porque el frontend se comunica directamente con el backend. En produccion, es una ruta relativa (`/api`) porque nginx hace el proxying.

---

## 7. Estrategia Docker

### 7.1 Vision General: 3 Archivos Docker Compose

| Archivo | Ubicacion | Servicios | Uso |
|---------|-----------|-----------|-----|
| `docker-compose.dev.yml` | `backend/` | PostgreSQL unicamente | Desarrollo local diario |
| `docker-compose.yml` | `backend/` | PostgreSQL + Backend | Testing local del container backend |
| `docker-compose.yml` | Raiz | PostgreSQL + Backend + Frontend | Produccion (orquestacion completa) |

**Flujo tipico:**
- Desarrollo diario → `backend/docker-compose.dev.yml` (solo DB, app corre nativa con hot-reload)
- Probar el build del backend → `backend/docker-compose.yml` (backend containerizado + DB)
- Deployment completo → `docker-compose.yml` en la raiz (los 3 servicios)

### 7.2 Desarrollo: docker-compose.dev.yml (Solo PostgreSQL)

Levanta unicamente PostgreSQL para desarrollo local. El backend y frontend se ejecutan nativamente con Node.js.

```yaml
# backend/docker-compose.dev.yml
services:
  postgres-dev:
    image: postgres:16
    restart: unless-stopped
    environment:
      POSTGRES_USER: myuser
      POSTGRES_PASSWORD: mypassword
      POSTGRES_DB: mydb
    ports:
      - '5432:5432'              # Expuesto al host para acceso directo
    volumes:
      - postgres_dev_data:/var/lib/postgresql/data
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U myuser -d mydb']
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_dev_data:
```

**Puntos clave:**
- Puerto `5432` expuesto al host — el backend NestJS se conecta directamente
- Credenciales hardcodeadas (es solo para desarrollo local)
- Healthcheck para verificar que PostgreSQL este listo
- Volumen nombrado para persistir datos entre reinicios

**Comandos desde la raiz:**
```bash
npm run backend:docker:up     # Levantar PostgreSQL
npm run backend:docker:down   # Detener
npm run backend:docker:reset  # Detener y borrar volumen (reset completo)
```

### 7.3 Testing Local: backend/docker-compose.yml (Backend + PostgreSQL)

Levanta el backend containerizado junto con PostgreSQL. Util para verificar que el Dockerfile funciona antes de hacer deployment completo.

```yaml
# backend/docker-compose.yml
version: '3.9'
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: myuser
      POSTGRES_PASSWORD: mypassword
      POSTGRES_DB: mydb
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ['CMD-SHELL', 'pg_isready -U myuser -d mydb']
      interval: 10s
      timeout: 5s
      retries: 5

  backend:
    build: .
    ports:
      - '3000:3000'                               # Mismo puerto que desarrollo
    command: sh -c "npx prisma migrate deploy && node dist/src/main"
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DATABASE_URL: postgresql://myuser:mypassword@postgres:5432/mydb?schema=public
      JWT_SECRET: super_secret_key
      JWT_EXPIRES_IN: 1h
      REFRESH_TOKEN_EXPIRES_IN_DAYS: 7
      PORT: 3000

volumes:
  postgres_data:
```

**Diferencias con el docker-compose de produccion (raiz):**
- Credenciales hardcodeadas (no usa `.env`)
- Solo 2 servicios (sin frontend)
- Backend en puerto `3000:3000` (no `3002:3000`)
- Sin `restart: unless-stopped`

**Uso:**
```bash
cd backend
docker compose up --build     # Construir y levantar
docker compose down           # Detener
```

### 7.4 Produccion: docker-compose.yml (Orquestacion Completa)

Orquesta los 3 servicios para produccion. Todas las variables vienen del archivo `.env` en la raiz.

```yaml
# docker-compose.yml (raiz)
services:
  postgres:
    image: postgres:16
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${POSTGRES_USER}             # Desde .env
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

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - '3002:3000'                               # Host:3002 → Container:3000
    command: sh -c "npx prisma migrate deploy && node dist/src/main"
    depends_on:
      postgres:
        condition: service_healthy                 # Espera healthcheck de postgres
    environment:
      # --- Base de datos ---
      DATABASE_URL: ${DATABASE_URL}
      # --- Autenticacion ---
      JWT_SECRET: ${JWT_SECRET}
      JWT_EXPIRES_IN: ${JWT_EXPIRES_IN}
      REFRESH_TOKEN_EXPIRES_IN_DAYS: ${REFRESH_TOKEN_EXPIRES_IN_DAYS}
      PORT: 3000
      # --- Email ---
      MAIL_SERVICE: ${MAIL_SERVICE}
      SMTP_USER: ${SMTP_USER}
      SMTP_PASS: ${SMTP_PASS}
      MAIL_FROM: ${MAIL_FROM}
      PASSWORD_RESET_URL: ${PASSWORD_RESET_URL}
      # --- Servicios externos (agregar segun necesidad) ---
      # EJEMPLO_API_KEY: ${EJEMPLO_API_KEY}
    restart: unless-stopped

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - '3700:80'                                 # Host:3700 → nginx:80
    restart: unless-stopped

volumes:
  postgres_data:
```

**Patrones clave:**

1. **Auto-migracion en startup:** `command: sh -c "npx prisma migrate deploy && node dist/src/main"` ejecuta las migraciones pendientes antes de iniciar el servidor. Esto garantiza que la base de datos siempre este actualizada.

2. **Dependencia con healthcheck:** `depends_on.postgres.condition: service_healthy` asegura que el backend no inicie hasta que PostgreSQL este aceptando conexiones.

3. **PostgreSQL sin puerto expuesto:** En produccion, PostgreSQL solo es accesible desde la red interna de Docker (otros containers). No hay puerto mapeado al host.

4. **Puertos de produccion:** Backend en `3002`, Frontend en `3700`. Distintos a los de desarrollo para evitar conflictos.

### 7.5 Dockerfile del Backend (Single-stage)

```dockerfile
# backend/Dockerfile
FROM node:20-alpine
WORKDIR /app

# 1. Copiar package.json y prisma primero (aprovecha cache de Docker)
COPY package*.json ./
COPY prisma ./prisma
RUN npm install

# 2. Copiar config de TypeScript y codigo fuente
COPY tsconfig*.json nest-cli.json ./
COPY src ./src

# 3. Generar cliente Prisma y compilar NestJS
RUN npx prisma generate
RUN npx nest build

EXPOSE 3000
CMD ["node", "dist/src/main"]
```

**Patron de cache:** Las dependencias se instalan ANTES de copiar el codigo fuente. Si el codigo cambia pero las dependencias no, Docker reutiliza la capa cacheada de `npm install`.

**Sobre la ruta `dist/src/main`:** NestJS compila con `outDir: ./dist` y `sourceRoot: src` (nest-cli.json), generando la salida en `dist/src/`. Por eso el entry point es `dist/src/main`, no `dist/main`.

**Nota:** El `CMD` puede ser sobreescrito por el `command` del docker-compose (que agrega el `prisma migrate deploy` antes).

### 7.6 Dockerfile del Frontend (Multi-stage + Nginx)

```dockerfile
# frontend/Dockerfile

# Stage 1: Build — compila la app Angular con AOT
FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm install --legacy-peer-deps
COPY . .
RUN npm run build:prod

# Stage 2: Runtime — sirve los archivos estaticos con nginx
FROM nginx:alpine AS runtime
COPY --from=build /app/dist/mi-proyecto-frontend/browser /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

**Por que multi-stage:**
- La imagen final solo contiene nginx + los archivos HTML/JS/CSS compilados
- Node.js, node_modules, y el codigo fuente TypeScript NO estan en la imagen de produccion
- Resultado: imagen de ~30MB en vez de ~1GB

**`--legacy-peer-deps`:** Necesario por conflictos de peer dependencies entre librerias del ecosistema Angular.

### 7.7 Nginx: Reverse Proxy y SPA Fallback

```nginx
# frontend/nginx.conf
server {
    listen 80;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    # Proxy API requests to backend
    location /api/ {
        proxy_pass http://backend:3000/api/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_cache_bypass $http_upgrade;
    }

    # Angular SPA fallback
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

**Dos funciones criticas:**

1. **Reverse proxy para API:** Toda peticion a `/api/*` se reenvía al container `backend` en el puerto 3000. El nombre `backend` se resuelve por DNS interno de Docker Compose.

2. **SPA fallback:** Toda ruta que no sea un archivo estatico ni `/api/*` devuelve `index.html`. Esto permite que Angular Router maneje las rutas del lado del cliente (ej: `/platform/userManagement`).

**Headers de proxy:** WebSocket-ready (`Upgrade`, `Connection`) para potencial uso futuro de WebSockets. `X-Real-IP` preserva la IP del cliente original.

---

## 8. Comunicacion entre Servicios

### 8.1 En Desarrollo (Puertos Directos)

```
┌────────────┐    HTTP directa     ┌────────────┐    TCP directo    ┌────────────┐
│  Browser   │ ──────────────────> │  NestJS    │ ──────────────── │ PostgreSQL │
│            │                     │  :3000     │                  │  :5432     │
│            │ <── ng serve :4200  │            │                  │  (Docker)  │
└────────────┘                     └────────────┘                  └────────────┘
                                        ▲
                                        │
                              apiBaseUrl: 'http://localhost:3000/api'
```

- El frontend Angular corre en `:4200` (dev server de Angular CLI)
- El backend NestJS corre en `:3000` (Node.js nativo)
- PostgreSQL corre en `:5432` (Docker container)
- El frontend llama directamente al backend via `http://localhost:3000/api`
- CORS esta habilitado en el backend para permitir requests cross-origin desde `:4200`

### 8.2 En Produccion (Red Docker + Nginx Proxy)

```
                        Docker Network
┌────────────┐    ┌─────────────────────────────────────────────┐
│            │    │                                             │
│  Browser   │    │  ┌────────────┐         ┌────────────┐     │
│            │ ───┼─>│  nginx     │──/api/──>│  NestJS    │     │
│            │    │  │  :80 (:3700)│         │  :3000     │     │
│            │    │  │            │         │  (:3002)    │     │
│            │    │  │  Angular   │         │            │     │
│            │    │  │  static    │         └─────┬──────┘     │
│            │    │  └────────────┘               │            │
│            │    │                        ┌──────▼─────┐      │
│            │    │                        │ PostgreSQL │      │
│            │    │                        │  :5432     │      │
│            │    │                        │ (internal) │      │
└────────────┘    └────────────────────────┴────────────┴──────┘
```

- El browser accede a nginx en el puerto `:3700`
- nginx sirve los archivos estaticos de Angular
- Las peticiones a `/api/*` se proxyean internamente a `backend:3000`
- PostgreSQL NO tiene puerto expuesto al host — solo accesible dentro de la red Docker
- `apiBaseUrl: '/api'` — ruta relativa, nginx se encarga del routing

### 8.3 Diferencia Clave: CORS

| Entorno | CORS necesario | Razon |
|---------|---------------|-------|
| Desarrollo | Si | Frontend (:4200) y Backend (:3000) estan en origenes distintos |
| Produccion | No | Todo pasa por nginx (:80), mismo origen |

---

## 9. Workflow de Desarrollo

### 9.1 Setup Inicial (Primera Vez)

```bash
# 1. Clonar e instalar dependencias (todos los workspaces)
git clone <repo-url>
cd mi-proyecto
npm install

# 2. Configurar variables de entorno del backend
cp backend/.env.example backend/.env
# Editar backend/.env con las credenciales reales

# 3. Levantar PostgreSQL
npm run backend:docker:up

# 4. Generar cliente Prisma, ejecutar migraciones y seed
npm run backend:db:generate
npm run backend:db:migrate
npm run backend:db:seed

# 5. Iniciar desarrollo
npm run dev
```

Para instrucciones detalladas, ver [`SETUP.md`](SETUP.md).

### 9.2 Desarrollo Diario

```bash
# Levantar PostgreSQL (si no esta corriendo)
npm run backend:docker:up

# Iniciar ambos servicios en paralelo
npm run dev
# Output:
# [BACKEND] Nest application successfully started
# [FRONTEND] Angular Live Development Server is listening on localhost:4200
```

El comando `dev` usa `concurrently` para ejecutar backend y frontend en paralelo con prefijos coloreados (`BACKEND` en cyan, `FRONTEND` en magenta).

### 9.3 Puertos y URLs

| Servicio | Entorno | Puerto | URL |
|----------|---------|--------|-----|
| Frontend | dev (ng serve) | 4200 | `http://localhost:4200` |
| Backend | dev (nest start) | 3000 | `http://localhost:3000` |
| Swagger API Docs | dev | 3000 | `http://localhost:3000/api/docs` |
| PostgreSQL | dev (docker-compose.dev) | 5432 | `postgresql://myuser:mypassword@localhost:5432/mydb` |
| Backend | test local (backend/docker-compose) | 3000 | `http://localhost:3000` |
| Backend | produccion (root docker-compose) | 3002 | `http://localhost:3002` |
| Frontend | produccion (root docker-compose) | 3700 | `http://localhost:3700` |

### 9.4 Base de Datos (Prisma CLI)

Todos los comandos se ejecutan desde la raiz:

| Comando | Descripcion |
|---------|-------------|
| `npm run backend:db:generate` | Regenerar el cliente Prisma (despues de cambiar el schema) |
| `npm run backend:db:migrate` | Crear y ejecutar migraciones (desarrollo) |
| `npm run backend:db:seed` | Poblar la base de datos con datos iniciales |
| `npm run backend:db:reset` | Resetear la base de datos (borra todo, re-ejecuta migraciones y seed) |

**Flujo tipico al modificar el schema:**
1. Editar `backend/prisma/schema.prisma`
2. `npm run backend:db:migrate` — crea la migracion y actualiza la DB
3. `npm run backend:db:generate` — regenera el cliente Prisma (si no lo hizo el paso anterior)

---

## 10. Deployment a Produccion

### 10.1 Build de Imagenes Docker

Docker Compose construye las imagenes automaticamente desde los Dockerfiles de cada workspace:

```bash
# Construir y levantar todos los servicios
docker compose up -d --build

# Solo construir sin levantar
docker compose build
```

**Flujo de build:**
1. PostgreSQL: usa imagen oficial `postgres:16` (no se construye)
2. Backend: `backend/Dockerfile` — instala deps, genera Prisma, compila NestJS
3. Frontend: `frontend/Dockerfile` — instala deps, compila Angular, copia a nginx

### 10.2 Migraciones Automaticas

El backend ejecuta migraciones automaticamente al iniciar via el `command` del docker-compose:

```yaml
command: sh -c "npx prisma migrate deploy && node dist/src/main"
```

- `prisma migrate deploy` aplica migraciones pendientes sin crear nuevas
- Solo si todas las migraciones pasan, inicia el servidor NestJS
- Si una migracion falla, el container se reinicia (por `restart: unless-stopped`)

**Importante:** `prisma migrate deploy` es distinto a `prisma migrate dev`. Deploy es seguro para produccion — solo aplica migraciones existentes, no genera nuevas ni toca el schema.

### 10.3 Orquestacion con Docker Compose

```bash
# Levantar en produccion
docker compose up -d

# Ver logs de todos los servicios
docker compose logs -f

# Ver logs de un servicio especifico
docker compose logs -f backend

# Detener todo
docker compose down

# Detener y borrar volumenes (CUIDADO: borra la base de datos)
docker compose down -v

# Rebuild y redeploy
docker compose up -d --build
```

### 10.4 Healthchecks y Dependencias entre Servicios

```
postgres (healthcheck: pg_isready)
    │
    ▼ condition: service_healthy
backend (auto-migration → start)
    │
    ▼ (sin dependencia explicita, pero necesita backend para /api/)
frontend (nginx + static files)
```

PostgreSQL tiene un healthcheck que ejecuta `pg_isready` cada 10 segundos. El backend NO inicia hasta que PostgreSQL reporte "healthy" (maximo 5 reintentos × 10s = 50s de espera).

El frontend no tiene dependencia explicita del backend porque nginx manejara los requests a `/api/` — si el backend no esta listo, nginx devolvera un 502 que el frontend puede manejar.

---

## 11. Testing

### 11.1 Backend: Jest

| Comando | Descripcion |
|---------|-------------|
| `npm run backend:test` | Ejecutar todos los tests unitarios |
| `npm run backend:test:cov` | Tests con reporte de cobertura |
| `npm run backend:test:e2e` | Tests end-to-end (Supertest) |

Configuracion en `backend/package.json`. Patron de archivos: `**/*.spec.ts`.

Para patrones de testing del backend, ver la seccion 12 de [`backend/PROJECT-PATTERNS-GUIDE.md`](backend/PROJECT-PATTERNS-GUIDE.md).

### 11.2 Frontend: Vitest

| Comando | Descripcion |
|---------|-------------|
| `npm run frontend:test` | Ejecutar tests una vez |
| `npm run frontend:test:watch` | Ejecutar en modo watch |

Usa `@analogjs/vitest-angular` como bridge entre Vitest y Angular TestBed.

### 11.3 Ejecutar Tests Combinados

```bash
# Ejecutar tests de ambos workspaces secuencialmente
npm run test
# Equivale a: npm run backend:test && npm run frontend:test
```

Los tests se ejecutan secuencialmente (no en paralelo) para evitar conflictos de recursos y hacer el output mas legible.

---

## 12. Linting y Formateo

Cada workspace gestiona su propio linting. No hay configuracion de linting a nivel root.

| Workspace | Herramienta | Comando | Config |
|-----------|------------|---------|--------|
| Backend | ESLint + Prettier | `npm run backend:lint` / `backend:format` | `.eslintrc.js` + `.prettierrc` |
| Frontend | Angular ESLint | `npm run frontend:lint` | `angular.json` (lint target) |

**Configuracion del Backend:**
- ESLint: `@typescript-eslint/recommended` + `plugin:prettier/recommended`
- Prettier: single quotes, trailing commas
- Reglas relajadas: `no-explicit-any` y `no-explicit-function-return-type` desactivadas

**Configuracion del Frontend:**
- Angular ESLint integrado via `ng lint`
- TypeScript strict mode habilitado (`strict: true`, `strictTemplates: true`)

---

## 13. Agregar Nuevas Features End-to-End

### 13.1 Nuevo Modulo Backend + Endpoint

1. Crear folder del modulo en `backend/src/`:
```
backend/src/mi-feature/
├── mi-feature.module.ts
├── mi-feature.controller.ts
├── mi-feature.service.ts
├── dto/
│   ├── mi-feature.req.dto.ts
│   └── mi-feature.res.dto.ts
```

2. Registrar el modulo en `backend/src/app.module.ts`

3. Agregar modelo en `backend/prisma/schema.prisma` (si aplica)

4. Ejecutar migracion: `npm run backend:db:migrate`

5. Agregar permisos al seed: `backend/prisma/seed.ts`

Para patrones detallados de controllers, servicios, DTOs y decoradores, ver [`backend/PROJECT-PATTERNS-GUIDE.md`](backend/PROJECT-PATTERNS-GUIDE.md).

### 13.2 Nueva Pagina Frontend + Ruta

1. Crear componente en `frontend/src/app/Pages/platform/`:
```
frontend/src/app/Pages/platform/mi-feature-management/
├── mi-feature-management.ts
├── mi-feature-management.html
└── mi-feature-management.sass
```

2. Agregar ruta lazy-loaded en `frontend/src/app/Pages/platform/platform.routes.ts`:
```typescript
{
  path: 'miFeatureManagement',
  loadComponent: () =>
    import('./mi-feature-management/mi-feature-management')
      .then((m) => m.MiFeatureManagement),
  canActivate: [permissionGuard],
  data: { permissions: ['mi-feature:read'] },
}
```

3. Agregar item de menu en el sidebar del layout

Para patrones detallados de componentes, servicios y routing, ver [`frontend/GUIA-PROYECTO-BASE.md`](frontend/GUIA-PROYECTO-BASE.md).

### 13.3 Conectar Frontend con Backend

1. Crear servicio en `frontend/src/app/core/services/`:
```
frontend/src/app/core/services/miFeatureService/
├── mi-feature.service.ts
├── mi-feature.req.dto.ts
└── mi-feature.res.dto.ts
```

2. El servicio usa `HttpPromiseBuilderService` para llamar al endpoint:
```typescript
@Injectable({ providedIn: 'root' })
export class MiFeatureService {
  private readonly baseUrl = `${environment.apiBaseUrl}/mi-feature`;

  constructor(private readonly httpBuilder: HttpPromiseBuilderService) {}

  async getAll(): Promise<MiFeatureResponse[]> {
    return this.httpBuilder
      .request<MiFeatureResponse[]>()
      .get()
      .url(this.baseUrl)
      .send();
  }
}
```

3. Inyectar en el componente con `inject()`:
```typescript
private readonly miFeatureService = inject(MiFeatureService);
```

### 13.4 Flujo de Permisos End-to-End

```
1. SEED (backend)
   └─ Definir permission { codigo: 'mi-feature:read', descripcion: '...' }
   └─ Asignar a roles correspondientes

2. BACKEND
   └─ Controller: @Permission('mi-feature:read') en cada endpoint

3. FRONTEND - Rutas
   └─ platform.routes.ts: data: { permissions: ['mi-feature:read'] }
   └─ permissionGuard valida antes de cargar el componente

4. FRONTEND - UI
   └─ Template: *hasPermission="'mi-feature:update'" para mostrar/ocultar botones
   └─ Logica: permissionCheckService.hasPermission('mi-feature:delete')
```

**Formato de permisos:** `recurso:accion` (ej: `users:read`, `roles:update`, `projects:delete`).

---

## 14. Directorios Auxiliares

### 14.1 bases/ — Contenido del Dominio

Contiene documentacion y contenido de referencia del dominio del proyecto. **No es un workspace de codigo** — es material de consulta que alimenta la logica de negocio de la aplicacion.

Puede incluir: glosarios, definiciones de entidades, reglas de negocio, taxonomias, o cualquier conocimiento del dominio que necesite estar versionado junto al codigo.

### 14.2 docs/ — Especificaciones de Diseno

Contiene documentos de diseno para features especificas:
```
docs/
└── superpowers/
    └── specs/
        └── YYYY-MM-DD-nombre-feature-design.md
```

**Convencion de nombres:** Fecha ISO + nombre descriptivo + `-design.md`.

### 14.3 .claude/ y .agents/ — Configuracion de IA

- **`.claude/`**: Configuracion de Claude Code (settings, skills, MCP servers)
- **`.agents/`**: Definiciones de skills para agentes IA (brainstorming, frontend-design)
- **`.mcp.json`**: Configuracion de MCP servers para herramientas de desarrollo

Estos directorios son configuracion de herramientas de desarrollo, no parte de la aplicacion.

---

## 15. Referencias Cruzadas

| Tema | Guia | Ruta |
|------|------|------|
| Estructura del monorepo, workspaces, Docker | Este documento | `GUIA-MONOREPO.md` |
| Patrones Angular: componentes, servicios, routing, auth, permisos | Guia Frontend | `frontend/GUIA-PROYECTO-BASE.md` |
| Patrones NestJS: modulos, DTOs, controllers, RBAC, Prisma, testing | Guia Backend | `backend/PROJECT-PATTERNS-GUIDE.md` |
| Instalacion y primera ejecucion | Setup | `SETUP.md` |
| Contexto general del proyecto para IA | Claude | `CLAUDE.md` |
| Schema de base de datos | Prisma Schema | `backend/prisma/schema.prisma` |
| Documentacion de API (runtime) | Swagger | `http://localhost:3000/api/docs` |

### Orden de Lectura Recomendado para Agentes IA

1. **`CLAUDE.md`** — Entender el proyecto (dominio, stack, estructura general)
2. **`GUIA-MONOREPO.md`** — Entender la infraestructura (workspaces, Docker, deployment)
3. **`frontend/GUIA-PROYECTO-BASE.md`** o **`backend/PROJECT-PATTERNS-GUIDE.md`** — Segun donde vayas a trabajar
4. **`SETUP.md`** — Solo si necesitas levantar el entorno desde cero
