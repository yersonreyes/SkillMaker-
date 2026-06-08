// Package handler — catalog handler for alumno-facing C2.4 endpoints.
// All routes require a valid JWT (protected group) — NO RequireRole restriction.
// Route table:
//
//	GET  /catalog             → ListCatalog
//	GET  /catalog/:id         → GetCatalogDetail (discriminated: preview or full tree)
//	POST /catalog/:id/enroll  → Enroll (idempotent)
//	GET  /users/me/courses    → ListMyCourses
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// Package-doc update: added MarkVideoProgress route (course-player-progress, Change 2).
// Route table updated:
//
//	GET  /catalog             → ListCatalog
//	GET  /catalog/:id         → GetCatalogDetail (discriminated: preview or full tree)
//	POST /catalog/:id/enroll  → Enroll (idempotent)
//	GET  /users/me/courses    → ListMyCourses
//	GET  /categorias          → ListCategorias
//	PUT  /videos/:id/progress → MarkVideoProgress (alumno, enrolled-gated)

// CatalogHandler holds the service dependency for catalog + enrollment endpoints.
type CatalogHandler struct {
	svc service.Service
}

// RegisterCatalog mounts the alumno-facing catalog routes + categorias onto protectedGrp.
// The group must already carry JWT middleware (protected) — NO RequireRole.
// CRITICAL: these routes use /catalog/* and /users/me/courses — NEVER /courses/*
// (that namespace is owned by creatorGrp and would Gin-panic on duplicate registration).
func RegisterCatalog(protectedGrp *gin.RouterGroup, svc service.Service) {
	h := &CatalogHandler{svc: svc}

	// Catalog routes (/catalog/* — new prefix, no conflict with /courses/*).
	protectedGrp.GET("/catalog", h.ListCatalog)
	protectedGrp.GET("/catalog/:id", h.GetCatalogDetail)
	protectedGrp.POST("/catalog/:id/enroll", h.Enroll)

	// My-courses route (static path — no conflict with /users/:id on adminGrp
	// because that group is distinct from protected).
	protectedGrp.GET("/users/me/courses", h.ListMyCourses)

	// Categorias (course-structure-v2): curated category list for any authenticated user.
	protectedGrp.GET("/categorias", h.ListCategorias)

	// Video progress (course-player-progress, Change 2): alumno-facing, enrolled-gated.
	// PUT is a new method+path on /videos/:id/progress — no conflict with creatorGrp
	// PATCH/DELETE /videos/:id or POST/GET /videos/:id/materials* (design §5 confirmed).
	protectedGrp.PUT("/videos/:id/progress", h.MarkVideoProgress)
}

// RegisterCategoriasAdmin mounts the admin-only categoria CRUD routes onto adminGrp
// (which already carries JWT + RequireRole("administrador")).
//
//	POST   /categorias       → CreateCategoria
//	PATCH  /categorias/:id    → UpdateCategoria
//	DELETE /categorias/:id    → DeleteCategoria
//
// GET /categorias stays on the protected group (any authenticated user) via RegisterCatalog.
func RegisterCategoriasAdmin(adminGrp *gin.RouterGroup, svc service.Service) {
	h := &CatalogHandler{svc: svc}
	adminGrp.POST("/categorias", h.CreateCategoria)
	adminGrp.PATCH("/categorias/:id", h.UpdateCategoria)
	adminGrp.DELETE("/categorias/:id", h.DeleteCategoria)
}

// renderCategoriaError maps categoria CRUD sentinels to HTTP responses.
func renderCategoriaError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCategoriaNotFound):
		httperr.Render(c, httperr.NotFound("CATEGORIA_NOT_FOUND", "categoria not found"))
	case errors.Is(err, service.ErrCategoriaDuplicate):
		httperr.Render(c, httperr.Conflict("CATEGORIA_DUPLICATE", "ya existe una categoria con ese nombre"))
	case errors.Is(err, service.ErrCategoriaInUse):
		httperr.Render(c, httperr.Conflict("CATEGORIA_IN_USE", "la categoria esta asignada a uno o mas cursos"))
	default:
		httperr.Render(c, httperr.Internal(err.Error()))
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// validNivel reports whether nivel is a valid closed-allow-list value (or empty = no filter).
// ADR-4: handler owns validation; empty = no filter; invalid = 400.
func validNivel(s string) bool {
	return s == "" || s == "basico" || s == "intermedio" || s == "avanzado"
}

