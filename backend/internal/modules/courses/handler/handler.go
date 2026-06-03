// Package handler contains the Gin HTTP handlers for the courses module.
// Handlers are intentionally thin: parse → call service → errors.Is → render.
// No domain logic lives here.
//
// CRITICAL: ErrNotOwner maps to DIFFERENT HTTP statuses depending on the route:
//   - GET  /api/courses/:id  → 404 via renderCourseErrorRead  (hides existence)
//   - PATCH /api/courses/:id → 403 via renderCourseErrorWrite (signals authz failure)
//
// This asymmetry is enforced by TWO separate render helpers (not a flag or switch)
// and is verified by TWO separate tests in handler_test.go. See REQ-DIVERGENCE.
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// Handler holds the service dependency injected at registration time.
type Handler struct {
	svc service.Service
}

// Register mounts the courses routes onto a pre-built Gin route group.
// The group must already carry JWT + RequireRole("creador") middleware.
func Register(creatorGrp *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}

	creatorGrp.POST("/courses", h.Create)
	creatorGrp.GET("/courses/:id", h.GetByID)
	creatorGrp.PATCH("/courses/:id", h.Update)
	creatorGrp.GET("/courses", h.List)
}

// Create godoc
// @Summary     Crea un curso (borrador)
// @Description Crea un nuevo curso con estado=borrador. El creadorID viene del JWT.
// @Tags        courses
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body   dto.CreateCourseRequest true "Datos del curso"
// @Success     201 {object} dto.CourseDetail
// @Failure     400 {object} httperr.Error "titulo faltante o invalido"
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error "rol creador requerido"
// @Failure     500 {object} httperr.Error
// @Router      /courses [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	creadorID := middleware.UserIDFrom(c)
	if creadorID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	model, err := h.svc.Create(c.Request.Context(), creadorID, service.CreateRequest{
		Titulo:      req.Titulo,
		Descripcion: req.Descripcion,
	})
	if err != nil {
		slog.Error("courses.create: unexpected error", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, dto.ToCourseDetail(model))
}

// GetByID godoc
// @Summary     Detalle de curso por ID (solo owner)
// @Description Retorna el detalle de un curso. Solo el creador propietario puede verlo.
//
//	ErrNotOwner → 404 (oculta existencia de borradores ajenos).
//
// @Tags        courses
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID del curso"
// @Success     200 {object} dto.CourseDetail
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error "rol creador requerido"
// @Failure     404 {object} httperr.Error "curso no encontrado o no pertenece al caller"
// @Failure     500 {object} httperr.Error
// @Router      /courses/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	model, err := h.svc.GetByID(c.Request.Context(), id, creadorID)
	if err != nil {
		h.renderCourseErrorRead(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToCourseDetail(model))
}

// Update godoc
// @Summary     Actualiza un curso (PATCH parcial, solo owner)
// @Description Actualiza titulo y/o descripcion de un curso. Solo borrador o rechazado.
//
//	ErrNotOwner → 403 (senyala fallo de autorizacion en escritura).
//
// @Tags        courses
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path   string                  true "UUID del curso"
// @Param       body body   dto.UpdateCourseRequest true "Campos a actualizar (parcial)"
// @Success     200 {object} dto.CourseDetail
// @Failure     400 {object} httperr.Error "body invalido o sin campos"
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error "no es propietario del curso"
// @Failure     404 {object} httperr.Error "curso no encontrado"
// @Failure     409 {object} httperr.Error "estado no permite edicion"
// @Failure     500 {object} httperr.Error
// @Router      /courses/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	// Reject requests with no actionable fields.
	if req.Titulo == nil && req.Descripcion == nil {
		httperr.Render(c, httperr.BadRequest("NO_FIELDS", "no fields to update"))
		return
	}

	creadorID := middleware.UserIDFrom(c)

	model, err := h.svc.UpdateByID(c.Request.Context(), id, creadorID, service.UpdateRequest{
		Titulo:      req.Titulo,
		Descripcion: req.Descripcion,
	})
	if err != nil {
		h.renderCourseErrorWrite(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToCourseDetail(model))
}

// List godoc
// @Summary     Lista mis cursos (paginado)
// @Description Retorna una pagina de cursos del caller. ?creator=me es obligatorio.
// @Tags        courses
// @Produce     json
// @Security    BearerAuth
// @Param       creator query string true  "Debe ser 'me' (solo cursos propios)"
// @Param       page    query int    false "Pagina (default 1)"
// @Param       size    query int    false "Tamano de pagina (max 100, default 20)"
// @Success     200 {object} object "pagina de cursos (items, page, size, total, totalPages)"
// @Failure     400 {object} httperr.Error "creator debe ser 'me'"
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error "rol creador requerido"
// @Failure     500 {object} httperr.Error
// @Router      /courses [get]
func (h *Handler) List(c *gin.Context) {
	creator := c.Query("creator")
	if creator != "me" {
		httperr.Render(c, httperr.BadRequest("INVALID_PARAM", "creator must be 'me'"))
		return
	}

	creadorID := middleware.UserIDFrom(c)
	p := pagination.ParseParams(c)

	page, err := h.svc.ListByCreator(c.Request.Context(), creadorID, p)
	if err != nil {
		slog.Error("courses.list: unexpected error", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.ToCourseListPage(page))
}

// ── Error render helpers (TWO — deliberate, enforces REQ-DIVERGENCE) ───────────

// renderCourseErrorRead maps service sentinels to HTTP statuses for READ routes (GET).
// CRITICAL: ErrNotOwner → 404 here (hides existence of private drafts from other creadores).
func (h *Handler) renderCourseErrorRead(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCourseNotFound):
		httperr.Render(c, httperr.NotFound("COURSE_NOT_FOUND", "course not found"))
	case errors.Is(err, service.ErrNotOwner):
		httperr.Render(c, httperr.NotFound("COURSE_NOT_FOUND", "course not found")) // 404: hide existence
	case errors.Is(err, service.ErrInvalidTransition):
		httperr.Render(c, httperr.Conflict("INVALID_TRANSITION", "course estado does not permit this edit"))
	default:
		slog.Error("courses: unexpected error (read)", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
	}
}

// renderCourseErrorWrite maps service sentinels to HTTP statuses for WRITE routes (PATCH).
// CRITICAL: ErrNotOwner → 403 here (signals authz failure per AC3 / REQ-DIVERGENCE).
// The ONLY difference from renderCourseErrorRead is the ErrNotOwner case.
func (h *Handler) renderCourseErrorWrite(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCourseNotFound):
		httperr.Render(c, httperr.NotFound("COURSE_NOT_FOUND", "course not found"))
	case errors.Is(err, service.ErrNotOwner):
		httperr.Render(c, httperr.Forbidden("NOT_OWNER", "you do not own this course")) // 403: authz signal
	case errors.Is(err, service.ErrInvalidTransition):
		httperr.Render(c, httperr.Conflict("INVALID_TRANSITION", "course estado does not permit this edit"))
	default:
		slog.Error("courses: unexpected error (write)", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
	}
}
