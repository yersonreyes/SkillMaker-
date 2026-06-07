# SDD Roadmap — SkillMaker

> Plan de implementacion incremental usando el workflow SDD (Spec-Driven Development).
> Cada **change** es atomico (propose → spec → tasks → apply → verify → archive).
> Las dependencias entre changes determinan el orden, pero changes de capas distintas
> sin overlap pueden paralelizarse cuando hay mas de un dev.
>
> **Fuente de verdad de scope:** [`documentacion/`](documentacion/) (RFs, RTs, modelo de datos).
> **Estado vivo del proyecto:** ver `git log --oneline` + topic keys en engram `sdd/*/archive-report`.

---

## Tabla de contenidos

1. [Estado actual del repo (snapshot)](#1-estado-actual-del-repo-snapshot)
2. [Convencion de naming](#2-convencion-de-naming)
3. [Vista general del roadmap](#3-vista-general-del-roadmap)
4. [Detalle de cada SDD change](#4-detalle-de-cada-sdd-change)
   - [Capa 0 — Foundation crosscutting](#capa-0--foundation-crosscutting)
   - [Capa 1 — Users completo](#capa-1--users-completo)
   - [Capa 2 — Courses (4 sub-changes)](#capa-2--courses-4-sub-changes)
   - [Capa 3 — Evaluations](#capa-3--evaluations)
   - [Capa 4 — Approvals](#capa-4--approvals)
   - [Capa 5 — Certificates](#capa-5--certificates)
   - [Capa 6 — Reporting](#capa-6--reporting)
   - [Capa 7 — Hardening post-MVP](#capa-7--hardening-post-mvp)
5. [Orden recomendado para llegar al MVP demo](#5-orden-recomendado-para-llegar-al-mvp-demo)
6. [Como arrancar un SDD change](#6-como-arrancar-un-sdd-change)
7. [Deuda tecnica acumulada (corregible en cualquier momento)](#7-deuda-tecnica-acumulada-corregible-en-cualquier-momento)

---

## 1. Estado actual del repo (snapshot)

> Esta seccion es el "punto de partida". Cualquier change posterior parte de este estado.
> Actualizar cuando se archive un change importante.

**Fecha del snapshot:** 2026-06-07 (actualizado tras P3 notifications-inapp archive — Post-MVP improvement program 3/4 complete)
**Commits totales del scaffold:** 16 (+ 3 chained PRs para C1.1) (+ 2 PRs para C2.1) (+ 2 PRs para C2.2) (+ 2 PRs para C2.3) (+ 3 PRs para C3.1 incl. UI polish) (+ 3 PRs para C3.2) (+ 3 PRs para C4.1 incl. swagger fix) (+ 2 PRs para C2.4) (+ 2 PRs para C5.1 incl. bundled eval-summary slice) (+ 2 PRs para C6.1) (+ 2 PRs para course-structure-v2 + styling polish) (+ 2 PRs para C8.1 refresh-token-tracking) (+ 2 PRs para P1 catalog-filters) (+ post-merge fixes)
**LOC totales (Go + TS + SQL):** ~2700 (+ ~900 LOC en C1.1) (+ ~1100 LOC en C2.1) (+ ~900 LOC en C2.2) (+ ~900 LOC en C2.3) (+ ~1200 LOC en C3.1) (+ ~1200 LOC en C3.2) (+ ~1100 LOC en C4.1) (+ ~1300 LOC en C2.4) (+ ~1700 LOC en C5.1 incl. bundled slice) (+ ~800 LOC en C6.1) (+ ~2150 LOC en course-structure-v2) (+ ~420 LOC en C8.1 refresh-token-tracking) (+ ~340 LOC en P1 catalog-filters)
**HITO ALCANZADO**: ✅ **MVP FEATURE COMPLETE** (2026-06-05) — Todas las capas C1–C6 archivadas. El loop completo (register → create → approve → publish → consume → evaluate → certify → badge → rank → report) está funcional en main. **Post-MVP hardening INICIADO**: C8.1 refresh-token-tracking archivado 2026-06-06.

### Modulos del dominio (los 7 declarados en RT)

| Modulo | Estado | Tablas SQL existentes |
|--------|--------|----------------------|
| `auth` | ✅ Completo | `refresh_token` |
| `users` | ✅ Completo (C1.1: list, roles, supervision, soft-delete, last-admin guard) | `user`, `role`, `user_role`, `supervision` |
| `courses` | ✅ Completo (C2.1: dominio + CRUD; C2.2: secciones/videos; C2.3: material; C2.4: catalog + enrollment) | `course`, `section`, `video`, `material`, `enrollment` (schema + C2.2 + C2.3 + C2.4 columns) |
| `evaluations` | ✅ Completo (C3.1: diseño de examen; C3.2: rendición de examen + seams de enrollment/certificates) | `evaluation`, `question`, `question_option`, `attempt`, `answer` (schema + C3.1 + C3.2 columns) |
| `approvals` | ✅ Completo (C4.1: module-approvals — submit/approve/reject/history, 5 routes, 2 seams) | `approval` (C4.1: resultado, comentario, resuelto_en) |
| `certificates` | ✅ Completo (C5.1: module-certificates — PDF generation, badges, ranking, seam wiring) | `certificate`, `badge`, `user_badge` (0010: UNIQUE(user_id,course_id), no attempt_id) |
| `reporting` | ✅ Completo (C6.1: module-reporting — 4 endpoints, 3 dashboards, SQL puro, sin migration) | — (read-only, no tablas propias) |

### Frontend pages

| Page | Estado |
|------|--------|
| `pages/auth/login` | ✅ Funcional (GIS + JWT + retry) |
| `pages/auth/callback` | ✅ Stub minimo (compat futuro redirect-flow) |
| `pages/platform/catalog` | ✅ Funcional (C2.4: grid + debounced search + PrimeNG Paginator) |
| `pages/platform/my-courses` | ✅ Funcional (C2.4: table + Completado badge + Continuar nav) |
| `pages/platform/profile` | ✅ Funcional (read-only desde JWT) |
| `pages/platform/courses/:id` | ✅ Funcional (C2.4: preview/enrolled discriminator + VideoEmbed; C5.1: "Mi certificado" + "Rendir evaluación" buttons bundled) |
| `pages/platform/certificates` | ✅ Funcional (C5.1: grid de tarjetas con descarga PDF) |
| `pages/platform/badges` | ✅ Funcional (C5.1: grid de insignias + tabla ranking) |
| `pages/platform/evaluaciones/:id` | 🟡 Routes → `PendingViewComponent` |
| `pages/platform/creator/mi-contenido` | ✅ Funcional (C2.1: lazy Table paginated + estado badge + create dialog) |
| `pages/platform/creator/curso-editar/:id` | ✅ Funcional (C2.1: form titulo/descripcion, Save, disabled "Enviar a revisión"; C2.2: secciones + videos + reorder; C2.3: material adjunto con uploader + tabla; Cyanotype Workshop design; C3.1: "Definir evaluación" nav button) |
| `pages/platform/creator/evaluacion-editar/:courseId` | ✅ Funcional (C3.1: evaluation form, question list + modal, opcion_multiple dynamic options, verdadero_falso radio selector, client validation) |
| `pages/platform/evaluacion-tomar/:id` | ✅ Funcional (C3.2: student exam-taking UI — start → answer (save-on-change) → submit → result with puntaje/aprobado) |
| `pages/platform/admin/user-management` | ✅ Funcional (C1.1: lazy Table, role/active filters, edit dialog) |
| `pages/platform/admin/supervision` | ✅ Funcional (C1.1: list, assign supervisor-employee, remove) |
| `pages/platform/admin/approvals` | ✅ Funcional (C4.1: pending list, approve inline, reject dialog with required comment, roleGuard) |
| `pages/platform/admin/global-reports` | ✅ Funcional (C6.1: metric cards + 2 charts, roleGuard administrador) |
| `pages/platform/admin/course-reports` | ✅ Funcional (C6.1: per-course table with approvalRate, roleGuard administrador) |
| `pages/platform/supervisor/team-progress` | ✅ Funcional (C6.1: team members table with lastAttemptDate, roleGuard supervisor) |

### Infra y tooling

- ✅ Makefile polyglot con targets `dev`, `build`, `test`, `lint`, `db-*`, `swagger`, `docker-*`
- ✅ Docker dev (postgres + minio + minio-init) en `backend/docker-compose.dev.yml`
- ✅ Docker prod (5 servicios: postgres + minio + migrate + backend + frontend nginx) en raiz
- ✅ Backend: Air (hot reload), golangci-lint, swaggo/swag, godotenv carga `.env`
- ✅ Frontend: ng serve, Tailwind 3 + tailwindcss-primeui, Vitest, ESLint
- ✅ Migrations: `golang-migrate` con `0001_init.up.sql` aplicado
- ✅ Seed: 4 roles fijos creados (`alumno`, `creador`, `supervisor`, `administrador`)

### Bugs descubiertos durante smoke test y resueltos en commits

- `a499e5a` config no cargaba `.env` (fix con godotenv)
- `6a6ae96` `hd` validation opcional cuando vacio (dev con Gmail personal)
- `306c5d0` IP/UserAgent del refresh_token nullable + GIS init tardio
- `a419e0c` Tailwind 4 → 3 (Angular 21 no aplica postcss.config para v4)

---

## 2. Convencion de naming

- Prefijo `module-` para changes que implementan un modulo del dominio: `module-users`, `module-courses-*`, `module-evaluations-*`, etc.
- Prefijo descriptivo para cross-cutting: `testing-foundation`, `api-types-codegen`, `ci-pipelines`.
- Cuando un modulo es muy grande (>~1500 LOC estimado), partirlo en sub-changes sufijados: `module-courses-domain`, `module-courses-content`, etc.
- El nombre se persiste en engram como `sdd/{change-name}/{proposal,spec,tasks,apply-progress,verify-report,archive-report}`.

---

## 3. Vista general del roadmap

| # | Change | Capa | LOC est. | Depende de | Cubre RFs |
|---|--------|------|---------:|------------|-----------|
| C0.1 | [`testing-foundation`](#c01--testing-foundation) | 0 | ~300 | — | (RNFT-05) |
| C0.2 | [`api-types-codegen`](#c02--api-types-codegen) | 0 | ~150 | — | (calidad) |
| C0.3 | [`ci-pipelines`](#c03--ci-pipelines) | 0 | ~200 | C0.1 | (calidad) |
| C1.1 | [`module-users`](#c11--module-users) | 1 | ~800 | — | RF-04, RF-04b |
| C2.1 | [`module-courses-domain`](#c21--module-courses-domain) | 2 | ~700 | C1.1 | RF-05, RF-09 |
| C2.2 | [`module-courses-content`](#c22--module-courses-content) | 2 | ~500 | C2.1 | RF-06 |
| C2.3 | [`module-courses-material`](#c23--module-courses-material) | 2 | ~400 | C2.1 | RF-07, RT-21..24b |
| C2.4 | [`module-courses-catalog`](#c24--module-courses-catalog) | 2 | ~600 | C2.1, C4.1 | RF-18, RF-18b, RF-19, RF-20b |
| C3.1 | [`module-evaluations-design`](#c31--module-evaluations-design) | 3 | ~700 | C2.1 | RF-08, RF-11, RF-12 |
| C3.2 | [`module-evaluations-attempts`](#c32--module-evaluations-attempts) | 3 | ~600 | C3.1 | RF-13, RF-14, RF-20 |
| C4.1 | [`module-approvals`](#c41--module-approvals) | 4 | ~600 | C2.1, C3.1 | RF-10, RF-15, RF-16, RF-17, RF-17b |
| C5.1 | [`module-certificates`](#c51--module-certificates) ✅ ARCHIVED | 5 | ~700 | C3.2 | RF-21, RF-22 |
| C6.1 | [`module-reporting`](#c61--module-reporting) ✅ ARCHIVED | 6 | ~800 | C2.x, C3.x, C5.1 | RF-23, RF-24, RF-25 |
| C8.1 | [`refresh-token-tracking`](#c81--refresh-token-tracking) ✅ ARCHIVED | 7 | ~420 | — | (hardening) |
| C8.2 | [`pagination-policy`](#c82--pagination-policy) | 7 | ~300 | — | (escala) |
| C8.3 | [`prod-deployment`](#c83--prod-deployment) | 7 | ~500 | — | (deploy) |

**Total estimado al MVP completo (C0-C6):** ~7000 LOC en 13 changes.

---

## **✅ HITO ALCANZADO: MVP FEATURE COMPLETE (2026-06-05)**

Todas las capas C1–C6 del roadmap están ARCHIVADAS. El MVP está 100% funcional en `origin/main` (commit e42c69a).

**Loop completo verificado end-to-end:**
- User registra (C1.1: auth + users)
- Creator crea curso + contenido + evaluacion (C2.1–C2.3, C3.1)
- Creator envía a review + approval (C4.1)
- Student descubre + se inscribe + consume (C2.4)
- Student rinde evaluacion + aprueba (C3.2)
- Student recibe certificado + badge (C5.1)
- Admin + Supervisor ven reportes (C6.1)

**Proximos pasos (Capa 7 — Hardening):**
- C7.1 refresh-token-tracking (IP + user-agent en sesiones)
- C7.2 pagination-policy (cursor pagination para escala)
- C7.3 prod-deployment (Helm, secrets, observabilidad)

---

## 4. Detalle de cada SDD change

### Capa 0 — Foundation crosscutting

#### C0.1 — `testing-foundation`

**Por que:** sin esto, cada feature acumula deuda de testing. Los 4 bugs detectados en el smoke test del scaffold se hubieran prevenido con tests basicos. Activar Strict TDD desde aca evita que se repita.

**Scope IN:**
- Test runner Go: configurar `testify` correctamente. Agregar `t.Parallel()` opcional. Helpers compartidos en `backend/internal/testutil/`.
- Test runner front: validar que Vitest corre y reporta. Helpers compartidos en `frontend/src/testing/`.
- `testcontainers-go`: setup base para tests de integracion con Postgres efimero. Build tag `integration`.
- **Primer test real**: cubrir el modulo `auth` end-to-end (rotation, reuse detection, hd validation opt-in). Sirve de template para los demas modulos.
- Actualizar engram `sdd/skillmaker/testing-capabilities` con `strict_tdd: true`.

**Scope OUT:**
- CI/CD pipelines (eso es C0.3)
- Tests de los otros modulos (vendran con cada change)
- E2E tests con Playwright/Cypress (post-MVP)

**Acceptance:**
- `make backend-test` ejecuta los unit tests del modulo auth y exit 0
- `make backend-test-integration` levanta Postgres con testcontainers y corre integration tests
- `make frontend-test` ejecuta al menos un spec del AuthService y exit 0
- Cobertura del service de auth >= 70%
- Strict TDD activado en engram

**Notas:**
- Usa el skill `go-testing` autoload cuando edites archivos `_test.go`
- Patron tabla-driven para los casos de auth (4 escenarios: token valido, hd mismatch, refresh rotation, replay attack)
- El test de replay attack es el ejemplo canonico de uso de testcontainers (necesita DB real)

---

#### C0.2 — `api-types-codegen`

**Por que:** sin tipos generados, cada DTO se duplica manual en `backend/internal/modules/*/dto/` y `frontend/src/app/core/services/*/req.dto.ts`. Drift garantizado conforme crece el dominio.

**Scope IN:**
- Instalar `openapi-typescript` como dev dep del frontend
- Target `make types` en Makefile raiz:
  - Corre `swag init` para regenerar `backend/docs/swagger.json`
  - Corre `openapi-typescript backend/docs/swagger.json -o frontend/src/app/api/types.ts`
- Agregar `@api/*` alias en `tsconfig.json` apuntando a `frontend/src/app/api/`
- Documentar el flujo en `frontend/GUIA-FRONT.md`

**Scope OUT:**
- Migrar los DTOs existentes (auth) — opcional, dejarlo como referencia historica
- Codegen del lado cliente HTTP (orval, openapi-fetch) — overkill por ahora

**Acceptance:**
- `make types` corre sin errores y produce `frontend/src/app/api/types.ts` valido
- El archivo se commitea (parte del contrato)
- El tipo `paths['/auth/google']['post']` resuelve correctamente al request/response de auth

---

#### C0.3 — `ci-pipelines`

**Por que:** automatizar el "no romper main". Sin esto, el primer dev externo va a pushear codigo sin lint/test.

**Scope IN:**
- `.github/workflows/ci.yml` con jobs:
  - `lint`: `make lint`
  - `test-backend`: `make backend-test`
  - `test-backend-integration`: `make backend-test-integration` (con Docker en el runner)
  - `test-frontend`: `make frontend-test`
  - `build`: `make build` (compila ambos workspaces)
  - `docker-build`: build de las imagenes Docker (sin push)
- Branch protection rules en `main`: requiere los checks pasar para merge

**Scope OUT:**
- Deploy automatico (eso es C7.3)
- Caches granulares (npm, go modules) — agregar despues si CI tarda > 5 min
- Coverage upload a Codecov/Coveralls

**Acceptance:**
- PR a `main` dispara todos los jobs
- Todos exit 0 con el HEAD actual
- Branch protection bloquea merge si algun check falla

---

### Capa 1 — Users completo

#### C1.1 — `module-users`

**Por que:** el stub solo permite que el modulo `auth` upserte un user. Para gestion real necesitamos endpoints, paginacion, asignacion de roles y supervision. Desbloquea promote/demote desde la UI (hoy se hace con SQL manual).

**Scope IN:**

Backend:
- Endpoints (todos protegidos con `RequireRole("administrador")`):
  - `GET /api/users` — listado paginado con `?page`, `?size`, `?q` (busca por nombre o email), `?role` (filtra por rol), `?active` (true/false)
  - `GET /api/users/:id` — detalle (admin) o `/api/users/me` (cualquiera autenticado)
  - `PATCH /api/users/:id/roles` — body `{ add: [], remove: [] }` para asignar/revocar
  - `PATCH /api/users/:id/active` — body `{ active: bool }` para inactivar (RT-20b)
- Endpoints de supervision:
  - `GET /api/supervisions` — listado de relaciones supervisor → empleado (admin)
  - `POST /api/supervisions` — body `{ supervisorId, empleadoId }`
  - `DELETE /api/supervisions/:id` — remover relacion
- DTOs separados: `UserListItem`, `UserDetail`, `RolesPatchRequest`, etc.
- Tests con `testify` para todos los endpoints + integration con testcontainers para queries complejas

Frontend:
- Service `UserService` con `getAll`, `getById`, `updateRoles`, `setActive`
- Service `SupervisionService` con `getAll`, `create`, `delete`
- Page `admin/user-management`:
  - Tabla paginada (PrimeNG `Table` con lazy loading)
  - Filtros por rol, busqueda por email/nombre, toggle activo/inactivo
  - Edit dialog: checkboxes para los 4 roles + button "Inactivar" con confirm
- Page `admin/supervision-setup`:
  - Lista de supervisores con sus empleados
  - Picker de empleado para asignar
  - Boton remover

**Scope OUT:**
- Bulk operations (importar CSV de usuarios)
- Audit log de cambios de rol (post-MVP)
- Self-service profile editing (solo lectura por ahora)

**Acceptance:**
- Admin puede ver, filtrar, paginar lista de usuarios
- Admin puede agregar/quitar cualquier rol a cualquier user (con confirm para `administrador`)
- Admin puede asignar empleados a un supervisor
- Admin puede inactivar a un user (soft delete via `activo=false`)
- Coverage del service >= 70%

**Notas:**
- Reutilizar `LoadRoleNames` del stub actual de users (ya existe)
- La tabla `user_role` tiene `ON DELETE CASCADE` desde user, asi que inactivar NO cascadea (solo flag)
- Validacion: no permitir que el ultimo admin se auto-revoque su rol (regla de negocio en el service)

---

### Capa 2 — Courses (4 sub-changes)

#### C2.1 — `module-courses-domain`

**Por que:** crear el esqueleto del modulo principal del producto. Despues de esto un creador puede crear cursos en estado borrador (sin contenido todavia, eso viene en C2.2-C2.3).

**Scope IN:**

Backend:
- Migration `0002_add_courses.up.sql`:
  - Tabla `course` (id, creador_id, titulo, descripcion, estado, publicado_en, created_at, updated_at)
  - Tabla `section` (id, course_id, titulo, orden) — solo el schema, sin endpoints todavia
  - Tabla `video` (id, section_id, titulo, url, proveedor, duracion_seg, orden) — solo schema
  - Tabla `material` (id, course_id, nombre, storage_key, mime_type, tamano_bytes) — solo schema
  - Tabla `enrollment` (id, user_id, course_id, inscrito_en, completado) — solo schema
- Modulo `internal/modules/courses/`:
  - Domain models de las 5 tablas (TableName + GORM tags)
  - Repository con `Create`, `GetByID`, `UpdateByID`, `ListByCreator`, `UpdateEstado`
  - Service:
    - `Create(ctx, creatorID, CreateRequest)` — solo permite `estado='borrador'`
    - `GetByID(ctx, id)` — devuelve detalle (sin secciones/videos en C2.1)
    - `UpdateByID(ctx, id, creatorID, UpdateRequest)` — solo si `estado='borrador'` o `'rechazado'` Y `creatorID` matchea
    - `ListByCreator(ctx, creatorID, paginacion)`
- Handler: `POST /api/courses`, `GET /api/courses/:id`, `PATCH /api/courses/:id`, `GET /api/courses?creator=me`
- Errores: `ErrNotOwner`, `ErrInvalidTransition`, `ErrNotFound`
- Tests unitarios + integration

Frontend:
- Service `CourseService` con `create`, `getById`, `update`, `listByMe`
- Page `creator/my-content`:
  - Tabla con los cursos del creador actual + estado (badge color por estado)
  - Boton "Nuevo curso" → dialog con form
  - Click en fila → navega a `creator/course-edit/:id`
- Page `creator/course-edit/:id`:
  - Form: titulo, descripcion (textarea)
  - Save button → llama `update`
  - "Enviar a revision" disabled por ahora (lo habilita C2.2 con contenido)

**Scope OUT:**
- Secciones/videos (C2.2)
- Material (C2.3)
- Catalogo publico (C2.4)
- Inscripcion (C2.4)

**Acceptance:**
- Creador crea curso → aparece en `creator/my-content` con estado "borrador"
- Creador edita titulo/descripcion → cambios persisten
- Otro creador NO puede editar el curso ajeno (403)
- Admin NO puede editar (regla: ownership solo del creador hasta que admin tenga override en C4.1)

---

#### C2.2 — `module-courses-content`

**Por que:** un curso sin videos no sirve. Este change agrega secciones y videos embebidos (YouTube/Vimeo, RT-24).

**Scope IN:**

Backend:
- Endpoints anidados:
  - `POST /api/courses/:courseId/sections` — crea seccion
  - `PATCH /api/sections/:id` — edita seccion (titulo, orden)
  - `DELETE /api/sections/:id` — borra seccion (CASCADE a videos)
  - `POST /api/sections/:sectionId/videos` — agrega video (validates `proveedor` IN ('youtube','vimeo'))
  - `PATCH /api/videos/:id` — edita video (titulo, url, orden, duracion)
  - `DELETE /api/videos/:id`
- Reordenar: `PATCH /api/courses/:id/sections/reorder` body `{ ids: [...] }` que setea `orden` segun el array
- Validar URL: el backend valida que `url` sea de youtube.com / youtu.be / vimeo.com (regex simple)
- Service expone `HasContent(courseID) bool` — devuelve true si tiene al menos 1 video. Lo usa C2.4 para validar "enviar a revision".

Frontend:
- Service `SectionService` y `VideoService`
- Component `VideoEmbed` en `shared/components/`:
  - Inputs: `url` y `proveedor`
  - Computa la URL canonica de embed via funciones puras `toYoutubeEmbed`, `toVimeoEmbed`
  - Renderiza `<iframe>` con sanitization via `DomSanitizer`
- Pagina `creator/course-edit/:id` ampliada:
  - Lista de secciones (drag-and-drop opcional via PrimeNG `DragDrop`)
  - Cada seccion expandible con lista de videos
  - Botones "+ Seccion", "+ Video" por seccion
  - Modal de video: titulo, URL, proveedor (select), duracion (opcional)

**Scope OUT:**
- Reproductor con tracking de progreso (eso es parte de C2.4)
- Carga de video propio (RT-24 dice solo enlaces externos)

**Acceptance:**
- Creador agrega secciones a su curso
- Creador agrega videos a una seccion con URL de YouTube valida
- Reordenar secciones persiste el orden
- Borrar seccion cascada los videos
- `VideoEmbed` renderiza un iframe seguro para ambos proveedores

---

#### C2.3 — `module-courses-material`

**Por que:** RF-07 exige material adjunto (PDFs, docs). Implementa el flujo de URL prefirmada (RT-23) end-to-end.

**Scope IN:**

Backend:
- Endpoints:
  - `POST /api/courses/:courseId/materials/presign` — body `{ nombre, contentType, tamanoBytes }` → response `{ uploadUrl, key, expiresAt }`. Valida `tamanoBytes <= MAX_UPLOAD_BYTES` (50MB) y `contentType` esta en whitelist (PDF, Word, ZIP, imagen)
  - `POST /api/courses/:courseId/materials` — confirma upload: body `{ key, nombre, contentType, tamanoBytes }` → persiste fila en `material`
  - `GET /api/courses/:courseId/materials/:id/download` — devuelve URL prefirmada GET con TTL 15min
  - `DELETE /api/materials/:id` — borra fila + elimina objeto en MinIO (best-effort)
- El service de courses recibe `storage.Client` inyectado (ya esta en el composition root)

Frontend:
- Component `MaterialUploader` en `shared/components/`:
  - Drag-and-drop zone (PrimeNG `FileUpload`)
  - Validacion client-side: MIME + tamano antes de PUT
  - Flujo: presign → PUT directo a MinIO → POST confirmacion
  - Progress bar durante el PUT
  - Toast de exito/error
- Pagina `creator/course-edit/:id` agrega seccion "Material adjunto" con uploader + tabla de materiales

**Scope OUT:**
- Versionado de material (sobreescribir el archivo)
- Conversion automatica de formatos
- Preview inline (solo download por ahora)

**Acceptance:**
- Creador sube un PDF de 5MB → aparece en la tabla con su nombre y tamano
- Click en download → archivo descarga
- Subir archivo de 60MB → bloqueado con mensaje claro
- Subir archivo `.exe` → bloqueado por MIME

**Notas:**
- La presigned URL del MinIO debe tener CORS configurado para aceptar PUT desde `http://localhost:4200` en dev. Verificar en `docker-compose.dev.yml` que MinIO tenga `MINIO_BROWSER_REDIRECT_URL` u alguna config CORS.

---

#### C2.4 — `module-courses-catalog` ✅ ARCHIVED (2026-06-05)

**Estado:** COMPLETE — 2 chained PRs (backend + frontend) merged to main. Manual smoke test PASSED. Closes the create→approve→**consume** loop.

**Delivered:**

Backend (PR-A: ee1a5aa):
- Migration `0009_add_enrollment_completado` (completado boolean column on enrollment table)
- 6 repo methods (ListApproved JOIN creator-name, GetApprovedDetail, CreateEnrollment idempotent, IsEnrolled, MarkCompleted, ListEnrollmentsByUser scoped)
- 5 service methods (ListCatalog, GetCatalogDetail branches on enrollment, Enroll with aprobado-guard, ListMyCourses, MarkEnrollmentCompleted fulfills seam)
- 4 DTOs: CoursePreviewResponse (no tree — structural no-leak), CourseDetailAlumnoResponse (with tree), CatalogCourseCard, MyCourseItem
- 4 routes on protected group (JWT-only): GET /catalog, GET /catalog/:id, POST /catalog/:id/enroll, GET /users/me/courses
- EnrollmentCompleter seam wired in main.go (non-vacuous end-to-end: passing attempt → completado=true)
- Tests: 30 tasks, all green; 0 CRITICAL, 6 adversarial probes RED-confirmed

Frontend (PR-B: f24c1bf + fixes):
- CourseCatalogService (getCatalog/getDetail/enroll/getMyCourses)
- CourseCardComponent (reusable grid card, Cyanotype design)
- catalog page (grid + 300ms debounced search + PrimeNG Paginator)
- course-detail page (preview/enrolled discriminator + VideoEmbed reuse)
- my-courses page (table + Completado badge + Continuar absolute nav)
- Tests: 16 tasks, 239 vitest all green; 0 CRITICAL, 4 probes RED-confirmed
- Discovery: Angular template type-check gap (caught by `ng build`, not `tsc --noEmit`)

**Unblocks:** C5.1 certificates (CertificateIssuer seam from C3.2 ready to wire)

**Scope IN:**

Backend:
- Endpoints:
  - `GET /api/courses` — catalogo publico (todos los autenticados). Filtra `WHERE estado = 'aprobado'`. Soporta `?page`, `?size`, `?q` (busca en titulo)
  - `GET /api/courses/:id` (para alumno autenticado) — incluye secciones + videos + material si el alumno esta inscrito; si no, solo titulo + descripcion + creador
  - `POST /api/courses/:id/enroll` — inscribe al user en el curso (idempotente: 200 si ya estaba inscrito)
  - `GET /api/users/me/courses` — lista de mis cursos inscritos con progreso
- Service: el `Enroll` crea fila en `enrollment` con `completado=false`. El `completado=true` lo setea el modulo `evaluations` cuando un attempt es aprobado (interface call cross-modulo).

Frontend:
- Service `CourseCatalogService` (`getCatalog`, `getDetail`, `enroll`, `getMyCourses`)
- Page `catalog` (reemplaza el stub):
  - Grid de `CourseCard` (component nuevo en `shared/`)
  - Cada card muestra titulo, descripcion truncada, creador, boton "Ver detalle"
  - SearchBar arriba (debounced 300ms)
  - Pagination con PrimeNG `Paginator`
- Page `course-detail/:id` (reemplaza el stub):
  - Header con titulo + creador
  - Si no inscrito: boton "Inscribirme" → POST enroll → recarga
  - Si inscrito: muestra secciones expandibles con videos + lista de material
- Page `my-courses` (reemplaza el stub):
  - Tabla de cursos inscritos
  - Badge "Completado" si `enrollment.completado = true`
  - Boton "Continuar" → navega a `course-detail/:id`

**Scope OUT:**
- Tracking de progreso por video (mark video as watched)
- Filtros avanzados (por creador, fecha, duracion)
- Sistema de tags/categorias (no esta en el modelo)

**Acceptance:**
- Catalogo muestra solo cursos `estado='aprobado'`
- Alumno se inscribe → aparece en `my-courses`
- Alumno entra a course-detail → ve secciones + videos embebidos + descarga materiales
- Alumno no inscrito ve solo el preview (titulo + descripcion)

---

### Capa 3 — Evaluations

#### C3.1 — `module-evaluations-design` ✅ ARCHIVED (2026-06-04)

**Estado:** COMPLETE — 2 chained PRs + UI polish merged to dev. Manual smoke test PASSED. Cross-module seam pattern established.

**Delivered:**

Backend (PR-A: 88a4a8f):
- Migration `0006_add_evaluations.up.sql` (5 tables: evaluation, question, question_option, attempt, answer; CHECK constraints, FK cascade, indexes).
- Modulo `evaluations/` (domain, repo, service, handler, dto, facade) con 13 repo methods, 9 service methods, 75% coverage.
- **Cross-module seam**: `courses.Service.GetCourseOwnership(ctx, courseID) (creadorID, estado string, error)` interface + implementation. Narrow seam (CoursesChecker) keeps evaluations isolated.
- Endpoints (all RequireRole creador, ownership gated via seam):
  - `POST /api/courses/:courseId/evaluation` (create 1-1 per course)
  - `GET /api/courses/:id/evaluation` (nested tree: eval→questions→options)
  - `PATCH /api/evaluations/:id` (update nota_minima, intentos_max)
  - CRUD question: `POST /api/evaluations/:id/questions`, `PATCH /DELETE /api/questions/:id`
  - CRUD option (opcion_multiple): `POST /api/questions/:id/options`, `PATCH /DELETE /api/options/:id`
  - V/F auto-creates 2 options; mutual-exclusion enforced on PATCH (sibling auto-cleared)
- `validateQuestionComplete` helper implemented but ungated (wired in C4.1).
- Tests: 49 integration tests + 50+ unit tests; all LOAD-BEARING scenarios covered. Defects caught: (1) migration step-drift in courses tests, (2) V/F mutual-exclusion not implemented — FIXED.

Frontend (PR-B: 2e91a48) + UI polish (859bc81):
- `EvaluationService` (getByCourse 404→null, create, update).
- `QuestionService` (CRUD question + options folded in).
- Page `evaluacion-editar/:courseId`: empty state on 404, full editor on found. Form (notaMinima, intentosMax), question list + modal, opcion_multiple dynamic options (correcta checkboxes), verdadero_falso fixed 2-option radio (exactly 1 correct). Client ≥1-correct validation. Nav "Definir evaluación" button in curso-editar with absolute `/platform/creator/evaluacion-editar/:courseId` path.
- Shared form/button/empty primitives promoted to global styles (reusable, budget relief).
- Tests: 6 + 10 + 14 tests (EvaluationService, QuestionService, component); 2 LOAD-BEARING probes confirm non-vacuous (navigator path, client validation).

**Aceptacion:** ✅
- Creador define evaluacion con nota minima 70, intentos_max 2
- Creador agrega 5 preguntas: 3 opcion_multiple + 2 verdadero_falso
- Cada pregunta tiene ≥1 correcta; GET devuelve nested tree completo
- Non-owner edit → 403; curso no editable → 409
- Mutual-exclusion: marcar una opcion V/F correct → la otra auto-clears
- Smoke test manual: migration, ownership, editor, reload persistence — PASSED

**Scope realizado:**
- ✅ Schema 5 tablas + indices + checks + FK cascade
- ✅ Cross-module seam (courses.GetCourseOwnership)
- ✅ All CRUD endpoints, nested tree, validation
- ✅ Full frontend editor con client-side validation
- ✅ Service coverage 75% ≥ 70% threshold

**Scope OUT (C3.2):**
- Attempts, answers rendering, taking, scoring (C3.2)
- Open-ended questions, question bank, multi-evaluation per course

---

#### C3.2 — `module-evaluations-attempts` ✅ ARCHIVED (2026-06-05)

**Estado:** COMPLETE — 2 chained PRs (backend + frontend) + focused UX fix merged to dev. Manual smoke test PASSED. Forward seams (enrollment, certificates) are nil-safe, ready for wiring.

**Delivered:**

Backend (PR-A: 837b1de):
- Migration `0007_answer_unique.up.sql` adds UNIQUE(attempt_id, question_id) for upsert.
- Modulo `evaluations/` extended: 7 new repo methods, 4 new service methods (StartAttempt/GetAttempt/SaveAnswer/SubmitAttempt), Go-composed scoring (earned/total*100 rounded to int).
- **Forward seams**: `EnrollmentCompleter.MarkEnrollmentCompleted(ctx, userID, courseID)` + `CertificateIssuer.IssueOnPass(ctx, userID, courseID)` — nil in C3.2, wired by C2.4/C5.1. Functional-options pattern keeps existing 2-arg `New()` compiling.
- Endpoints (all protected JWT-only):
  - `POST /api/evaluations/:id/attempts` — start attempt (resume-on-start if open exists, no error); enforce intentos_max (0=unlimited)
  - `GET /api/attempts/:id` — state + questions WITHOUT correcta while in-progress; puntaje+aprobado post-submit (still no correcta revealed)
  - `POST /api/attempts/:id/answers` — upsert via UNIQUE; snapshot correcta at save time
  - `POST /api/attempts/:id/submit` — compute puntaje, set aprobado, invoke seams on pass (non-fatal errors)
- Tests: 183 service/handler tests pass (exit 0), integration migration round-trip clean, mock seams prove invocation, 76.3% service coverage.

Frontend (PR-B: bf0baa7):
- `AttemptService` (startAttempt, getAttempt, saveAnswer, submitAttempt) + LOCAL AttemptStateOption type (NO correcta field — compile-time guarantee).
- Page `evaluacion-tomar/:id` (replaces stub): standalone + zoneless + signals. Flow: start card → all questions on scrollable form (radio options) → save-on-change → submit confirm dialog → result screen (puntaje + aprobado/desaprobado). Cyanotype Workshop primitives. Pre-populates savedAnswers from state on resume-on-start.
- Tests: 181 total (158 baseline + 23 new PR-B). LOAD-BEARING: no-correcta verified across 4 layers (type, service, component, template); 3 adversarial probes caught injected leaks.

Focused Fix (af0a32e):
- **Problem**: StartAttempt blocked with ErrAttemptOpen when open attempt existed → dead-end UX (can't start, can't resume).
- **Solution**: StartAttempt now RESUMES the open attempt (returns it, no new attempt, no intentos_max consumption). Frontend pre-populates answers. Discovered + applied during manual smoke test.
- **Scope**: seams nil, answer.correcta snapshot, intentos_max (0=unlimited), ownership (404 on mismatch), scoring (integer %, total==0 guard), upsert-answers before-submit, ErrAttemptAlreadySubmitted after-submit, resume-on-start.

**Acceptance:** ✅
- Alumno inicia attempt → ve preguntas (no correcta visible)
- Alumno responde todas (save-on-change) → submit → recibe puntaje + aprobado/desaprobado
- Si aprobado, seams invoked (via mock proven; real enrollment/certificates wired later by C2.4/C5.1)
- Si intentos_max alcanzado, POST /attempts → 409 ErrMaxAttemptsReached
- Re-submit → 409 ErrAttemptAlreadySubmitted
- Open attempt → resume on next start (UX fix discovery, no error)

**Unblocks:**
- C5.1 certificates (CertificateIssuer seam ready)
- C2.4 enrollment completion (EnrollmentCompleter seam ready)

---

### Capa 4 — Approvals

#### C4.1 — `module-approvals` ✅ ARCHIVED (2026-06-05)

**Estado:** COMPLETE — 2 chained PRs + swagger collision fix merged to dev. Manual smoke test PASSED. Unblocks C2.4 catalog and C5.1 certificates.

**Delivered:**

Backend (PR-A: cbd6299):
- Migration `0008_add_approvals.up.sql` (publicado_en column + approval table with CHECK constraint on resultado)
- Modulo `approvals/` (domain, repo, service, handler, dto, facade) con 5 service methods, 2 seam interfaces (CourseStateManager, EvaluationValidator), 78.3% coverage
- 5 endpoints: `POST /courses/:courseId/submit` (creador), `GET /approvals/pending` (admin), `POST /courses/:courseId/approve` (admin), `POST /courses/:courseId/reject` (admin), `GET /courses/:id/approvals` (admin + owner)
- Cross-module seams: courses.SetEstado (stamps publicado_en on aprobado), courses.ListByEstado, evaluations.ValidateSubmitReady
- Two-write ordering (approval row → SetEstado) mitigates partial-failure risk; documented as R1 follow-up
- 26 tasks complete; 11 unit tests PASS, integration tests PASS, 7 adversarial probes RED-confirmed, routes boot-test confirms no Gin panic

Frontend (PR-B: bb39758):
- `ApprovalService` (5 methods: submitToReview, listPending, approve, reject, history) + 12 tests
- curso-editar: submitDisabled computed fixed (drops `|| true`), submit button wired, rejection banner + comment display
- AprovacionesComponent: replaces PendingViewComponent stub, pending list + approve inline + reject dialog with mandatory comment + roleGuard
- 8 tasks complete; 209/209 vitest PASS, lint clean; 2 adversarial probes RED-confirmed

Bug Fix (commit 185f3d7):
- Resolved swagger DTO short-name collision (approvals.SubmitResponse vs evaluations.SubmitResponse) by renaming → SubmitReviewResponse + reverting --parseDependency band-aid
- Fixed pre-existing tsc errors (vi import, AttemptStateOption cast)
- Learning: check for DTO name duplicates before adding; never use --parseDependency as it breaks all consumers; always run `tsc --noEmit` in verification

**Aceptacion:** ✅
- Creador con curso en borrador + ≥1 video + evaluacion completa → submit OK → estado=en_revision
- Admin ve el curso en /approvals/pending → aprueba → estado=aprobado + publicado_en set + approval row persisted
- Admin rechaza con comentario → estado=rechazado + approval row persisted
- Creador ve estado/comentario en curso-editar + historial de revisiones
- Historial gated: admin acceso, owner acceso, non-owner 403
- Submit y approve/reject buttons disabled appropriately per estado + role
- Security: ownership check before content checks, comentario required pre-write on reject, adminId from JWT not body

**Scope realizado:**
- ✅ Migration 0008 (publicado_en + approval table)
- ✅ All 5 API endpoints, approval seam pattern, history (RF-17b)
- ✅ Frontend ApprovalService + curso-editar + AprovacionesComponent + roleGuard
- ✅ Full test coverage (78.3% backend, 209/209 frontend vitest)
- ✅ Acyclic dependency graph (no import cycles)
- ✅ Manual smoke test PASSED

**Scope OUT (follow-ups):**
- Email/push notifications (RF-17 passive-only for MVP)
- Cross-repo transactional seam (post-C4.1)
- Multi-admin consensus (RF-15 single admin decides)

**Unblocks:**
- C2.4 module-courses-catalog (can filter estado='aprobado')
- C5.1 module-certificates (CertificateIssuer seam ready from C3.2)

---

### Capa 5 — Certificates

#### C5.1 — `module-certificates` ✅ ARCHIVED (2026-06-05)

**Estado:** COMPLETE — 2 chained PRs (backend + frontend stacked) + bundled student-eval-summary slice merged to dev/main. Manual smoke test PASSED end-to-end. Closes gamification loop and C3.2 CertificateIssuer seam.

**Delivered:**

Backend (PR-A: 7ade0d6):
- Migration `0010_add_certificates.up.sql` (3 tables: certificate, badge, user_badge; UNIQUE(user_id,course_id) — **no attempt_id** per frozen seam; 3-badge seed: "Primer curso completado" (≥1), "5 cursos completados" (≥5), "10 cursos completados" (≥10))
- Modulo `certificates/` (domain, repo 8 methods, service 6 methods, handler 5 routes, dto 5 types, facade)
- Cross-module seams: `UserNameReader` (users facade), `CourseTituloReader.GetCourseTitulo` (new on courses.Service)
- Endpoints: GET /api/certificates/me, /:id, /:id/download, GET /api/badges/me, /badges/ranking — all JWT-protected
- Service: `IssueOnPass(ctx, userID, courseID)` idempotent seam (FROZEN signature, no attemptID), non-fatal to attempt submission
- PDF via go-pdf/fpdf (typographic A4 landscape, no logo for MVP)
- Tests: 72.5% coverage (service+pdf combined), 5 adversarial probes RED-confirmed, seam e2e integration test with real Postgres
- Storage extension: storage.Client.PutObject new method (server-side upload) rippled to all stubs

Frontend (PR-B: c52084b):
- CertificateService + BadgeService (HttpPromiseBuilderService pattern)
- CertificatesComponent (grid cards + window.open PDF download)
- BadgesComponent (earned grid + ranking table top 10)
- course-detail enhancement: "Mi certificado" button (C5.1) + "Rendir evaluación" button (bundled student-eval-summary slice)
- Tests: 23 new vitest (262/262 total); 3 adversarial probes RED-confirmed
- Styles: --page-* tokens only (Cyanotype Atelier pattern)

Bundled Slice (student-eval-summary):
- Backend: GET /api/courses/:id/evaluation/summary (aprobado-only, no-leak security)
- Frontend: evaluation.service.ts + course-detail "Rendir evaluación" button
- Closes UX gap: students can now discover and take exams (was missing from C3.2)

**Aceptacion:** ✅
- Alumno aprueba evaluacion → `certificate` aparece con codigo unico
- Descarga PDF → abre correctamente (nombre usuario, titulo curso, fecha, codigo)
- Re-submit no duplica (UNIQUE(user_id, course_id) en DB)
- Badge "Primer curso completado" automático
- Ranking top 10 por cantidad de certificados
- "Rendir evaluación" button visible en curso inscrito
- "Mi certificado" button visible en curso completado
- Storage/PDF error non-fatal → attempt submit still succeeds

**Scope realizado:**
- ✅ Migration 0010 (3 tables, 3-badge seed, step-drift +1 applied to 5 tests)
- ✅ storage.Client.PutObject extension + all stubs (noopStorage, mockStorageClient, new test doubles)
- ✅ courses.Service.GetCourseTitulo seam + mock ripple (3 mocks updated)
- ✅ All 5 API endpoints, PDF generation, badge thresholds
- ✅ Frontend 2 pages + 2 services + course-detail enhancements
- ✅ Full test coverage (72.5% backend, 262/262 vitest frontend)
- ✅ Bundled student-eval-summary slice (closes C3.2 UX gap)
- ✅ Manual smoke test PASSED (full happy path: enroll → take exam → certificate issued → PDF download → badge awarded → ranking updated)

**Scope OUT (post-MVP):**
- Public verify-by-code endpoint (`/api/certificates/verify/:codigo`)
- Badge icons + asset infrastructure (iconoKey intentionally dropped from MVP — deferred)
- PDF logo branding / custom templates per course
- Social sharing, NFTs

**Unblocks:**
- C6.1 module-reporting (read-only aggregate queries across all modules)

---

### Capa 6 — Reporting

#### C6.1 — `module-reporting` ✅ ARCHIVED (2026-06-05)

**Estado:** COMPLETE — 2 chained PRs (backend + frontend) merged to main. Manual smoke test PASSED. **FINAL MVP CHANGE — MVP NOW COMPLETE.**

**Delivered:**

Backend (PR-A: 119a650):
- Modulo `internal/modules/reporting/` — read-only, SQL puro, cero migration, 8 repo methods, 4 endpoints, 38/38 tasks
- 4 endpoints (all protegidos):
  - `GET /api/reports/global` (admin): activeUsers, coursesByEstado (4 estados), totalAttempts, certificatesIssued, topCreators, usersPerMonth, approvedCoursesPerMonth
  - `GET /api/reports/courses` (admin): per-course enrollments, attempts, approvalRate (0.0–1.0)
  - `GET /api/reports/users/:id/progress` (admin-or-self in-handler): enrolledCount, completedCount, attemptsCount, passedAttemptsCount, certificatesCount
  - `GET /api/reports/team` (supervisor, NEW group): scoped por supervisor_id (isolation invariant), includes lastAttemptDate via correlated subquery
- Cross-module SQL coupling: repository holds `*gorm.DB`, queries document table/column deps, NO Go imports de otros modulos (SQL layer only)
- Tests: 38 tasks complete, integration tests PASS (25 packages), handler authz matrix exhaustive, adversarial probes RED-confirmed (team-scoping isolation invariant proven)
- Coverage: handler 73.7% / repo 87.5% / svc 80.4%
- Verification gates: `make backend-test` PASS, `make backend-test-integration` PASS, `go list -deps` CLEAN, `ls migrations | wc -l == 20` (no new migration)

Frontend (PR-B: e42c69a):
- chart.js dependency added (`npm install chart.js`)
- 3 lazy-loaded pages:
  - `admin/global-reports`: metric cards + 2 PrimeNG charts (line usersPerMonth, bar approvedCoursesPerMonth) + top-creators list. **SECURITY FIX**: added roleGuard [administrador]
  - `admin/course-reports` (NEW): p-table titulo/estado/enrollments/attempts/approvalRate (%). RoleGuard [administrador]
  - `supervisor/team-progress`: p-table empleadoNombre/enrolledCount/completedCount/lastAttemptDate (null → em-dash). RoleGuard [supervisor]
- ReportingService (4 methods via HttpPromiseBuilderService)
- Tests: 289 tests PASS (35 files), `ng build` PASS (p-chart template type validation gate), tsc PASS, ng lint PASS
- Navigation: platform-layout adminItems + supervisorItems updated with absolute paths

**Learnings captured:**
1. Cross-module read exception: SQL coupling only; integration tests are sole safety net
2. Correlated subquery prevents join fan-out (critical for accurate COUNT DISTINCT)
3. Nullable column edge case: `publicado_en IS NOT NULL` on date_trunc aggregates
4. First zero-migration change since C1.x → no step-drift
5. completedCount semantic (certificate count per design); document non-obvious choices
6. lastAttemptDate field omission caught by spec (not marked deferred) → always check spec vs design
7. ng build is MANDATORY frontend verify gate (tsc + vitest miss p-chart template errors)
8. supervisorGrp establishes first supervisor-gated route pattern

**Aceptacion:** ✅
- Admin ve metricas globales actualizadas (7 campos: activeUsers, coursesByEstado, totalAttempts, certificatesIssued, topCreators, 2 charts)
- Admin filtra reporte por curso y ve tasa de aprobacion (0.0–1.0 rango)
- Supervisor ve solo su equipo (isolation invariant probado con adversarial probe)
- `/users/:id/progress`: admin acceso, mismo user acceso, supervisor-de-otro 403
- Performance: cada endpoint < 2s con 1000 users / 100 cursos / 5000 attempts
- **FINAL MVP**: register → create → approve → publish → consume → evaluate → certify → badge → rank → **report** ✅

**Scope OUT (C7+):**
- Excel/CSV export
- Email/push notifications
- Report caching (Redis)
- Perf indexes cert(course_id)/attempt(aprobado)

---

### Capa 7 — Post-MVP Refinements

#### `course-structure-v2` ✅ ARCHIVED (2026-06-06)

**Estado:** COMPLETE — 2 chained PRs (backend + frontend) + styling polish merged to main. Manual smoke test PASSED. 3 demo architecture courses seeded with SVG thumbnails.

**Delivered:**

**Scope**: Restructure courses domain to move material from course-level to per-video; add course metadata (nivel/miniatura/horas-practico/categorias); compute and surface horasVideo + cantidadClases; enrich catalog and detail UX.

Backend (PR-A: e26d717):
- Migrations 0011-0013 (video.descripcion, material→video relocation+backfill, course metadata+categorias seed)
- Material ownership chain: material→video→section→course→creador (single GetMaterialOwnership call)
- Per-video material routes: /videos/:id/materials/* (presign/confirm/list); flat download /materials/:id/download (owner-OR-enrolled)
- Course metadata: nivel (CHECK enum), miniatura_key (presigned GET), horas_practico (manual), categorias (curated 8-item seed) + GET /api/categorias
- Computed: horasVideo (ROUND(SUM duracion_s/3600, 1)), cantidadClases (COUNT videos)
- DTO breaking change: CourseDetailAlumnoResponse drops course-level materiales (now per-video in VideoResponse)
- Coverage: handler 75.2%, service 71.5%

Frontend (PR-B: c212d43):
- MaterialService re-key to videoId (presign/confirm/list paths changed)
- NEW categoriaService for GET /api/categorias
- courseService += presignThumbnail/confirmThumbnail
- curso-editar: metadata panel (nivel p-select, categorias p-multiselect, horasPractico, miniatura uploader), per-video material section, video descripcion
- course-detail: per-video materiales render, video.descripcion, metadata display, no course-level material block
- course-card: miniatura img/placeholder, nivel tag, categorias chips, cantidadClases/horasVideo/horasPractico stats
- Tests: 311 vitest PASS, ng build clean

Polish (b310324):
- course-detail spacing + Cyanotype refinements
- SVG thumbnail placeholders for demo data

**Acceptance:** ✅
- Material per-video with download authorized via ownership chain
- Course metadata visible on catalog cards and detail
- Categorias curated, selectable in course editor
- horasVideo auto-computed and accurate to 1 decimal
- cantidadClases matches video count
- Thumbnail upload + presigned GET working
- Per-video material UX in course-detail (no course-level block)
- Manual smoke test: 3 demo courses, all consumption flows PASSED

**Learnings:**
1. Material backfill heuristic (first-video-by-orden): works for dev scale; zero-NULL invariant is safety net
2. Ownership chain consolidation: single repo call prevents N+1 on per-video lists
3. Gin wildcard consistency: within method tree, use :id uniformly or panic; different methods = different trees
4. Step-drift pattern: +3 migrations = update m.Steps(-N) at 6 sites (courses ×3, approvals, certificates, evaluations)
5. Curated categories (seed-only): no admin UI for MVP; acceptable tradeoff; migration required to expand taxonomy
6. Thumbnail presign per-card: O(n) for n cards; scales OK now, flag for batching/caching post-MVP
7. Breaking contract mitigation: chained-PR ordering (backend first, frontend builds against new API); dev-only so no live-client window
8. Type accuracy (DTO mismatches): frontend miniaturaUrl should align (string non-nullable); openapi-typescript adoption recommended

**Deferred:**
- Admin UI for categorias (seed-only for now)
- Per-card miniatura presign batching/caching
- Real thumbnail images (demo uses SVG; production needs asset sourcing)

---

### Capa 8 — Hardening post-MVP

#### Post-MVP Improvement Program (4 changes)

**Iniciado:** 2026-06-06. El MVP funcional está completo en main (commit e42c69a). Los siguientes 4 cambios van a refinamientos focalizados de discovery/UX, no nuevas capas.

| # | Change | Scope | Unblocks |
|---|--------|-------|----------|
| P1 | `catalog-filters` ✅ ARCHIVED | GET /catalog gain `?nivel`, `?categoria` (repeated, OR), `?sort`; filter bar UI; backward compatible | P2 course-player-progress |
| P2 | `course-player-progress` ✅ ARCHIVED | Per-video watched flag (manual toggle); resume to first-incomplete; progress bar (X/N · %); 2-column enrolled player | P3 notifications |
| P3 | `notifications-inapp` ✅ ARCHIVED | In-app notification center (bell + panel in header, 4 endpoints, 30s poll); course-approved/rejected, certificate-issued events; non-fatal seam to approvals/certificates | P4 |
| P4 | `ux-polish` (planned) | Accessibility audit, dark mode, mobile UX, empty states consistency | **MVP fully ready for users** |

##### P1 — `catalog-filters` ✅ ARCHIVED (2026-06-06)

**Estado:** COMPLETE — 2 chained PRs (backend c7043c4 + frontend 24ab9f4) merged to dev/main. Manual smoke test PASSED. First of the post-MVP improvement series.

**Delivered:** GET /api/catalog extended with `?nivel` (single: basico|intermedio|avanzado), `?categoria` (repeated params, OR semantics via EXISTS semi-join, no fan-out), `?sort` (recientes|titulo). Handler validates → 400 on bad params. Frontend filter bar (p-select nivel + p-multiselect categorias + p-select sort) + "Limpiar filtros" button + filtered empty state. No migration (schema ready since course-structure-v2); queryParamArray (HttpParams.append) emits repeated params correctly. CRITICAL fix applied: malformed categoria UUID → 400 (not 500) via uuid.Parse validation in handler.

**Spec Compliance:** 15/15 backend scenarios PASS, 16/16 frontend scenarios PASS. Adversarial probes verified: EXISTS semi-join prevents JOIN fan-out/duplicate-count; niveau/sort allow-lists block injection; nonexistent UUID = match-nothing (not 400); multi-categoria OR semantics exact-once counting; backward compat preserved (no params = today's behavior).

**Key Learnings:**
1. EXISTS semi-join is LOAD-BEARING for paginated COUNT correctness — any JOIN refactor breaks silently (guarded by test).
2. queryParamArray must use .append (not .set) or repeated params collapse to last value (multi-categoria silently broken).
3. Categoria UUID validation must happen in handler (not at repo layer) — malformed UUIDs hit Postgres 22P02, must return 400 not 500.
4. sort allow-list + hardcoded ORDER switch blocks SQL injection; no user string reaches SQL.
5. No migration required; schema ready from course-structure-v2 → first filter change with no step-drift.

**Next:** Course-player-progress (watches + resume) to unblock notifications.

---

##### P2 — `course-player-progress` ✅ ARCHIVED (2026-06-07)

**Estado:** COMPLETE — 2 stacked PRs (backend 1fc2e26 + frontend b8c6833) merged to dev/main. Manual smoke test PASSED. Per-video progress tracking + Udemy-style 2-column player now live.

**Delivered:** Migration 0014 (video_progress table: user_id + video_id UNIQUE, completado bool, last_position_s int forward-compat). Backend: repo/service/handler for PUT /api/videos/:id/progress (protected, enrolled-gated, caller-scoped upsert); buildContentTree extended to batch-load completado flags (no N+1); DTO VideoResponse += completado. Frontend: course-detail enrolled branch reworked to 2-column grid (LEFT: active video + desc + materiales + mark-complete toggle; RIGHT: sticky curriculum with progress bar X/N · %, secciones, videos with completion checks + current highlight, click-to-play); activeVideoId signal resumes to first-incomplete video. ALL test gates PASS (handler 76.8%, service 72.6% coverage backend; vitest 358/358 frontend). Adversarial probes RED-confirmed: caller-scoped read/write isolation, enrolled-gate 404 no-leak, enrollment.completado unaffected by video progress. Step-drift +1 applied to 9 integration test sites. CRITICAL UUID validation bug found in verify + FIXED before archive + test added (pattern established for future param validation sweep).

**Key Learnings:** (1) UUID param validation at handler/service boundary → 404/400, not 500 (applies to category, video ID, others); (2) Caller-scoping at SQL/JWT layer is dominant security property; (3) enrollment.completado ↔ per-video progress decoupled intentionally, seams never touch; (4) One batch query (ListVideoProgressByUserAndCourse) prevents N+1, reusable pattern; (5) Spec-class preservation during big HTML rework — keep asserted classes, add new ones additively; (6) Plain iframe + no auto-detect baseline, manual toggle + lightweight resume shipped, auto-detect deferred.

**Unblocks:** P3 notifications-inapp (progress instrumentation as foundational signal source).

---

##### P3 — `notifications-inapp` ✅ ARCHIVED (2026-06-07)

**Estado:** COMPLETE — 2 chained PRs (backend 3a3f2e6 + frontend 77461f2) + 3 navigation smoke-test fixes (d4a1987, 53715cd, 9c7fc5a) merged to dev/main. Manual smoke test PASSED. In-app notification center live for course approvals, rejections, and certificate issuance.

**Delivered:** Migration 0015 (notification table with user_id FK, tipo CHECK, titulo, cuerpo, leida, ref_id, created_en; partial unread index + (user_id, creado_en DESC) composite). Backend: new leaf module notifications (domain, repo, service, handler, dto, facade) with 4 caller-scoped JWT-protected endpoints (GET /me paginated, GET /me/unread-count, PATCH /:id/read uuid-validated→404, PATCH /me/read-all). Consumer-declared Notifier seam (approvals + certificates declare own interface, notifications.Service satisfies structurally, NON-FATAL slog+swallow). Events: approvals.Approve→creator (curso_aprobado), Reject→creator (curso_rechazado + comentario), certificates.IssueOnPass first-issue→alumno (certificado_emitido). Frontend: platform-layout header bell with unread badge, p-popover panel listing notifications, 30s poll + load-on-open, click→navigate via ref_id+tipo (curso_aprobado/rechazado→/courses, certificado→/certificates). Notifications module coverage 91.3% service / 81.1% handler. Step-drift +1 applied to 10 integration test sites. All test gates PASS (backend-test, backend-test-integration, make types, frontend vitest 375/375, tsc 0 errors, ng build 0 errors). 9 adversarial probes RED-confirmed: caller-scoping (list/mark-read/mark-all isolation), non-fatal seam (approve/reject/issue never break), uuid.Parse→404, idempotent cert re-issue no-renotify, proper event targets (creator for course events, alumno for cert).

**Key Learnings:** (1) Notification navigation must match recipient + target's reachable route (approved→public /courses, rejected→creator editor, cert→list, not a naive /resource/:id); (2) Caller-scoping at SQL boundary (WHERE user_id=?) is dominant security property; (3) Acyclic leaf module + consumer-declares seam (no imports between notifications ↔ approvals/certificates) proven pattern from evaluations CertificateIssuer; (4) Non-fatal seam: capture source-op error FIRST, notify only on success, slog+swallow on Notifier failure; (5) Cert idempotent no-renotify: Notify inside first-issue branch, after idempotency early-return; (6) uuid.Parse at handler boundary (not 500, but 404 no-leak).

**Unblocks:** P4 ux-polish (foundation for further notifications feature — email/push/websockets).

---

#### C8.1 — `refresh-token-tracking` ✅ ARCHIVED (2026-06-06)

**Estado:** COMPLETE — 2 chained PRs (backend + frontend) merged to dev/main (commits 676ec49 + 760a0bb). Manual smoke test PASSED. Forensic session management + per-device revoke now live.

**Delivered:**

Backend (PR-A: 676ec49):
- No migration (ip/user_agent columns existed in migration 0001; latest 0013 unchanged).
- Service signatures updated: `LoginWithGoogle(ctx, idTokenStr, ip, userAgent)` + `Refresh(ctx, refreshTokenPlain, ip, userAgent)` thread ip+ua into issueRefreshToken via strPtr helper (empty→nil for inet safety).
- Repo methods: ListActiveByUser (caller-scoped WHERE user_id), RevokeByID (caller-scoped WHERE user_id + revoked_at IS NULL) with ErrNotAffected sentinel.
- Service methods: ListActiveSessions, RevokeSession (maps ErrNotAffected to ErrSessionNotFound).
- New SessionResponse DTO (id, ip?, userAgent?, createdAt, expiresAt, usedAt?) — never exposes token_hash/parent_id.
- RegisterSessionRoutes facade + 2 handlers (GetMySessions, RevokeSession) on protected group (JWT-only).
- main.go wiring: auth.RegisterSessionRoutes(protected, authSvc) after protected group built.
- Handlers capture c.ClientIP() + c.Request.UserAgent() and thread to service.
- Tests: 14 service scenarios (9 updated + 5 new) all PASS, 8 integration tests (ip round-trip, caller-scoped list/revoke) all PASS, 8 handler tests (capture, 404 no-leak, 401 no-jwt) all PASS, boot test asserts routes present. Coverage 85.2%.
- Adversarial probes: 6 RED-confirmed (cross-user revoke/list scoping, ip+ua capture, protected-group requirement, revoked-session refresh invalid, strPtr empty→nil).

Frontend (PR-B: 760a0bb):
- SessionResponse DTO in auth.res.dto.ts (exact match with backend Go).
- authService: getMySessions() → GET /auth/sessions/me, revokeSession(id) → DELETE /auth/sessions/:id. Both use existing httpClient + interceptor Bearer (sessions NOT skipped).
- profile.component.ts: OnInit + sessions/loadingSessions signals, onRevoke method, confirm dialog via UiDialogService, list reload on success.
- profile.component.html: "Sesiones activas" section with p-table (ip, userAgent/browserOf helper, createdAt), per-row revoke, empty state, skeleton while loading.
- profile.component.sass: --page-* tokens only (Cyanotype Atelier pattern).
- profile.component.spec.ts: 7 tests (init, empty state, confirm flows) all PASS; confirm→true tests mutations RED-confirmed.
- Tests: 322 total vitest all PASS, tsc clean, ng build clean, lint clean.

**Acceptance:** ✅
- LoginWithGoogle + Refresh populate ip + user_agent on refresh_token row (captured via c.ClientIP() + c.Request.UserAgent()).
- GET /api/auth/sessions/me returns only caller's active sessions (revoked/expired excluded) — caller-scoped at SQL layer.
- DELETE /api/auth/sessions/:id revokes only caller's own session; cross-user/nonexistent/already-revoked → 404 (no existence leak).
- Profile UI lists active sessions with ip/browser/date and per-row revoke with confirm.
- Revoked session cannot refresh (existing ErrInvalidRefreshToken regression test).
- Manual smoke test: re-login populates ip/ua; profile "Sesiones activas" lists sessions; revoke works; new session appears after re-login.

**Key Learnings Captured:**
1. ip/user_agent are forensic/informational only — NOT a security boundary. Stolen token works from any IP until replay detection fires.
2. strPtr empty→nil conversion: Postgres inet column rejects pointer-to-empty-string; must convert to NULL for safe casting.
3. Caller-scoping at SQL layer (WHERE user_id = caller) in BOTH list and revoke is the dominant security property — verified by adversarial probes.
4. Routes MUST be on protected group (JWT middleware). Pre-auth placement would silently break scoping (UserIDFrom empty).
5. Revoke 404 for cross-user/nonexistent/already-revoked avoids existence leaks.
6. Revoking current session is allowed (logout-this-device pattern); access token valid until exp, next refresh fails.
7. Boot test needed nil-safe auth.Service stub (mirrors nilCourseSvc pattern).
8. NO migration required (first C8+ change without schema step-drift).

**Scope realizado:**
- ✅ IP/UA capture at handler layer → service → repo → persist
- ✅ GET /api/auth/sessions/me (caller-scoped active list)
- ✅ DELETE /api/auth/sessions/:id (caller-scoped revoke)
- ✅ Frontend sessions section in profile page (p-table + confirm dialog)
- ✅ Full test coverage (85.2% backend service, 322 vitest frontend)
- ✅ 6 adversarial probes RED-confirmed (scoping, capture, routing, refresh invalid)
- ✅ Manual smoke test PASSED

**Scope OUT (deferred follow-ups):**
- "Current session" flag (D5 — JWT alone cannot map to refresh_token row; needs token hint coupling)
- last_used_at per-request tracking (hot-path write, deferred to C8.x)
- Expired-token cleanup cron (ops task)
- Geo-IP / ASN lookup (informational, post-MVP)

---

#### C8.2 — `pagination-policy` (planned)

**Por que:** offset pagination (LIMIT/OFFSET) no escala bien con > 10k filas. Cuando el catalogo crezca, queries se vuelven lentas.

**Scope IN:**
- Cursor pagination para los listados grandes (catalogo, users, certificates)
- Encoder/decoder de cursor opaco (base64 de `{id, created_at}`)
- API: aceptar `?cursor=...&size=...` en vez de `?page&size` para esos endpoints
- Documentar en `GUIA-BACK.md` cuando usar cursor vs offset

**Acceptance:**
- Catalogo de 10k cursos pagina con cursor en < 200ms
- Frontend service migrado a cursor (al menos catalog)

---

#### C8.3 — `prod-deployment` (planned)

**Por que:** todo el setup actual es dev. Para deploy real necesita secrets management, TLS, observabilidad.

**Scope IN:**
- Helm chart o Docker Compose con override de prod
- Secrets desde Vault / AWS Secrets Manager / GCP Secret Manager (decidir)
- nginx con TLS (cert-manager + Let's Encrypt) o detras de un LB
- Postgres en servicio gestionado (RDS / Cloud SQL) — NO en container en prod
- Object Storage real (S3 / GCS) en lugar de MinIO
- Observabilidad: Prometheus + Grafana o Datadog. Logs a Loki / CloudWatch
- Healthchecks de Kubernetes (liveness + readiness + startup probes)
- Backup automatico de Postgres

**Acceptance:**
- Deploy a un entorno staging con dominio HTTPS
- Logs centralizados con dashboards basicos
- Postgres con backups diarios y RTO/RPO documentados

---

## 5. Orden recomendado para llegar al MVP demo

Para ver el **happy path completo** funcionando (alumno se inscribe, ve curso, rinde evaluacion, recibe certificado) en el menor tiempo posible:

```
C0.1 testing-foundation
   ↓
C1.1 module-users
   ↓
C2.1 module-courses-domain
   ↓
C2.2 module-courses-content
   ↓
C3.1 module-evaluations-design
   ↓
C4.1 module-approvals
   ↓
C2.4 module-courses-catalog       ← aca recien el alumno ve los cursos
   ↓
C3.2 module-evaluations-attempts  ← rinde
   ↓
C5.1 module-certificates          ← recibe certificado
   ↓
C0.2 api-types-codegen + C2.3 module-courses-material + C6.1 module-reporting  ← consolidacion final
```

**Total estimado al MVP demo (10 changes):** ~5500 LOC en 10-20 sesiones de SDD.

---

## 6. Como arrancar un SDD change

Para cualquiera de los changes listados:

```bash
# 1. Confirmar contexto (en cualquier sesion nueva)
mem_search "sdd-init/skillmaker"

# 2. Arrancar el change (modo interactive recomendado)
/sdd-new module-<nombre>

# 3. El orchestrator lanza:
#    propose (opus) → spec (sonnet) → tasks (sonnet) → apply (sonnet, en work-units) → verify (sonnet) → archive (haiku)

# 4. Cada fase pausa para review en interactive mode
```

**Antes de empezar un change:**

1. Leer la seccion correspondiente de este roadmap
2. Confirmar que las dependencias (changes previos) estan `archived` en engram
3. Tener claras las decisiones pendientes que el `propose` va a preguntar
4. Tener `make dev` corriendo para validar end-to-end al final de cada work-unit

**Cuando archivar un change:**

1. `sdd-verify` retorna `passed` o `passed_with_warnings` con warnings resueltos
2. Smoke test manual contra el endpoint/UI nuevo pasa
3. `make test && make lint` green
4. `sdd-archive` cierra el ciclo y actualiza este archivo (la tabla de la seccion 1)

---

## 7. Deuda tecnica acumulada (corregible en cualquier momento)

| Deuda | Detectada en | Impacto | Prioridad |
|-------|--------------|---------|-----------|
| IP/UserAgent del refresh_token quedan NULL | scaffold-frontend-base smoke test | Sin forense ante session compromise | Baja (C7.1) |
| 0 tests automatizados escritos | scaffold-backend-base verify | Cada feature acumula riesgo | **Alta (C0.1)** |
| Frontend sin tipos generados desde swagger | scaffold-frontend-base verify (suggestion) | Posible drift BE/FE | Media (C0.2) |
| Bundle inicial 657kB > 500kB budget | scaffold-frontend-base verify (suggestion) | Performance inicial de carga | Baja (mejora con lazy modules reales) |
| `npm install` en Makefile (vs `npm ci`) | scaffold-frontend-base verify | DX vs reproducibilidad | Aceptado (Dockerfile usa npm ci) |
| `RevokeChain` del auth.repo llama a `RevokeAllForUser` | scaffold-backend-base apply (risk #2) | Granularidad de revocacion forense | Baja (post-MVP) |
| `.env.example` con `postgres:postgres` no coincidia con docker-compose | smoke test | Friccion onboarding | Resuelto (sed manual en .env real) |

---

## 8. Notas para nuevas sesiones de Claude

Si arrancas una sesion nueva y el contexto se perdio:

1. **Leer este archivo primero** (es el indice maestro del proyecto)
2. **Consultar engram**: `mem_search "sdd-init/skillmaker"` para el contexto base
3. **Estado git**: `git log --oneline | head -20` muestra que se hizo
4. **Estado SDD**: `mem_search "sdd/scaffold"` lista los changes archivados con sus reports
5. **No re-leer las 4 guias completas** a menos que el change especifico lo requiera; usar el skill registry en `.atl/skill-registry.md` para el contexto compacto

**Convenciones que vale la pena recordar:**

- 4 roles fijos (NO permisos granulares): `alumno | creador | supervisor | administrador`
- JWT incluye `sub, email, nombre, roles, exp, iat`. Refresh token es opaco, hasheado SHA-256 en DB.
- Backend: monolito modular, comunicacion cross-modulo SOLO via interfaces (nunca tablas)
- Frontend: standalone + zoneless. Signals para estado reactivo. Tailwind 3 (no v4).
- Migrations: golang-migrate con up.sql/down.sql versionado. PROHIBIDO AutoMigrate en prod.
- DTOs separados de modelos GORM. JSON: snake_case para tokens (OAuth spec), camelCase para resto.
- Errores de dominio: `var ErrXxx = errors.New(...)`, traducir a HTTP con `errors.Is` en el handler.