// validSort reports whether sort is a valid closed-allow-list value (or empty = default).
// ADR-4: handler owns validation; empty = default recientes; invalid = 400.
func validSort(s string) bool {
	return s == "" || s == "recientes" || s == "titulo"
}

// ListCatalog godoc
// @Summary     Lista cursos aprobados (catálogo público para alumnos)
// @Description Retorna una página de cursos con estado='aprobado'. Soporta ?page, ?size, ?q (ILIKE titulo),
//
//	?nivel (basico|intermedio|avanzado), ?categoria (UUID, repetible, OR), ?sort (recientes|titulo).
//
// @Tags        catalog
// @Produce     json
// @Security    BearerAuth
// @Param       page     query int      false "Página (default 1)"
// @Param       size     query int      false "Tamaño de página (max 100, default 20)"
// @Param       q        query string   false "Filtro ILIKE en titulo"
// @Param       nivel    query string   false "basico|intermedio|avanzado"
// @Param       categoria query []string false "UUID(s) de categoría, repetible → semántica OR"
// @Param       sort     query string   false "recientes|titulo (default recientes)"
// @Success     200 {object} object "Página de CatalogCourseCard (items, page, size, total, totalPages)"
// @Failure     400 {object} httperr.Error "filtro inválido (nivel o sort fuera del allow-list)"
// @Failure     401 {object} httperr.Error
// @Failure     500 {object} httperr.Error
// @Router      /catalog [get]
func (h *CatalogHandler) ListCatalog(c *gin.Context) {
	p := pagination.ParseParams(c)

	nivel := c.Query("nivel")
	sort := c.Query("sort")
	cats := c.QueryArray("categoria") // repeated param: ?categoria=A&categoria=B (ADR-5)

	// Validate closed allow-lists → 400 on invalid value (ADR-4).
	if !validNivel(nivel) {
		httperr.Render(c, httperr.BadRequest("INVALID_FILTER", "nivel inválido: debe ser basico, intermedio o avanzado"))
		return
	}
	if !validSort(sort) {
		httperr.Render(c, httperr.BadRequest("INVALID_FILTER", "sort inválido: debe ser recientes o titulo"))
		return
	}
	// Validate categoria values are well-formed UUIDs (REQ-FILTER-CATEGORIA + REQ-SEC).
	// A malformed value is rejected immediately (400); a well-formed but nonexistent UUID
	// passes through and produces an empty result (match-nothing — ADR-4).
	for _, cat := range cats {
		if _, err := uuid.Parse(cat); err != nil {
			httperr.Render(c, httperr.BadRequest("INVALID_FILTER", "categoria inválida: cada valor debe ser un UUID válido"))
			return
		}
	}

	// Default empty sort to "recientes" before forwarding.
	if sort == "" {
		sort = "recientes"
	}

	filter := service.CatalogFilter{
		Q:            c.Query("q"),
		Nivel:        nivel,
		CategoriaIDs: cats,
		Sort:         sort,
	}

	page, err := h.svc.ListCatalog(c.Request.Context(), p, filter)
	if err != nil {
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.ToCatalogCardPage(page))
}

