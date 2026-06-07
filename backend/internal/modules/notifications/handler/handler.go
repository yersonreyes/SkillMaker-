// Package handler contains the Gin HTTP handlers for the notifications module.
// Handlers are intentionally thin: parse → call service → errors.Is → render.
// No domain logic lives here.
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// Handler holds the service dependency injected at registration time.
type Handler struct {
	svc service.Service
}

// Register mounts the notifications routes onto a JWT-protected route group.
// No RequireRole restriction: all authenticated users can access these routes.
//
// Routes registered:
//
//	GET  /notifications/me              → ListMine
//	GET  /notifications/me/unread-count → UnreadCount
//	PATCH /notifications/:id/read        → MarkRead
//	PATCH /notifications/me/read-all     → MarkAllRead
//
// Gin PATCH sibling safety: Gin v1.7+ supports a static literal ("me") and a
// param (":id") as siblings in the same method tree, with the static taking
// priority. The routes_boot_test.go verifies no panic occurs at registration.
func Register(protected *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}

	protected.GET("/notifications/me", h.ListMine)
	protected.GET("/notifications/me/unread-count", h.UnreadCount)
	protected.PATCH("/notifications/:id/read", h.MarkRead)
	protected.PATCH("/notifications/me/read-all", h.MarkAllRead)
}

// ── ListMine ──────────────────────────────────────────────────────────────────

// ListMine godoc
// @Summary     Lista mis notificaciones
// @Description Devuelve las notificaciones del usuario autenticado, paginadas por creado_en DESC.
// @Tags        notifications
// @Produce     json
// @Security    BearerAuth
// @Param       page query int false "Página (default 1)"
// @Param       size query int false "Tamaño (default 20)"
// @Success     200 {object} dto.NotificationListResponse
// @Failure     401 {object} httperr.Error
// @Router      /notifications/me [get]
func (h *Handler) ListMine(c *gin.Context) {
	userID := middleware.UserIDFrom(c)
	if userID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	p := pagination.ParseParams(c)
	page, err := h.svc.ListMine(c.Request.Context(), userID, p)
	if err != nil {
		httperr.Render(c, httperr.Internal("unexpected error listing notifications"))
		return
	}

	resp := dto.NotificationListResponse{
		Items:      toResponseItems(page.Items),
		Page:       page.Page,
		Size:       page.Size,
		Total:      page.Total,
		TotalPages: page.TotalPages,
	}
	c.JSON(http.StatusOK, resp)
}

// ── UnreadCount ───────────────────────────────────────────────────────────────

// UnreadCount godoc
// @Summary     Conteo de notificaciones sin leer
// @Description Devuelve el número de notificaciones no leídas del usuario autenticado.
// @Tags        notifications
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} dto.UnreadCountResponse
// @Failure     401 {object} httperr.Error
// @Router      /notifications/me/unread-count [get]
func (h *Handler) UnreadCount(c *gin.Context) {
	userID := middleware.UserIDFrom(c)
	if userID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	count, err := h.svc.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		httperr.Render(c, httperr.Internal("unexpected error getting unread count"))
		return
	}

	c.JSON(http.StatusOK, dto.UnreadCountResponse{Unread: count})
}

// ── MarkRead ──────────────────────────────────────────────────────────────────

// MarkRead godoc
// @Summary     Marcar notificación como leída
// @Description Marca una notificación específica como leída. Solo el propietario puede hacerlo.
// @Tags        notifications
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Notification UUID"
// @Success     200 {object} map[string]bool
// @Failure     401 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Router      /notifications/{id}/read [patch]
func (h *Handler) MarkRead(c *gin.Context) {
	userID := middleware.UserIDFrom(c)
	if userID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	id := c.Param("id")
	// uuid.Parse → 404 (NOT 500, NOT 400). This is a security invariant (REQ-SEC).
	if _, err := uuid.Parse(id); err != nil {
		httperr.Render(c, httperr.NotFound("NOTIF_NOT_FOUND", "notification not found"))
		return
	}

	err := h.svc.MarkRead(c.Request.Context(), id, userID)
	if errors.Is(err, service.ErrNotFound) {
		httperr.Render(c, httperr.NotFound("NOTIF_NOT_FOUND", "notification not found"))
		return
	}
	if err != nil {
		httperr.Render(c, httperr.Internal("unexpected error marking notification as read"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── MarkAllRead ───────────────────────────────────────────────────────────────

// MarkAllRead godoc
// @Summary     Marcar todas las notificaciones como leídas
// @Description Marca todas las notificaciones del usuario autenticado como leídas.
// @Tags        notifications
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} map[string]bool
// @Failure     401 {object} httperr.Error
// @Router      /notifications/me/read-all [patch]
func (h *Handler) MarkAllRead(c *gin.Context) {
	userID := middleware.UserIDFrom(c)
	if userID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	if err := h.svc.MarkAllRead(c.Request.Context(), userID); err != nil {
		httperr.Render(c, httperr.Internal("unexpected error marking all notifications as read"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── helpers ────────────────────────────────────────────────────────────────────

func toResponseItems(models []service.NotificationModel) []dto.NotificationResponse {
	items := make([]dto.NotificationResponse, 0, len(models))
	for _, m := range models {
		items = append(items, dto.NotificationResponse{
			ID:       m.ID,
			Tipo:     m.Tipo,
			Titulo:   m.Titulo,
			Cuerpo:   m.Cuerpo,
			Leida:    m.Leida,
			RefID:    m.RefID,
			CreadoEn: m.CreadoEn,
		})
	}
	return items
}
