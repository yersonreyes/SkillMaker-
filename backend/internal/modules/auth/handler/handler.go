// Package handler contains the Gin HTTP handlers for the auth module.
// Handlers are intentionally thin: parse request → call service → render response.
// No domain logic lives here.
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

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

	res, err := h.svc.LoginWithGoogle(c.Request.Context(), req.IDToken)
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

	res, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
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
