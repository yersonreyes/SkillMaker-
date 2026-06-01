# Smoke Tests — scaffold-frontend-base

Manual verification recipes for the SCAFFOLD-FE-RF-* and SCAFFOLD-FE-RNF-* requirements.
Run these after `make install && make db-up && make db-migrate && make db-seed && make dev`
(frontend served at `:4200` for dev smoke tests, or `make docker-up` for Docker smoke tests).

## Prerequisitos

- Backend corriendo en `:3000` con `make backend-dev` o `make dev`
- Frontend corriendo en `:4200` con `make frontend-dev` o `make dev`
- `GOOGLE_CLIENT_ID` y `GOOGLE_HOSTED_DOMAIN` configurados en `frontend/src/environments/environment.ts`
- Cuenta de Google del dominio corporativo disponible
- DevTools del browser abierto (Network + Application tab)

---

## SCAFFOLD-FE-RF-01 — Zoneless bootstrap

1. Abrir `http://localhost:4200`
2. En DevTools Console verificar que no hay errores relacionados a Zone.js
3. Verificar en Sources que `provideZonelessChangeDetection()` esta en `src/app/app.config.ts`
4. **Expected**: la app monta sin Zone.js; `zone.js` NO aparece en el bundle de Network

---

## SCAFFOLD-FE-RF-02 — Root redirect

1. Sin tokens en `localStorage`, ir a `http://localhost:4200/`
2. **Expected**: redirige automaticamente a `/auth/login`
3. URL en la barra de direcciones muestra `/auth/login`

---

## SCAFFOLD-FE-RF-03 — Login page renders + GIS loaded

1. Navegar a `/auth/login`
2. Verificar que el boton "Continuar con Google" es visible y esta habilitado
3. En Console: ejecutar `window.google?.accounts?.id` — debe retornar un objeto, no `undefined`
4. **Expected**: pagina visible, GIS API disponible y correctamente inicializada

---

## SCAFFOLD-FE-RF-04 + RF-05 — Login flow exitoso

1. En `/auth/login`, click en "Continuar con Google"
2. Completar el popup de Google con una cuenta del dominio configurado (`googleHostedDomain`)
3. En Network tab: verificar `POST /api/auth/google` con status 200 y body `{ idToken: "..." }`
4. Response debe contener `access_token`, `refresh_token`, `expires_at`, `user`
5. En Application > Local Storage verificar:
   - `auth_token` — JWT string no nulo
   - `auth_refresh_token` — string no nulo
   - `auth_user` — JSON con `id`, `email`, `nombre`
6. URL cambia a `/platform/catalog`
7. Header muestra el `nombre` del usuario
8. Sidebar visible con items de aprendizaje
9. **Expected**: tokens persistidos, navegacion exitosa, layout cargado

---

## SCAFFOLD-FE-RF-04 (failure) — Login con dominio incorrecto

1. Click en "Continuar con Google", seleccionar cuenta que NO pertenece al dominio corporativo
2. En Network: `POST /api/auth/google` con status 401 (`UNAUTHORIZED_DOMAIN`)
3. **Expected**: toast de error visible con mensaje de dominio incorrecto
4. `localStorage.getItem('auth_token')` es `null`
5. URL sigue en `/auth/login`

---

## SCAFFOLD-FE-RF-10 — Restore session tras recarga

1. Login exitoso, navegar a `/platform/profile`
2. Presionar F5 (recargar pagina)
3. **Expected**: la pagina carga en `/platform/profile` sin redirigir a login
4. Header muestra el `nombre` del usuario (restaurado desde localStorage)
5. `AuthService.user()` Signal es no-nulo (verificable en Angular DevTools)

---

## SCAFFOLD-FE-RF-07 + RF-08 — Refresh rotation

1. Login exitoso
2. En DevTools > Application > Local Storage: eliminar `auth_token` manualmente (simula token expirado)
3. Navegar a cualquier ruta de plataforma (ej: `/platform/catalog`)
4. En Network tab: verificar `POST /api/auth/refresh` con status 200
5. En Application > Local Storage: `auth_token` y `auth_refresh_token` tienen nuevos valores
6. **Expected**: refresh transparente, el usuario no ve errores ni es redirigido a login

---

## SCAFFOLD-FE-RF-08 (race) — Refresh singleton concurrente

1. Login exitoso, JWT con poco tiempo restante
2. Eliminar `auth_token` de localStorage manualmente
3. En DevTools Console, lanzar 3 requests simultaneos:

