// Package handler contains the Gin HTTP handlers for the users module.
// Handlers are intentionally thin: parse → call service → errors.Is → render.
// No domain logic lives here.
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// Handler holds the service dependency injected at registration time.
type Handler struct {
	svc service.Service
}

// Register mounts the users routes onto two pre-built Gin route groups.
//
//   - admin: already carries JWT + RequireRole("administrador")
//   - me:    already carries JWT only
func Register(admin, me *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}

	// Admin-only routes — user management
	admin.GET("/users", h.List)
	admin.GET("/users/:id", h.GetByID)
	admin.PATCH("/users/:id/roles", h.PatchRoles)
	admin.PATCH("/users/:id/active", h.SetActive)

	// Admin-only routes — supervision management (PR-B)
	admin.GET("/supervisions", h.ListSupervisions)
	admin.POST("/supervisions", h.CreateSupervision)
	admin.DELETE("/supervisions/:id", h.DeleteSupervision)

	// JWT-only route (any authenticated user)
	me.GET("/users/me", h.GetMe)
}

// List godoc
// @Summary     Lista usuarios (paginado y filtrado)
// @Description Retorna una pagina de usuarios. Solo administradores.
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Param       page   query int    false "Pagina (default 1)"
// @Param       size   query int    false "Tamano de pagina (max 100, default 20)"
// @Param       q      query string false "Busqueda por nombre o email (ILIKE)"
// @Param       role   query string false "Filtro por rol exacto (alumno|creador|supervisor|administrador)"
// @Param       active query bool   false "Filtro por estado activo"
// @Success     200 {object} object "pagina de usuarios (items, page, size, total, totalPages)"
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error
// @Failure     500 {object} httperr.Error
// @Router      /users [get]
func (h *Handler) List(c *gin.Context) {
	p := pagination.ParseParams(c)

	filters := service.ListFilters{
		Q:    c.Query("q"),
		Role: c.Query("role"),
	}

	if raw := c.Query("active"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			httperr.Render(c, httperr.BadRequest("INVALID_PARAM", "active must be true or false"))
			return
		}

		filters.Active = &v
	}

	page, err := h.svc.List(c.Request.Context(), filters, p)
	if err != nil {
		slog.Error("users.list: unexpected error", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.ToUserListPage(page))
}

// GetByID godoc
// @Summary     Detalle de usuario por ID
// @Description Retorna el detalle de un usuario. Solo administradores.
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Param       id   path string true "UUID del usuario"
// @Success     200 {object} dto.UserDetail
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Failure     500 {object} httperr.Error
// @Router      /users/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")

	detail, err := h.svc.GetDetail(c.Request.Context(), id)
	if err != nil {
		h.renderUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToUserDetail(detail))
}

// GetMe godoc
// @Summary     Perfil del usuario autenticado
// @Description Retorna el detalle del usuario que realiza la peticion. Requiere JWT.
// @Tags        users
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} dto.UserDetail
// @Failure     401 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Failure     500 {object} httperr.Error
// @Router      /users/me [get]
func (h *Handler) GetMe(c *gin.Context) {
	id := middleware.UserIDFrom(c)
	if id == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	detail, err := h.svc.GetDetail(c.Request.Context(), id)
	if err != nil {
		h.renderUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToUserDetail(detail))
}

// PatchRoles godoc
// @Summary     Asigna o revoca roles a un usuario
// @Description Aplica un delta de roles (add/remove). Solo administradores.
// @Tags        users
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path   string               true "UUID del usuario"
// @Param       body body   dto.RolesPatchRequest true "Delta de roles"
// @Success     200 {object} dto.UserDetail
// @Failure     400 {object} httperr.Error "rol invalido, conflicto add/remove, o body invalido"
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Failure     409 {object} httperr.Error "ultimo administrador activo"
// @Failure     500 {object} httperr.Error
// @Router      /users/{id}/roles [patch]
func (h *Handler) PatchRoles(c *gin.Context) {
	id := c.Param("id")

	var req dto.RolesPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	// Nil slices → empty slices (service handles them as no-ops).
	if req.Add == nil {
		req.Add = []string{}
	}

	if req.Remove == nil {
		req.Remove = []string{}
	}

	detail, err := h.svc.PatchRoles(c.Request.Context(), id, req.Add, req.Remove)
	if err != nil {
		h.renderUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToUserDetail(detail))
}