// GetCatalogDetail godoc
// @Summary     Detalle de un curso aprobado (alumno)
// @Description Retorna preview (enrolled=false) o árbol completo (enrolled=true) según inscripción.
//
//	Cursos no-aprobados → 404 (draft-invisibility).
//
// @Tags        catalog
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID del curso"
// @Success     200 {object} dto.CoursePreviewResponse "preview — no secciones ni materiales"
// @Success     200 {object} dto.CourseDetailAlumnoResponse "enrolled — árbol completo"
// @Failure     401 {object} httperr.Error
// @Failure     404 {object} httperr.Error "curso no encontrado o no aprobado"
// @Failure     500 {object} httperr.Error
// @Router      /catalog/{id} [get]
func (h *CatalogHandler) GetCatalogDetail(c *gin.Context) {
	courseID := c.Param("id")
	userID := middleware.UserIDFrom(c)

	d, err := h.svc.GetCatalogDetail(c.Request.Context(), userID, courseID)
	if err != nil {
		renderCatalogError(c, err)
		return
	}

	// Structural no-leak: branch on Enrolled to return TWO distinct struct types.
	// CoursePreviewResponse has NO secciones/materiales field — structural absence (AC-9).
	if !d.Enrolled {
		c.JSON(http.StatusOK, dto.ToCoursePreview(d))
	} else {
		c.JSON(http.StatusOK, dto.ToCourseDetailAlumno(d))
	}
}

// Enroll godoc
// @Summary     Inscribe al alumno en un curso aprobado (idempotente)
// @Description Crea una fila de enrollment. Idempotente: segunda llamada devuelve 200 sin duplicar.
//
//	Curso no-aprobado → 404 (draft-invisibility).
//
// @Tags        catalog
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID del curso"
// @Success     200 {object} dto.EnrollmentResponse
// @Failure     401 {object} httperr.Error "JWT requerido"
// @Failure     404 {object} httperr.Error "curso no encontrado o no aprobado"
// @Failure     500 {object} httperr.Error
// @Router      /catalog/{id}/enroll [post]
func (h *CatalogHandler) Enroll(c *gin.Context) {
	courseID := c.Param("id")
	userID := middleware.UserIDFrom(c)

	if userID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	if err := h.svc.Enroll(c.Request.Context(), userID, courseID); err != nil {
		renderCatalogError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.EnrollmentResponse{
		CourseID: courseID,
		Enrolled: true,
	})
}

// ListMyCourses godoc
// @Summary     Lista los cursos en los que está inscrito el alumno autenticado
// @Description Retorna los enrollments del caller con titulo, creadorNombre, completado, inscritoEn.
//
//	Ordered por inscritoEn DESC. Scoped estrictamente al caller (JWT userID).
//
// @Tags        catalog
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array}  dto.MyCourseItem
// @Failure     401 {object} httperr.Error "JWT requerido"
// @Failure     500 {object} httperr.Error
// @Router      /users/me/courses [get]
func (h *CatalogHandler) ListMyCourses(c *gin.Context) {
	userID := middleware.UserIDFrom(c)

	if userID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	rows, err := h.svc.ListMyCourses(c.Request.Context(), userID)
	if err != nil {
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.ToMyCourseItems(rows))
}

// ListCategorias godoc
// @Summary     Lista las categorias curadas
// @Description Returns all curated categories. JWT required; any authenticated role (no role gate).
// @Tags        categorias
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array}  dto.CategoriaResponse
// @Failure     401 {object} httperr.Error
// @Router      /categorias [get]
func (h *CatalogHandler) ListCategorias(c *gin.Context) {
	cats, err := h.svc.ListCategorias(c.Request.Context())
	if err != nil {
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}
	resp := make([]dto.CategoriaResponse, 0, len(cats))
	for _, cat := range cats {
		resp = append(resp, dto.ToCategoria(cat))
	}
	c.JSON(http.StatusOK, resp)
}

// CreateCategoria godoc
// @Summary     Crea una categoria (admin)
// @Tags        categorias
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body dto.CategoriaCreateRequest true "Nombre de la categoria"
// @Success     201 {object} dto.CategoriaResponse
// @Failure     400 {object} httperr.Error
// @Failure     409 {object} httperr.Error
// @Router      /categorias [post]
func (h *CatalogHandler) CreateCategoria(c *gin.Context) {
	var req dto.CategoriaCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}
	cat, err := h.svc.CreateCategoria(c.Request.Context(), req.Nombre)
	if err != nil {
		renderCategoriaError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.ToCategoria(*cat))
}

