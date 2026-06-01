# SETUP — Instalacion y primera ejecucion

Guia paso a paso para levantar SkillMaker en desarrollo local desde cero. Tiempo estimado: 20-30 minutos (la mayor parte es configurar Google OAuth la primera vez).

> Para entender la arquitectura general antes de levantar el entorno, leer [`GUIA-MONOREPO.md`](GUIA-MONOREPO.md). Para patrones internos de cada workspace, ver [`backend/GUIA-BACK.md`](backend/GUIA-BACK.md) y [`frontend/GUIA-FRONT.md`](frontend/GUIA-FRONT.md).

---

## 1. Pre-requisitos

Instalar una vez en la maquina:

| Herramienta | Version minima | Verificacion |
|-------------|----------------|--------------|
| **Go** | 1.23 | `go version` |
| **Node.js** | 20 | `node --version` |
| **npm** | 10 | `npm --version` |
| **Docker + Docker Compose** | reciente | `docker --version` && `docker compose version` |
| **Make** | GNU make | `make --version` *(en Windows usar WSL2)* |
| **Git** | reciente | `git --version` |

### Herramientas Go (instaladas con `go install`)

```bash
# Migrate CLI (migraciones reproducibles)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Air (hot reload del backend)
go install github.com/air-verse/air@latest

# Swag (generacion de OpenAPI desde anotaciones)
go install github.com/swaggo/swag/cmd/swag@latest

# golangci-lint (linter unificado) — instalacion recomendada:
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.62.0
```

Verificar que todos esten en el PATH:

```bash
migrate -version && air -v && swag --version && golangci-lint --version
```

Si `$(go env GOPATH)/bin` no esta en el PATH, agregarlo a `~/.zshrc` / `~/.bashrc`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

---

## 2. Clonar y instalar dependencias

```bash
git clone <repo-url> skillmaker
cd skillmaker

make install
# equivale a:
#   cd backend  && go mod download
#   cd frontend && npm install --legacy-peer-deps
```

---

## 3. Configurar Google OAuth (15 minutos)

Esta es la parte mas larga. El sistema **delega toda la autenticacion en Google Workspace** (RT-12), por eso necesitas crear un OAuth Client ID antes de poder loguearte.

### 3.1. Crear el proyecto en Google Cloud Console

