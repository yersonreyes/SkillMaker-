# Smoke Tests — scaffold-backend-base

Manual verification recipes for the SCAFFOLD-RF-* and SCAFFOLD-RNF-* requirements.
Run these after `make install && make db-up && make db-migrate && make db-seed && make backend-dev`.

## Pre-requisitos

- Backend corriendo en :3000 (`make backend-dev` desde la raiz del monorepo)
- PostgreSQL en :5432 (`make db-up`)
- MinIO en :9000 (`make db-up`)
- `psql` cliente disponible
- `jq` disponible (`apt install jq` o `brew install jq`)

---

## SCAFFOLD-RF-01 — Health endpoint

```bash
curl -s http://localhost:3000/api/health | jq
# Expected: { "status": "ok" }
```

---

## SCAFFOLD-RF-02 — Readiness endpoint

```bash
curl -s http://localhost:3000/api/health/ready | jq
# Expected (todo OK):    HTTP 200 + { "status": "ok" }
# Expected (algo down):  HTTP 503 + { "status": "degraded", "issues": ["db"] }
```

Para probar el caso de fallo:

```bash
make db-down
curl -s -w '\nHTTP %{http_code}\n' http://localhost:3000/api/health/ready
# Expected: HTTP 503 + { "status": "degraded", "issues": ["db"] }
make db-up
```

---

## SCAFFOLD-RF-03 — Login con Google

```bash
# Requiere ID token real de Google del dominio configurado.
# Obtener uno con Google Identity Services en el frontend, o con OAuth Playground.
curl -X POST http://localhost:3000/api/auth/google \
  -H 'Content-Type: application/json' \
  -d '{"idToken":"<paste-google-id-token-here>"}' | jq

# Expected (ok):                200 + { access_token, refresh_token, expires_at, user }
# Expected (idToken invalido):  401 + { code: "INVALID_GOOGLE_TOKEN", message }
# Expected (hd mismatch RT-13): 401 + { code: "UNAUTHORIZED_DOMAIN", message }
```

> **Contrato de error codes (consumir desde el frontend):**
> - `INVALID_GOOGLE_TOKEN` — el ID token de Google no validó (firma o audience incorrecta)
> - `UNAUTHORIZED_DOMAIN` — el `hd` claim no coincide con `GOOGLE_HOSTED_DOMAIN`
> - `INVALID_REFRESH` — refresh token inválido, expirado, revocado, o reutilizado tras rotación

---

## SCAFFOLD-RF-04 — Refresh rota el token

```bash
# Tras un login exitoso, exportar el token de acceso y refresh
export IDTOKEN="<tu-google-id-token>"

LOGIN=$(curl -s -X POST http://localhost:3000/api/auth/google \
  -H 'Content-Type: application/json' \
  -d "{\"idToken\":\"$IDTOKEN\"}")

export R1=$(echo "$LOGIN" | jq -r '.refresh_token')
export ACCESS=$(echo "$LOGIN" | jq -r '.access_token')

# Rotar el refresh token
REFRESH_RESP=$(curl -s -X POST http://localhost:3000/api/auth/refresh \
  -H 'Content-Type: application/json' \
  -d "{\"refreshToken\":\"$R1\"}")

export R2=$(echo "$REFRESH_RESP" | jq -r '.refresh_token')

# Verificar que R2 != R1
[ "$R1" != "$R2" ] && echo "✅ rotation OK" || echo "❌ rotation FAILED"

# Verificar parent_id en DB (R2 debe apuntar al id de R1)
docker compose -f backend/docker-compose.dev.yml exec postgres-dev \
  psql -U skillmaker -d skillmaker \
  -c "SELECT id, parent_id IS NOT NULL as has_parent FROM refresh_token ORDER BY created_at DESC LIMIT 3;"
```

---

## SCAFFOLD-RF-04b — Reuse detection (OWASP)

```bash
# Reusar R1 (ya rotado) debe revocar TODAS las sesiones del usuario
curl -s -X POST http://localhost:3000/api/auth/refresh \
  -H 'Content-Type: application/json' \
  -d "{\"refreshToken\":\"$R1\"}" | jq
# Expected: 401 con { "code": "INVALID_REFRESH" }

# Verificar que TODOS los refresh tokens del usuario quedan revoked_at != NULL
docker compose -f backend/docker-compose.dev.yml exec postgres-dev \
  psql -U skillmaker -d skillmaker \
  -c "SELECT COUNT(*) FROM refresh_token WHERE revoked_at IS NULL;"
# Expected: 0
```

---

## SCAFFOLD-RF-05 — Logout idempotente

