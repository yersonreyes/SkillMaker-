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

**Fecha del snapshot:** 2026-06-01
**Commits totales del scaffold:** 16
**LOC totales (Go + TS + SQL):** ~2700

### Modulos del dominio (los 7 declarados en RT)

| Modulo | Estado | Tablas SQL existentes |
|--------|--------|----------------------|
| `auth` | ✅ Completo | `refresh_token` |
| `users` | 🟡 Stub minimo (`UpsertFromGoogle`, `GetByID`, `LoadRoleNames`) | `user`, `role`, `user_role` |
| `courses` | ❌ No existe | ninguna |
| `evaluations` | ❌ No existe | ninguna |
| `approvals` | ❌ No existe | ninguna |
| `certificates` | ❌ No existe | ninguna |
| `reporting` | ❌ No existe | — (read-only) |

### Frontend pages

| Page | Estado |
|------|--------|
| `pages/auth/login` | ✅ Funcional (GIS + JWT + retry) |
| `pages/auth/callback` | ✅ Stub minimo (compat futuro redirect-flow) |
| `pages/platform/catalog` | 🟡 Stub "Pendiente" |
| `pages/platform/my-courses` | 🟡 Stub "Pendiente" |
| `pages/platform/profile` | ✅ Funcional (read-only desde JWT) |
| `pages/platform/{certificates,badges,courses/:id,evaluations/:id}` | 🟡 Routes → `PendingViewComponent` |
| `pages/platform/creator/*` | 🟡 Routes → `PendingViewComponent` |
| `pages/platform/admin/*` | 🟡 Routes → `PendingViewComponent` |
| `pages/platform/supervisor/*` | 🟡 Routes → `PendingViewComponent` |

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
| C5.1 | [`module-certificates`](#c51--module-certificates) | 5 | ~700 | C3.2 | RF-21, RF-22 |
| C6.1 | [`module-reporting`](#c61--module-reporting) | 6 | ~500 | C2.x, C3.x, C5.1 | RF-23, RF-24, RF-25 |
| C7.1 | [`refresh-token-tracking`](#c71--refresh-token-tracking) | 7 | ~200 | — | (hardening) |
| C7.2 | [`pagination-policy`](#c72--pagination-policy) | 7 | ~300 | — | (escala) |
| C7.3 | [`prod-deployment`](#c73--prod-deployment) | 7 | ~500 | — | (deploy) |

**Total estimado al MVP completo (C0-C6):** ~7000 LOC en 13 changes.

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

#### C2.4 — `module-courses-catalog`

**Por que:** sin esto, los alumnos no ven los cursos. Implementa el catalogo publico + detalle + inscripcion.

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

#### C3.1 — `module-evaluations-design`

**Por que:** sin evaluacion no hay puntaje. Este change le da al creador la capacidad de definir el examen.

**Scope IN:**

Backend:
- Migration `0003_add_evaluations.up.sql`:
  - `evaluation` (id, course_id UQ, nota_minima, intentos_max)
  - `question` (id, evaluation_id, enunciado, tipo, puntaje, orden) — `tipo IN ('opcion_multiple','verdadero_falso')`
  - `option` (id, question_id, texto, correcta)
  - `attempt` (id, user_id, evaluation_id, numero, puntaje, aprobado, iniciado_en, finalizado_en)
  - `answer` (id, attempt_id, question_id, option_id, correcta) — schema solo, endpoints en C3.2
- Modulo `evaluations/`:
  - Repo + service + handlers
  - Endpoints (todos `RequireRole("creador")` con ownership check del course):
    - `POST /api/courses/:courseId/evaluation` — crea evaluacion (1-1 con course, UQ)
    - `PATCH /api/evaluations/:id` — actualiza `nota_minima`, `intentos_max`
    - `POST /api/evaluations/:id/questions` — agrega pregunta
    - `PATCH /api/questions/:id`
    - `DELETE /api/questions/:id`
    - Para verdadero_falso, el backend auto-crea las 2 opciones (V y F) al crear la pregunta
    - Para opcion_multiple, endpoints `/api/questions/:id/options` para CRUD de opciones
  - Validacion: minimo 1 opcion `correcta=true` por pregunta opcion_multiple

Frontend:
- Service `EvaluationService` + `QuestionService`
- Page `creator/evaluation-edit/:courseId`:
  - Form de la evaluacion (nota minima, intentos maximos)
  - Lista de preguntas con tipo + puntaje
  - Modal de pregunta: enunciado, tipo (select), opciones dinamicas con radio "correcta"
  - Validacion client-side: minimo 1 correcta

**Scope OUT:**
- Pregunta de respuesta abierta o multiple-choice (varias correctas) — el modelo de datos limita a una sola
- Banco de preguntas reutilizables

**Acceptance:**
- Creador define evaluacion con nota minima 70
- Creador agrega 5 preguntas: 3 opcion_multiple + 2 verdadero_falso
- La evaluacion esta lista para ser rendida (la rendicion viene en C3.2)

---

#### C3.2 — `module-evaluations-attempts`

**Por que:** RF-13 exige que el alumno pueda rendir. Este change cierra el flujo de aprendizaje.

**Scope IN:**

Backend:
- Endpoints:
  - `POST /api/evaluations/:id/attempts` — inicia intento. Valida `intentos_max` no excedido para ese user. Crea fila `attempt` con `iniciado_en=now`, `numero=N+1`
  - `GET /api/attempts/:id` — devuelve estado + preguntas (sin las respuestas correctas, eso es post-finalizacion)
  - `POST /api/attempts/:id/answers` — body `{ questionId, optionId }` registra una respuesta (UQ por attempt+question)
  - `POST /api/attempts/:id/submit` — finaliza: setea `finalizado_en`, calcula `puntaje` y `aprobado` segun `nota_minima`. Si `aprobado=true`, llama a `courses.MarkEnrollmentCompleted(...)` y a `certificates.IssueOnPass(...)` (interface call cross-modulo).
- Calculo de puntaje: suma de `puntaje` de las preguntas donde la respuesta es `correcta=true`, dividido por suma total * 100. Aprobado si >= `nota_minima`.

Frontend:
- Service `AttemptService`
- Page `evaluation/:courseId`:
  - Step 1: confirmacion "Iniciar intento N de M"
  - Step 2: una pregunta a la vez (o todas, decidir por UX)
  - Radio buttons para opciones
  - Boton "Siguiente" / "Finalizar"
  - Al finalizar: muestra puntaje + estado (aprobado/desaprobado) + boton "Volver al curso"

**Scope OUT:**
- Timer para limitar duracion del intento (no esta en RFs)
- Mostrar respuestas correctas tras desaprobar (decidir UX)
- Save & resume (si el alumno cierra sin submitir, el attempt queda abierto)

**Acceptance:**
- Alumno inicia attempt → ve preguntas
- Alumno responde todas → submit → recibe puntaje
- Si aprobado, su `enrollment.completado` queda en true Y se le emite certificado (cross-modulo con C5.1)
- Si intentos_max alcanzado, el endpoint `POST /attempts` devuelve 409

---

### Capa 4 — Approvals

#### C4.1 — `module-approvals`

**Por que:** RF-15 exige que el admin apruebe los cursos antes de que aparezcan en el catalogo. Este change desbloquea el catalogo publico.

**Scope IN:**

Backend:
- Migration `0004_add_approvals.up.sql`:
  - `approval` (id, course_id, admin_id, resultado, comentario, resuelto_en) — `resultado IN ('aprobado','rechazado')`
- Endpoints:
  - `POST /api/courses/:id/submit` (creador) — cambia estado a `en_revision`. Valida que tenga al menos 1 video + 1 evaluacion (`HasContent` y `evaluations.GetByCourseID`).
  - `GET /api/approvals/pending` (admin) — lista de cursos `estado='en_revision'`
  - `POST /api/courses/:id/approve` (admin) — body `{ comentario? }`. Crea fila `approval` + cambia estado a `aprobado` + setea `publicado_en`
  - `POST /api/courses/:id/reject` (admin) — body `{ comentario }` (comentario REQUERIDO en rechazo). Crea fila `approval` + cambia estado a `rechazado`
  - `GET /api/courses/:id/approvals` — historial de revisiones (RF-17b)
- Notificaciones: por ahora "in-app pasivo" (el creador ve el estado cambiar cuando entra a `my-content`). Email queda fuera de scope.

Frontend:
- Service `ApprovalService`
- Page `admin/approvals`:
  - Tabla de cursos pendientes con preview (titulo, creador, fecha submit)
  - Click → drawer/detail con preview completo (secciones, videos, material, evaluacion)
  - Botones "Aprobar" (confirm) / "Rechazar" (modal con comentario obligatorio)
- Page `creator/my-content`:
  - Boton "Enviar a revision" cuando estado es `borrador` o `rechazado`
  - Si estado `rechazado`, mostrar el ultimo comentario del admin
  - Historial colapsable de revisiones

**Scope OUT:**
- Sistema de notificaciones email (RF-17 menciona canal "por definir")
- Aprobacion por consenso (multiple admins votan) — RF-15 dice "cualquier admin disponible"
- Re-aprobacion automatica tras edicion menor

**Acceptance:**
- Creador con curso en borrador + contenido + evaluacion → submit OK
- Admin ve el curso en bandeja → aprueba → curso aparece en catalogo
- Admin rechaza con comentario → creador ve el comentario → puede editar + reenviar
- Historial de revisiones queda persistido

---

### Capa 5 — Certificates

#### C5.1 — `module-certificates`

**Por que:** RF-21 (certificado descargable) + RF-22 (insignias + ranking). Cierra el ciclo de gamificacion del LMS.

**Scope IN:**

Backend:
- Migration `0005_add_certificates.up.sql`:
  - `certificate` (id, user_id, attempt_id UQ, course_id, codigo UQ, emitido_en, storage_key opcional)
  - `badge` (id, nombre UQ, descripcion, icono_key)
  - `user_badge` (user_id, badge_id, otorgado_en) — PK compuesta
- Seed de badges iniciales (ejemplo: "Primer curso completado", "5 cursos completados", "Top 10 del ranking")
- Endpoints:
  - `GET /api/certificates/me` — listado del user
  - `GET /api/certificates/:id` — detalle (incluye URL prefirmada al PDF)
  - `GET /api/certificates/:id/download` — devuelve URL prefirmada GET al PDF
  - `GET /api/badges/me` — insignias del user
  - `GET /api/badges/ranking` — top N por cantidad de certificados (o badges) — query agregada
- Service:
  - `IssueOnPass(ctx, attemptID)` llamado por `evaluations.SubmitAttempt`. Genera PDF, guarda en MinIO, persiste fila en `certificate`. Idempotente: si ya existe certificate para ese attempt, no duplicar.
  - PDF generation con `github.com/go-pdf/fpdf` o `github.com/jung-kurt/gofpdf`. Template simple: logo, nombre del user, nombre del curso, fecha, codigo de verificacion.
  - `EvaluateBadges(ctx, userID)` evalua si gano nuevas badges (lazy, llamado tras emit certificate).

Frontend:
- Service `CertificateService` y `BadgeService`
- Page `certificates` (reemplaza stub):
  - Grid de tarjetas: nombre del curso, fecha, codigo, boton "Descargar PDF"
- Page `badges` (reemplaza stub):
  - Grid de insignias ganadas + tabla del ranking top N
- En `course-detail` de un curso completado, mostrar boton "Mi certificado" si existe

**Scope OUT:**
- Verificacion publica de certificado por codigo (endpoint publico `/verify/:codigo` — post-MVP)
- Compartir en redes sociales
- NFTs / blockchain (no, gracias)
- Personalizacion del template PDF por curso

**Acceptance:**
- Alumno aprueba evaluacion → `certificate` aparece en su listado con codigo unico
- Descarga PDF → archivo abre correctamente con nombre + curso + fecha + codigo
- Re-submitir el mismo attempt no duplica certificados
- Badge "Primer curso completado" aparece automaticamente
- Ranking muestra top 10 con cantidad de certificados

---

### Capa 6 — Reporting

#### C6.1 — `module-reporting`

**Por que:** RF-23 (admin), RF-24 (supervisor), RF-25 (reportes globales). Read-only, sin schema propio — vistas SQL agregadas.

**Scope IN:**

Backend:
- Modulo `internal/modules/reporting/` (lectura cruzada read-only, unica excepcion a la regla de no joins cross-modulo)
- Endpoints:
  - `GET /api/reports/global` (admin) — totales del sistema: usuarios activos, cursos por estado, intentos, certificados emitidos, top creadores
  - `GET /api/reports/courses` (admin) — desempeno por curso: inscripciones, intentos, tasa de aprobacion
  - `GET /api/reports/users/:id/progress` (admin o el mismo user) — progreso individual
  - `GET /api/reports/team` (supervisor) — empleados a cargo + sus cursos + puntajes (join via `supervision`)
- Queries: SQL puro (CTE o JOIN) para mantener performance. NO usar GORM eager loading para esto.
- Cache opcional: si las queries son costosas, agregar Redis o materialized view. NO en MVP.

Frontend:
- Service `ReportingService`
- Page `admin/global-reports`:
  - Cards con metricas globales
  - Charts con PrimeNG `Chart` (Chart.js wrapper) — usuarios por mes, cursos aprobados por mes
- Page `admin/reports` (variante de global, mas detallado por curso)
- Page `supervisor/team-progress`:
  - Tabla de empleados a cargo
  - Por empleado: cursos inscritos, completados, ultimo intento, puntaje promedio

**Scope OUT:**
- Export a Excel/CSV (post-MVP)
- Email automatico de reporte semanal
- Custom dashboards configurables

**Acceptance:**
- Admin ve metricas globales actualizadas
- Admin filtra reporte por curso y ve tasa de aprobacion
- Supervisor ve solo su equipo (no otros supervisores)
- Performance: cada endpoint < 2s con 1000 users + 100 cursos + 5000 attempts

---

### Capa 7 — Hardening post-MVP

#### C7.1 — `refresh-token-tracking`

**Por que:** las columnas `ip` y `user_agent` del `refresh_token` quedan NULL hoy. Capturarlas habilita forense ante incidentes (sesion comprometida, revocacion por dispositivo).

**Scope IN:**
- Handler `LoginWithGoogle` y `Refresh` capturan `c.ClientIP()` y `c.Request.UserAgent()` y los pasan al service
- Service los persiste en la fila de `refresh_token`
- Endpoint `GET /api/auth/sessions/me` — listado de sesiones activas del user con IP + user agent + fecha
- Endpoint `DELETE /api/auth/sessions/:id` — revocar una sesion especifica

**Acceptance:**
- Al loguear desde Chrome desktop, la fila tiene `user_agent` con "Chrome/..." y `ip` con la IP del request
- User ve sus sesiones activas y puede cerrar una remotamente

---

#### C7.2 — `pagination-policy`

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

#### C7.3 — `prod-deployment`

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