1. Ir a [console.cloud.google.com](https://console.cloud.google.com).
2. Crear un proyecto nuevo (o usar uno existente del workspace de tu empresa).
3. En el menu lateral: **APIs & Services** → **OAuth consent screen**.
4. Tipo de usuario: **Internal** (restringe acceso al dominio de Google Workspace de la empresa — RT-13).
5. Completar la pantalla de consentimiento con nombre de la app, email de soporte y dominios autorizados.

### 3.2. Crear el OAuth Client ID

1. **APIs & Services** → **Credentials** → **Create Credentials** → **OAuth client ID**.
2. Tipo: **Web application**.
3. **Authorized JavaScript origins** (necesario para Google Identity Services):
   - `http://localhost:4200`
   - `http://localhost:3000`
4. **Authorized redirect URIs** (solo si usas code flow; con ID token flow no es necesario):
   - `http://localhost:3000/api/auth/google/callback`
5. Click **Create** → copiar el **Client ID** y el **Client Secret**.

### 3.3. Obtener el "hosted domain"

Es el dominio corporativo del workspace de Google (ej: `tuempresa.com`). El backend lo valida en cada login para asegurar que solo cuentas de la empresa entren (RT-13, defensa en profundidad ante manipulacion del frontend).

---

## 4. Configurar variables de entorno del backend

```bash
cp backend/.env.example backend/.env
```

Editar `backend/.env` y completar al menos estas variables criticas:

```bash
# ─── Database ─────────────────────────────────────────────
DATABASE_URL=postgres://skillmaker:skillmaker@localhost:5432/skillmaker?sslmode=disable

# ─── Auth ─────────────────────────────────────────────────
JWT_SECRET=cambia-esto-por-un-string-aleatorio-largo-min-32-chars
JWT_EXPIRES_IN=1h
REFRESH_TOKEN_EXPIRES_IN=168h

GOOGLE_CLIENT_ID=xxxxxxxxxxxxx.apps.googleusercontent.com    # del paso 3.2
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxx                     # del paso 3.2
GOOGLE_HOSTED_DOMAIN=tuempresa.com                            # del paso 3.3
GOOGLE_REDIRECT_URI=http://localhost:3000/api/auth/google/callback

# ─── Storage (MinIO local) ────────────────────────────────
STORAGE_ENDPOINT=http://localhost:9000
STORAGE_REGION=us-east-1
STORAGE_BUCKET=skillmaker-materials
STORAGE_ACCESS_KEY=minioadmin
STORAGE_SECRET_KEY=minioadmin
STORAGE_USE_SSL=false
STORAGE_PRESIGN_TTL=15m
MAX_UPLOAD_BYTES=52428800

# ─── Server ───────────────────────────────────────────────
APP_ENV=development
PORT=3000
LOG_LEVEL=debug
ALLOWED_ORIGINS=http://localhost:4200
```

**Generar un `JWT_SECRET` seguro:**

```bash
openssl rand -base64 48
```

Copiar el resultado en `JWT_SECRET=...`.

---

## 5. Configurar variables del frontend

Editar `frontend/src/environments/environment.ts`:

```typescript
export const environment = {
  production: false,
  apiBaseUrl: 'http://localhost:3000/api',
  googleClientId: 'xxxxxxxxxxxxx.apps.googleusercontent.com',  // mismo del paso 3.2
  googleHostedDomain: 'tuempresa.com',                          // mismo del paso 3.3
};
```

> Nota: `googleClientId` es **publico** por diseno del flujo OAuth (el frontend lo expone al iniciar el popup). El `GOOGLE_CLIENT_SECRET` vive solo en el backend.

---

## 6. Levantar dependencias (PostgreSQL + MinIO)

```bash
make db-up
```

Esto levanta dos contenedores:
- **PostgreSQL** en `:5432` con usuario/db `skillmaker`
- **MinIO** en `:9000` (API S3-compatible) y `:9001` (consola web)

El job `minio-init` se ejecuta una vez para crear el bucket `skillmaker-materials`.

Verificar que esten arriba:

```bash
docker compose -f backend/docker-compose.dev.yml ps
```

Acceder a la consola de MinIO en [http://localhost:9001](http://localhost:9001) (user: `minioadmin` / pass: `minioadmin`) para confirmar que el bucket fue creado.

---

## 7. Aplicar migraciones y seed

```bash
make db-migrate     # crea las tablas (user, role, course, evaluation, ...)
make db-seed        # crea los 4 roles fijos del dominio
```

El seed crea: `alumno`, `creador`, `supervisor`, `administrador`. Cualquier usuario que se loguee con Google recibe automaticamente el rol `alumno`; los demas roles los asigna un administrador desde la UI.

**Verificar:**

```bash
docker compose -f backend/docker-compose.dev.yml exec postgres-dev \
  psql -U skillmaker -d skillmaker -c "SELECT nombre FROM role;"
```

Debe listar los 4 roles.

---

## 8. Generar el spec OpenAPI

```bash
make swagger
```

Esto crea `backend/docs/swagger.json`, `swagger.yaml` y `docs.go` a partir de las anotaciones swag en los handlers.

> **Importante:** repetir este paso despues de agregar o modificar endpoints. El frontend puede generar tipos TypeScript desde este spec con:
> ```bash
> npx openapi-typescript backend/docs/swagger.json -o frontend/src/app/api/types.ts
> ```

---

## 9. Promoverte a administrador (primera vez)

Tras el primer login con Google, tu usuario tendra solo el rol `alumno`. Para acceder a las vistas de administracion, necesitas promoverte manualmente UNA vez:

```bash
docker compose -f backend/docker-compose.dev.yml exec postgres-dev \
  psql -U skillmaker -d skillmaker -c "
INSERT INTO user_role (user_id, role_id)
SELECT u.id, r.id
FROM \"user\" u, role r
WHERE u.email = 'tu-email@tuempresa.com'
  AND r.nombre IN ('creador', 'administrador')
ON CONFLICT DO NOTHING;
"
```

Reemplazar `tu-email@tuempresa.com` con tu email corporativo. Tras esto, hacer logout y volver a loguearte para que el JWT contenga los roles nuevos.

---

## 10. Iniciar desarrollo

```bash
make dev
```

Esto arranca en paralelo:
- **Backend** con [Air](https://github.com/air-verse/air) (hot reload de Go)
- **Frontend** con `ng serve` (HMR de Angular)

Output esperado:

```
[BACKEND]  watching .
[BACKEND]  building...
[BACKEND]  running...
[BACKEND]  servidor escuchando port=3000
[FRONTEND] Angular Live Development Server is listening on localhost:4200
```

---

## 11. Verificar que todo funciona

| Check | URL / Comando | Resultado esperado |
|-------|---------------|---------------------|
| Frontend carga | http://localhost:4200 | Pantalla de login con boton "Continuar con Google" |
| Health del backend | http://localhost:3000/api/health | `{"status":"ok"}` |
| Readiness (DB + storage) | http://localhost:3000/api/health/ready | `{"status":"ready"}` |
| Swagger UI | http://localhost:3000/api/docs/index.html | UI interactiva con endpoints |
| Login con Google | Click "Continuar con Google" en /auth/login | Redirige a /platform/catalog tras autenticarse |
| MinIO Console | http://localhost:9001 | Bucket `skillmaker-materials` visible |

---

## Troubleshooting

### "permission denied" al instalar Air, swag o golang-migrate

`go install` necesita poder escribir en `$(go env GOPATH)/bin`. Asegurate de que ese directorio existe y es escribible:

```bash
mkdir -p $(go env GOPATH)/bin
```

Y que el PATH lo incluye (`echo $PATH | grep -o "$(go env GOPATH)/bin"`).

### Google Login devuelve "popup_closed" o "idpiframe_initialization_failed"

- Verificar que el `GOOGLE_CLIENT_ID` del frontend y del backend sean **identicos**.
- Verificar que `http://localhost:4200` este en **Authorized JavaScript origins** del Client ID.
- Si la cuenta que intenta loguearse NO pertenece al `googleHostedDomain` configurado, Google bloquea el popup. Usar una cuenta del dominio corporativo.

### Backend devuelve 401 "cuenta no pertenece al dominio corporativo"

El email de Google no termina en `GOOGLE_HOSTED_DOMAIN`. Esto es deliberado (RT-13). Solo cuentas del workspace de la empresa pueden entrar.

### `migrate` dice "no change" pero esperaba aplicar algo

`golang-migrate` mantiene un registro en la tabla `schema_migrations`. Para empezar de cero:

```bash
make db-reset       # borra el volumen, vuelve a levantar Postgres
make db-migrate     # aplica todas las migraciones de nuevo
make db-seed
```

### Air no recompila al cambiar un archivo

Verificar que `.air.toml` en `backend/` tenga el `include_ext` correcto (al menos `.go`). Si seguis con problemas, parar Air, borrar `backend/tmp/`, y volver a correr `make backend-dev`.

### `make swagger` falla con "swag command not found"

`swag` no esta en el PATH. Ver seccion 1 — instalacion de herramientas Go.

### `npm install` falla con conflictos de peer dependencies

Usar `--legacy-peer-deps` (ya esta incluido en `make install`). Si lo corres a mano:

```bash
cd frontend && npm install --legacy-peer-deps
```

---

## Siguientes pasos

- Leer [`GUIA-MONOREPO.md`](GUIA-MONOREPO.md) para entender la estructura del monorepo.
- Si vas a trabajar en backend: [`backend/GUIA-BACK.md`](backend/GUIA-BACK.md).
- Si vas a trabajar en frontend: [`frontend/GUIA-FRONT.md`](frontend/GUIA-FRONT.md).
- Para el dominio y los requerimientos: [`bases/documentacion/`](bases/documentacion/).