```bash
# Logout con R2 (el token activo tras la rotacion)
curl -s -X POST http://localhost:3000/api/auth/logout \
  -H 'Content-Type: application/json' \
  -d "{\"refreshToken\":\"$R2\"}" -i
# Expected: HTTP 204 No Content

# Repetir — debe seguir devolviendo 204 (idempotente)
curl -s -X POST http://localhost:3000/api/auth/logout \
  -H 'Content-Type: application/json' \
  -d "{\"refreshToken\":\"$R2\"}" -i
# Expected: HTTP 204 No Content
```

---

## SCAFFOLD-RF-06 — Middleware JWT rechaza token expirado

```bash
# Por ahora no hay endpoints protegidos cableados en este scaffold.
# El middleware JWT (internal/middleware/jwt.go) esta implementado y listo.
# Verificacion completa pendiente hasta que se cableen endpoints protegidos
# en cambios posteriores (courses, evaluations, etc.).

# Prueba manual con token invalido usando un endpoint futuro:
curl -s http://localhost:3000/api/<endpoint-protegido> \
  -H 'Authorization: Bearer token.invalido.aqui' | jq
# Expected: 401
```

---

## SCAFFOLD-RF-07 — Seed idempotente

```bash
# Primer seed
make db-seed
# Expected: exit 0, log "roles sembrados"

# Segundo seed (idempotente)
make db-seed
# Expected: exit 0, mismo resultado sin duplicados

# Verificar en DB
docker compose -f backend/docker-compose.dev.yml exec postgres-dev \
  psql -U skillmaker -d skillmaker \
  -c "SELECT id, nombre FROM role ORDER BY id;"
# Expected: 4 filas: alumno, creador, supervisor, administrador
```

---

## SCAFFOLD-RF-08 — Migraciones up/down

```bash
# Down: revertir migracion
make db-migrate-down
docker compose -f backend/docker-compose.dev.yml exec postgres-dev \
  psql -U skillmaker -d skillmaker -c "\dt"
# Expected: solo schema_migrations (o ninguna tabla si down revirtio todo)

# Up: reaplicar migracion
make db-migrate
docker compose -f backend/docker-compose.dev.yml exec postgres-dev \
  psql -U skillmaker -d skillmaker -c "\dt"
# Expected: schema_migrations, role, "user", user_role, refresh_token
```

---

## SCAFFOLD-RNF-01 — End-to-end make dev

```bash
# Desde la raiz del monorepo (skillmaker/)
make install
make db-up
make db-migrate
make db-seed
make backend-dev
# Expected: logs "[BACKEND] servidor escuchando port=3000"
# Frontend stub warnea "⚠️  frontend/ no scaffoldeado..." pero NO rompe el flujo
```

---

## SCAFFOLD-RNF-02 — Fail-fast con env vars faltantes

```bash
# Desde backend/
mv backend/.env backend/.env.bak
cd backend && go run ./cmd/api
# Expected: exit code != 0 con mensaje sobre variable requerida faltante
mv backend/.env.bak backend/.env
```

---

## SCAFFOLD-RNF-03 — Distroless build

```bash
cd backend && docker build -t skillmaker-backend:test .

# Verificar tamano de imagen
docker image inspect skillmaker-backend:test --format='{{.Size}}' | \
  awk '{printf "%.1f MB\n", $1/1024/1024}'
# Expected: < 50MB (tipicamente ~20-30MB)

# Verificar que NO tiene shell (distroless)
docker run --rm skillmaker-backend:test sh 2>&1 || true
# Expected: error porque distroless NO tiene sh
```

---

## SCAFFOLD-RNF-04 — Structured logging

```bash
# JSON en produccion
cd backend && APP_ENV=production go run ./cmd/api 2>&1 | head -3
# Expected: cada linea es JSON valido con request_id

# Text en desarrollo
cd backend && APP_ENV=development go run ./cmd/api 2>&1 | head -3
# Expected: texto legible con time, level, msg
```

---

## SCAFFOLD-RNF-05 — Swagger UI

```bash
# Con el backend corriendo:
curl -s http://localhost:3000/api/docs/index.html | head -5
# Expected: responde con HTML de la Swagger UI

# O abrir en el navegador:
# http://localhost:3000/api/docs/index.html
# Expected: UI interactiva con los endpoints /auth/google, /auth/refresh, /auth/logout
```

---

## SCAFFOLD-RNF-06 — Graceful shutdown

```bash
# Terminal 1: levantar backend
make backend-dev

# Terminal 2: enviar SIGTERM al proceso Air
kill -TERM $(pgrep -f 'tmp/api') 2>/dev/null || kill -TERM $(pgrep -f 'go run')
# Expected en terminal 1: logs "apagando servidor" + "servidor apagado" en <= 15s
```