```js
await Promise.all([
  fetch('/api/health', { headers: { Authorization: 'Bearer ' + localStorage.getItem('auth_token') } }),
  fetch('/api/health', { headers: { Authorization: 'Bearer ' + localStorage.getItem('auth_token') } }),
  fetch('/api/health', { headers: { Authorization: 'Bearer ' + localStorage.getItem('auth_token') } }),
]);
```

4. **Expected**: en Network tab, UN solo `POST /api/auth/refresh` (no tres)
5. Los 3 fetches originales reintentan con el mismo access token nuevo

---

## SCAFFOLD-FE-RF-07 (reuse detection) — Backend revoca sesion

1. Login exitoso (obtenemos R1 = refresh token inicial)
2. Llamar `/api/auth/refresh` con R1 → recibir R2 (localStorage ya tiene R2)
3. Forzar reutilizacion de R1: en Console ejecutar una request de refresh con el token viejo
4. **Expected**: backend responde 401 `INVALID_REFRESH`
5. `AuthService.sessionExpired` Signal pasa a `true`
6. Frontend redirige a `/auth/login` (o muestra modal de sesion expirada)
7. `localStorage.getItem('auth_token')` es `null`
8. `localStorage.getItem('auth_refresh_token')` es `null`

---

## SCAFFOLD-FE-RF-11 — Logout

1. Login exitoso, navegar a `/platform/catalog`
2. Click en el avatar del header → popover → "Cerrar sesion"
3. En Network: `POST /api/auth/logout` con `{ refreshToken: "..." }`, response 204
4. En Application > Local Storage: `auth_token`, `auth_refresh_token`, `auth_user` eliminados
5. URL cambia a `/auth/login`
6. En Console: `google.accounts.id.disableAutoSelect` fue llamado (sin error)
7. **Expected**: sesion completamente cerrada, no hay auto-login al recargar

---

## SCAFFOLD-FE-RF-12 — Auth guard bloquea acceso sin sesion

1. Sin tokens en localStorage, navegar directamente a `http://localhost:4200/platform/catalog`
2. **Expected**: redirige a `/auth/login`
3. El contenido de catalog NO es visible

---

## SCAFFOLD-FE-RF-13 — Guest guard bloquea re-login autenticado

1. Login exitoso (token valido en localStorage)
2. Navegar a `http://localhost:4200/auth/login`
3. **Expected**: redirige a `/platform/catalog`
4. El formulario de login NO es visible

---

## SCAFFOLD-FE-RF-15 + RF-16 — Sidebar items por rol

1. Login con usuario con rol `alumno` (rol por defecto)
2. **Expected sidebar visible**: Catalogo, Mis cursos, Certificados, Insignias, Perfil
3. **Expected NO visible**: Mi contenido, Crear curso, Avance del equipo, Aprobaciones, Usuarios y roles
4. Logout → promover usuario a `creador` via SQL en backend (ver `SETUP.md §9`)
5. Login nuevamente
6. **Expected sidebar agrega**: seccion "Creador" con Mi contenido y Crear curso
7. Similar para `administrador` (Aprobaciones, Gestion usuarios) y `supervisor` (Avance del equipo)

---

## SCAFFOLD-FE-RF-17 — Sidebar colapsable persistido

1. Login, sidebar en estado expandido (16rem / ancho completo)
2. Click en el toggle del sidebar (icono angle-left o equivalente)
3. **Expected**: sidebar pasa a 4rem, solo iconos visibles, tooltips al hover sobre cada item
4. En Application > Local Storage: `ui.sidebar.collapsed` = `"true"`
5. Recargar pagina (F5)
6. **Expected**: sidebar arranca en estado colapsado sin flash al estado expandido

---

## SCAFFOLD-FE-RF-19 — Stubs renderizan correctamente

1. Login, navegar a `/platform/catalog`
2. **Expected**: titulo "Catalogo de cursos" visible + mensaje "Pendiente de implementacion"
3. Navegar a `/platform/my-courses`
4. **Expected**: titulo "Mis cursos" visible + mensaje "Pendiente de implementacion"
5. Navegar a cualquier ruta de `PendingViewComponent` (ej: `/platform/certificates`)
6. **Expected**: titulo dinamico de la ruta + mensaje generico visible
7. Sin errores en Console en ninguno de los pasos

---

## SCAFFOLD-FE-RF-20 — Profile muestra datos del JWT

1. Login con usuario `alice@tuempresa.com` / `nombre: "Alice Doe"`
2. Navegar a `/platform/profile`
3. **Expected**: `alice@tuempresa.com` visible en la pagina
4. **Expected**: `"Alice Doe"` visible en la pagina
5. Los roles del usuario visibles (ej: "alumno")
6. Campos en modo de solo lectura (sin inputs editables)
7. Logout + login con otra cuenta → datos se actualizan