// UpdateCategoria godoc
// @Summary     Renombra una categoria (admin)
// @Tags        categorias
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path string                     true "UUID de la categoria"
// @Param       body body dto.CategoriaUpdateRequest true "Nuevo nombre"
// @Success     200 {object} dto.CategoriaResponse
// @Failure     400 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Failure     409 {object} httperr.Error
// @Router      /categorias/{id} [patch]
func (h *CatalogHandler) UpdateCategoria(c *gin.Context) {
	id := c.Param("id")
	var req dto.CategoriaUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}
	cat, err := h.svc.UpdateCategoria(c.Request.Context(), id, req.Nombre)
	if err != nil {
		renderCategoriaError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToCategoria(*cat))
}

// DeleteCategoria godoc
// @Summary     Elimina una categoria (admin). Bloqueado si esta asignada a cursos.
// @Tags        categorias
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID de la categoria"
// @Success     204
// @Failure     404 {object} httperr.Error
// @Failure     409 {object} httperr.Error
// @Router      /categorias/{id} [delete]
func (h *CatalogHandler) DeleteCategoria(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteCategoria(c.Request.Context(), id); err != nil {
		renderCategoriaError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ── Error render helper ────────────────────────────────────────────────────────

// renderCatalogError maps catalog service sentinels to HTTP statuses.
// Catalog-facing reads only produce ErrCourseNotFound (missing or not aprobado → 404).
// Reuses the read semantics of renderCourseErrorRead but is inlined here to keep the
// catalog handler independent (no coupling to the creador handler's error helper).
func renderCatalogError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrCourseNotFound) {
		httperr.Render(c, httperr.NotFound("COURSE_NOT_FOUND", "course not found"))
		return
	}
	httperr.Render(c, httperr.Internal(err.Error()))
}

// MarkVideoProgress godoc
// @Summary     Registra o actualiza el progreso del alumno en un video
// @Description PUT /api/videos/:id/progress — upsert caller-scoped video progress.
//
//	El alumno debe estar inscrito en el curso del video; de lo contrario → 404 (no-leak, D3).
//	El userId del cuerpo es IGNORADO; siempre se usa el userID del JWT (REQ-SEC).
//
// @Tags        catalog
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path     string                      true "UUID del video"
// @Param       body body     dto.VideoProgressRequest    true "Progreso del video"
// @Success     204  "sin contenido — progreso registrado"
// @Failure     400  {object} httperr.Error "cuerpo inválido"
// @Failure     401  {object} httperr.Error "JWT requerido"
// @Failure     404  {object} httperr.Error "video no encontrado o no inscrito (no-leak)"
// @Failure     500  {object} httperr.Error
// @Router      /videos/{id}/progress [put]
func (h *CatalogHandler) MarkVideoProgress(c *gin.Context) {
	userID := middleware.UserIDFrom(c)
	if userID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	videoID := c.Param("id")

	var req dto.VideoProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", err.Error()))
		return
	}

	pos := 0
	if req.LastPositionS != nil {
		pos = *req.LastPositionS
	}

	if err := h.svc.MarkVideoProgress(c.Request.Context(), userID, videoID, req.Completado, pos); err != nil {
		renderProgressError(c, err)
		return
	}

	c.Status(http.StatusNoContent) // 204 — idempotent write, no body
}

// renderProgressError maps video-progress service sentinels to HTTP statuses.
// ErrVideoNotFound + ErrNotEnrolled both → 404 (no-leak, D3: caller cannot distinguish the two cases).
func renderProgressError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrVideoNotFound), errors.Is(err, service.ErrNotEnrolled):
		httperr.Render(c, httperr.NotFound("VIDEO_NOT_FOUND", "video not found"))
	default:
		httperr.Render(c, httperr.Internal(err.Error()))
	}
}