// SetActive godoc
// @Summary     Activa o desactiva un usuario (soft-delete)
// @Description Establece el flag activo del usuario. Solo administradores.
// @Tags        users
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path   string               true  "UUID del usuario"
// @Param       body body   dto.ActivePatchRequest true  "Estado activo"
// @Success     200 {object} dto.UserDetail
// @Failure     400 {object} httperr.Error "body invalido"
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Failure     409 {object} httperr.Error "ultimo administrador activo"
// @Failure     500 {object} httperr.Error
// @Router      /users/{id}/active [patch]
func (h *Handler) SetActive(c *gin.Context) {
	id := c.Param("id")

	var req dto.ActivePatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	detail, err := h.svc.SetActive(c.Request.Context(), id, *req.Active)
	if err != nil {
		h.renderUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToUserDetail(detail))
}

// ListSupervisions godoc
// @Summary     Lista relaciones de supervision
// @Description Retorna todas las relaciones supervisor-empleado. Solo administradores.
// @Tags        supervisions
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array}  dto.SupervisionItem
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error
// @Failure     500 {object} httperr.Error
// @Router      /supervisions [get]
func (h *Handler) ListSupervisions(c *gin.Context) {
	svs, err := h.svc.ListSupervisions(c.Request.Context())
	if err != nil {
		slog.Error("users.listSupervisions: unexpected error", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}

	items := make([]dto.SupervisionItem, 0, len(svs))
	for _, sv := range svs {
		items = append(items, dto.ToSupervisionItem(sv))
	}

	c.JSON(http.StatusOK, items)
}

// CreateSupervision godoc
// @Summary     Crea una relacion de supervision
// @Description Asigna un supervisor a un empleado. Solo administradores.
// @Tags        supervisions
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body   dto.SupervisionCreateRequest true "Relacion supervisor-empleado"
// @Success     201 {object} dto.SupervisionItem
// @Failure     400 {object} httperr.Error "auto-supervision o body invalido"
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error "usuario no encontrado"
// @Failure     409 {object} httperr.Error "empleado ya tiene supervisor"
// @Failure     500 {object} httperr.Error
// @Router      /supervisions [post]
func (h *Handler) CreateSupervision(c *gin.Context) {
	var req dto.SupervisionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	sv, err := h.svc.CreateSupervision(c.Request.Context(), req.SupervisorID, req.EmpleadoID)
	if err != nil {
		h.renderUserError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToSupervisionItem(*sv))
}

// DeleteSupervision godoc
// @Summary     Elimina una relacion de supervision
// @Description Elimina la relacion por su ID. Solo administradores.
// @Tags        supervisions
// @Produce     json
// @Security    BearerAuth
// @Param       id   path string true "UUID de la relacion de supervision"
// @Success     204 "Sin contenido"
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error "relacion no encontrada"
// @Failure     500 {object} httperr.Error
// @Router      /supervisions/{id} [delete]
func (h *Handler) DeleteSupervision(c *gin.Context) {
	id := c.Param("id")

	if err := h.svc.DeleteSupervision(c.Request.Context(), id); err != nil {
		h.renderUserError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ── error mapping ─────────────────────────────────────────────────────────────

// renderUserError maps service sentinels to the correct HTTP error via errors.Is.
func (h *Handler) renderUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		httperr.Render(c, httperr.NotFound("USER_NOT_FOUND", "user not found"))
	case errors.Is(err, service.ErrSupervisionNotFound):
		httperr.Render(c, httperr.NotFound("SUPERVISION_NOT_FOUND", "supervision relation not found"))
	case errors.Is(err, service.ErrLastAdmin):
		httperr.Render(c, httperr.Conflict("LAST_ADMIN", "cannot remove the last active administrator"))
	case errors.Is(err, service.ErrInvalidRole):
		httperr.Render(c, httperr.BadRequest("INVALID_ROLE", "unknown role name"))
	case errors.Is(err, service.ErrAddRemoveConflict):
		httperr.Render(c, httperr.BadRequest("ROLE_CONFLICT", "a role appears in both add and remove"))
	case errors.Is(err, service.ErrSelfSupervision):
		httperr.Render(c, httperr.BadRequest("SELF_SUPERVISION", "a user cannot supervise themselves"))
	case errors.Is(err, service.ErrSupervisionExists):
		httperr.Render(c, httperr.Conflict("SUPERVISION_EXISTS", "employee already has a supervisor"))
	default:
		slog.Error("users: unexpected error", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
	}
}