---

## SCAFFOLD-FE-RNF-01 — make dev end-to-end

```bash
make install
make db-up
make db-migrate && make db-seed
make dev
```

**Expected output:**
```
[BACKEND]  servidor escuchando port=3000
[FRONTEND] Angular Live Development Server is listening on localhost:4200
```

Ambos procesos corriendo con prefijos coloreados (BACKEND/FRONTEND). No hay errores de compilacion.

---

## SCAFFOLD-FE-RNF-02 — TypeScript strict (build limpio)

```bash
cd /home/dev/proyectos/personal/skillmaker/frontend
npx ng build
# Expected: compilacion exitosa, exit 0
# WARNING de budget (>500kB inicial) es aceptable y pre-existente
# ERROR de TypeScript o de template = FALLA
```

---

## SCAFFOLD-FE-RNF-03 — Docker image < 100MB

```bash
cd /home/dev/proyectos/personal/skillmaker/frontend
docker build -t skillmaker-frontend:test .
docker image inspect skillmaker-frontend:test --format='{{.Size}}'
# Expected: numero < 100000000 (100 MB en bytes)
```

La imagen multi-stage (node:20-alpine build + nginx:alpine runtime) tipicamente resulta en ~40-60MB.

---

## SCAFFOLD-FE-RNF-04 — nginx proxy + SPA fallback (Docker stack)

```bash
# Levantar stack completo
make docker-up

# Verificar proxy API (nginx -> backend)
curl -s http://localhost:3700/api/health
# Expected: {"status":"ok"}

# Verificar SPA fallback (ruta Angular arbitraria -> 200 con index.html)
curl -s http://localhost:3700/platform/some/random/route -o /dev/null -w '%{http_code}\n'
# Expected: 200

# Verificar root
curl -s http://localhost:3700/ -o /dev/null -w '%{http_code}\n'
# Expected: 200

# Verificar cache headers para assets (los .js/.css tienen hash en el nombre)
curl -sI http://localhost:3700/ | grep -i cache-control
# index.html: Cache-Control: no-cache, no-store, must-revalidate
```

---

## SCAFFOLD-FE-RNF-05 — TypeScript strict (sin errores de tipos)

```bash
cd /home/dev/proyectos/personal/skillmaker/frontend
npx tsc --noEmit
# Expected: exit 0, sin errores de tipos
```

---

## SCAFFOLD-FE-RNF-07 — Test runner

```bash
make frontend-test
# Expected: exit 0
# "No test files found" o similar es aceptable (passWithNoTests: true)
```

---

## SCAFFOLD-FE-RNF-11 — Makefile targets operativos

Verificar que todos los targets frontend del Makefile raiz ejecutan sin el mensaje de stub:

```bash
# Desde la raiz del monorepo
make frontend-install   # debe ejecutar npm install, no imprimir warning
make frontend-build     # debe ejecutar ng build --configuration=production
make frontend-test      # debe ejecutar vitest run
make frontend-lint      # debe ejecutar ng lint
```

Ninguno de los comandos debe imprimir `"no scaffoldeado todavia"`.

---

## Checklist de Verificacion Completa

| Escenario | Archivo de spec | Estado |
|-----------|-----------------|--------|
| Zoneless bootstrap | RF-01 | [ ] |
| Root redirect | RF-02 | [ ] |
| Login page + GIS | RF-03 | [ ] |
| Login exitoso + tokens | RF-04 + RF-05 | [ ] |
| Login dominio incorrecto | RF-04 | [ ] |
| Restore session tras F5 | RF-10 | [ ] |
| Refresh rotation | RF-07 + RF-08 | [ ] |
| Refresh singleton (race) | RF-08 | [ ] |
| Reuse detection / revocacion | RF-07 | [ ] |
| Logout | RF-11 | [ ] |
| Auth guard bloquea sin sesion | RF-12 | [ ] |
| Guest guard bloquea autenticado | RF-13 | [ ] |
| Sidebar items por rol | RF-15 + RF-16 | [ ] |
| Sidebar collapse persistido | RF-17 | [ ] |
| Stubs renderizan | RF-19 | [ ] |
| Profile muestra JWT data | RF-20 | [ ] |
| make dev e2e | RNF-01 | [ ] |
| Build limpio (ng build) | RNF-02 | [ ] |
| Docker image < 100MB | RNF-03 | [ ] |
| nginx proxy + SPA fallback | RNF-04 | [ ] |
| TypeScript strict (tsc --noEmit) | RNF-05 | [ ] |
| Test runner | RNF-07 | [ ] |
| Makefile targets reales | RNF-11 | [ ] |
