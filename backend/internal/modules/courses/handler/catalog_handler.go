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

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

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
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// ListCatalog godoc
// @Summary     Lista cursos aprobados (catálogo público para alumnos)
// @Description Retorna una página de cursos con estado='aprobado'. Soporta ?page, ?size, ?q (ILIKE titulo).
// @Tags        catalog
// @Produce     json
// @Security    BearerAuth
// @Param       page query int    false "Página (default 1)"
// @Param       size query int    false "Tamaño de página (max 100, default 20)"
// @Param       q    query string false "Filtro ILIKE en titulo"
// @Success     200 {object} object "Página de CatalogCourseCard (items, page, size, total, totalPages)"
// @Failure     401 {object} httperr.Error
// @Failure     500 {object} httperr.Error
// @Router      /catalog [get]
func (h *CatalogHandler) ListCatalog(c *gin.Context) {
	p := pagination.ParseParams(c)
	q := c.Query("q")

	page, err := h.svc.ListCatalog(c.Request.Context(), p, q)
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
