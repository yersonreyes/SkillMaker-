// Package handler contains the Gin HTTP handlers for the auth module.
// Handlers are intentionally thin: parse request → call service → render response.
// No domain logic lives here.
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
)

// Handler holds the service dependency injected at registration time.
type Handler struct {
	svc service.Service
}

// Register mounts the auth routes under the provided router group.
// Expected usage: Register(apiGroup, svc) → POST /api/auth/google,refresh,logout
func Register(rg *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}
	g := rg.Group("/auth")
	g.POST("/google", h.LoginWithGoogle)
	g.POST("/refresh", h.Refresh)
	g.POST("/logout", h.Logout)
}

// LoginWithGoogle godoc
// @Summary     Login con Google Workspace
// @Description Valida el ID token de Google, verifica que el campo hd coincida con el
// @Description dominio corporativo configurado (GOOGLE_HOSTED_DOMAIN), crea o actualiza
// @Description el usuario por google_sub y retorna un JWT de acceso + refresh token opaco.
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body dto.GoogleLoginRequest true "Google ID token obtenido del cliente"
// @Success     200 {object} dto.LoginResponse
// @Failure     400 {object} httperr.Error "body invalido"
// @Failure     401 {object} httperr.Error "token invalido o dominio no autorizado"
// @Failure     500 {object} httperr.Error "error interno"
// @Router      /auth/google [post]
func (h *Handler) LoginWithGoogle(c *gin.Context) {
	var req dto.GoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	res, err := h.svc.LoginWithGoogle(c.Request.Context(), req.IDToken, ip, ua)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidGoogleToken):
			slog.Warn("auth.google: invalid token", "err", err)
			httperr.Render(c, &httperr.Error{Status: http.StatusUnauthorized, Code: "INVALID_GOOGLE_TOKEN", Message: err.Error()})
		case errors.Is(err, service.ErrUnauthorizedDomain):
			slog.Warn("auth.google: unauthorized domain", "err", err)
			httperr.Render(c, &httperr.Error{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED_DOMAIN", Message: err.Error()})
		default:
			// Loggea la causa real antes de devolver 500 generico al cliente.
			slog.Error("auth.google: unexpected error", "err", err)
			httperr.Render(c, httperr.Internal(err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, res)
}

// Refresh godoc
// @Summary     Renueva el access token rotando el refresh token
// @Description Valida el refresh token presentado, detecta replay attacks (OWASP),
// @Description revoca el token anterior y emite un par nuevo (access + refresh).
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body dto.RefreshRequest true "Refresh token actual"
// @Success     200 {object} dto.LoginResponse
// @Failure     400 {object} httperr.Error "body invalido"
// @Failure     401 {object} httperr.Error "token invalido, expirado o reutilizado"
// @Failure     500 {object} httperr.Error "error interno"
// @Router      /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	res, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken, ip, ua)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRefreshToken),
			errors.Is(err, service.ErrRefreshTokenReused):
			httperr.Render(c, &httperr.Error{Status: http.StatusUnauthorized, Code: "INVALID_REFRESH", Message: err.Error()})
		default:
			httperr.Render(c, httperr.Internal(err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, res)
}

// Logout godoc
// @Summary     Revoca el refresh token de la sesion actual (idempotente)
// @Description Revoca el refresh token si existe. Retorna 204 en todos los casos
// @Description (token inexistente, ya revocado, o revocado exitosamente).
// @Description No requiere JWT en el header.
// @Tags        auth
// @Accept      json
// @Param       body body dto.RefreshRequest false "Refresh token a revocar"
// @Success     204
// @Router      /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var req dto.RefreshRequest
	_ = c.ShouldBindJSON(&req) // logout is silent even if body is missing or malformed
	_ = h.svc.Logout(c.Request.Context(), req.RefreshToken)
	c.Status(http.StatusNoContent)
}

// RegisterSessionRoutes mounts the session-management routes on the JWT-protected
// group. The protected group MUST already carry the JWT middleware so that
// middleware.UserIDFrom(c) returns the authenticated caller's ID.
//
// Routes:
//
//	GET    /auth/sessions/me     → GetMySessions
//	DELETE /auth/sessions/:id    → RevokeSession
func RegisterSessionRoutes(protected *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}
	protected.GET("/auth/sessions/me", h.GetMySessions)
	protected.DELETE("/auth/sessions/:id", h.RevokeSession)
}

// GetMySessions godoc
// @Summary     Lista las sesiones activas del usuario autenticado
// @Description Retorna los refresh tokens activos (no revocados, no expirados) del caller,
// @Description ordenados por created_at DESC. ip y userAgent son informativos/forenses;
// @Description no se usan para controlar la autenticacion.
// @Tags        auth
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array}  dto.SessionResponse
// @Failure     401 {object} httperr.Error "sin autenticacion"
// @Failure     500 {object} httperr.Error "error interno"
// @Router      /auth/sessions/me [get]
func (h *Handler) GetMySessions(c *gin.Context) {
	userID := middleware.UserIDFrom(c)

	sessions, err := h.svc.ListActiveSessions(c.Request.Context(), userID)
	if err != nil {
		slog.Error("auth.sessions.me: unexpected error", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}
	c.JSON(http.StatusOK, sessions)
}

// RevokeSession godoc
// @Summary     Revoca una sesion activa por ID
// @Description Establece revoked_at en el refresh token indicado, siempre que pertenezca
// @Description al caller y no este ya revocado. Retorna 404 en cualquier caso que no sea
// @Description propiedad del caller (sin filtrar existencia para no dar info de otras sesiones).
// @Tags        auth
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID de la sesion (refresh_token.id)"
// @Success     204
// @Failure     401 {object} httperr.Error "sin autenticacion"
// @Failure     404 {object} httperr.Error "sesion no encontrada o no pertenece al caller"
// @Failure     500 {object} httperr.Error "error interno"
// @Router      /auth/sessions/{id} [delete]
func (h *Handler) RevokeSession(c *gin.Context) {
	userID := middleware.UserIDFrom(c)
	id := c.Param("id")

	err := h.svc.RevokeSession(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			httperr.Render(c, httperr.NotFound("SESSION_NOT_FOUND", "sesion no encontrada"))
			return
		}
		slog.Error("auth.sessions.revoke: unexpected error", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}
