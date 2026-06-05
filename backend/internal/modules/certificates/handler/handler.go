// Package handler contains the Gin HTTP handlers for the certificates module.
// Handlers are intentionally thin: parse → call service → errors.Is → render.
// No domain logic lives here.
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
)

// Handler holds the service dependency injected at registration time.
type Handler struct {
	svc service.Service
}

// Register mounts the certificates and badges routes onto a JWT-protected route group.
// No RequireRole restriction: all authenticated users can access these routes.
//
// Routes registered:
//
//	GET /certificates/me          → ListMine
//	GET /certificates/:id         → GetByID
//	GET /certificates/:id/download→ Download
//	GET /badges/me                → ListMyBadges
//	GET /badges/ranking           → Ranking
func Register(protected *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}

	protected.GET("/certificates/me", h.ListMine)
	protected.GET("/certificates/:id", h.GetByID)
	protected.GET("/certificates/:id/download", h.Download)
	protected.GET("/badges/me", h.ListMyBadges)
	protected.GET("/badges/ranking", h.Ranking)
}

// ── renderReadError ────────────────────────────────────────────────────────────

// renderReadError maps certificate read sentinels to HTTP responses.
// All not-found / owner violations return 404 to avoid leaking certificate existence.
func renderReadError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrCertificateNotFound) || errors.Is(err, service.ErrNotOwner) {
		httperr.Render(c, httperr.NotFound("CERT_NOT_FOUND", "certificate not found"))
		return
	}
	if errors.Is(err, service.ErrNoPDF) {
		httperr.Render(c, httperr.NotFound("CERT_NO_PDF", "certificate PDF not available"))
		return
	}
	httperr.Render(c, httperr.Internal("unexpected error"))
}

// ── ListMine ──────────────────────────────────────────────────────────────────

// ListMine godoc
// @Summary     Lista mis certificados
// @Description Devuelve los certificados del usuario autenticado, ordenados por emitidoEn DESC.
// @Tags        certificates
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} dto.ListCertificatesResponse
// @Failure     401 {object} httperr.Error
// @Router      /certificates/me [get]
func (h *Handler) ListMine(c *gin.Context) {
	userID := middleware.UserIDFrom(c)
	if userID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	certs, err := h.svc.ListMyCertificates(c.Request.Context(), userID)
	if err != nil {
		renderReadError(c, err)
		return
	}

	items := make([]dto.CertificateListItem, 0, len(certs))
	for _, cert := range certs {
		items = append(items, dto.CertificateListItem{
			ID:           cert.ID,
			CourseID:     cert.CourseID,
			CourseTitulo: cert.CourseTitulo,
			Codigo:       cert.Codigo,
			EmitidoEn:    cert.EmitidoEn,
		})
	}

	c.JSON(http.StatusOK, dto.ListCertificatesResponse{Certificates: items})
}

// ── GetByID ───────────────────────────────────────────────────────────────────

// GetByID godoc
// @Summary     Obtiene un certificado por ID
// @Description Devuelve el certificado si pertenece al usuario autenticado. 404 para no-propietarios (anti-enumeración).
// @Tags        certificates
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Certificate ID"
// @Success     200 {object} dto.CertificateResponse
// @Failure     401 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Router      /certificates/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	userID := middleware.UserIDFrom(c)
	if userID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	certID := c.Param("id")
	cert, err := h.svc.GetCertificate(c.Request.Context(), certID, userID)
	if err != nil {
		renderReadError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CertificateResponse{
		ID:           cert.ID,
		CourseID:     cert.CourseID,
		CourseTitulo: cert.CourseTitulo,
		Codigo:       cert.Codigo,
		EmitidoEn:    cert.EmitidoEn,
	})
}

// ── Download ──────────────────────────────────────────────────────────────────

// Download godoc
// @Summary     Obtiene URL de descarga del certificado
// @Description Devuelve una URL presignada para descargar el PDF del certificado. Requiere ser el propietario.
// @Tags        certificates
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Certificate ID"
// @Success     200 {object} dto.DownloadURLResponse
// @Failure     401 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Router      /certificates/{id}/download [get]
func (h *Handler) Download(c *gin.Context) {
	userID := middleware.UserIDFrom(c)
	if userID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	certID := c.Param("id")
	result, err := h.svc.GetDownloadURL(c.Request.Context(), certID, userID)
	if err != nil {
		renderReadError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.DownloadURLResponse{
		URL:       result.URL,
		ExpiresAt: result.ExpiresAt,
	})
}

// ── ListMyBadges ──────────────────────────────────────────────────────────────

// ListMyBadges godoc
// @Summary     Lista mis insignias
// @Description Devuelve las insignias ganadas por el usuario autenticado.
// @Tags        badges
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} dto.ListBadgesResponse
// @Failure     401 {object} httperr.Error
// @Router      /badges/me [get]
func (h *Handler) ListMyBadges(c *gin.Context) {
	userID := middleware.UserIDFrom(c)
	if userID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	badges, err := h.svc.ListMyBadges(c.Request.Context(), userID)
	if err != nil {
		httperr.Render(c, httperr.Internal("unexpected error"))
		return
	}

	items := make([]dto.BadgeResponse, 0, len(badges))
	for _, b := range badges {
		items = append(items, dto.BadgeResponse{
			ID:          b.ID,
			Nombre:      b.Nombre,
			Descripcion: b.Descripcion,
			OtorgadoEn:  b.OtorgadoEn,
		})
	}

	c.JSON(http.StatusOK, dto.ListBadgesResponse{Badges: items})
}

// ── Ranking ───────────────────────────────────────────────────────────────────

// Ranking godoc
// @Summary     Ranking de usuarios por certificados
// @Description Devuelve el top-10 de usuarios con más certificados. Excluye usuarios con 0 certificados.
// @Tags        badges
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} dto.RankingResponse
// @Failure     401 {object} httperr.Error
// @Router      /badges/ranking [get]
func (h *Handler) Ranking(c *gin.Context) {
	userID := middleware.UserIDFrom(c)
	if userID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	rows, err := h.svc.Ranking(c.Request.Context(), 10)
	if err != nil {
		httperr.Render(c, httperr.Internal("unexpected error"))
		return
	}

	items := make([]dto.RankingItem, 0, len(rows))
	for i, r := range rows {
		items = append(items, dto.RankingItem{
			Posicion:   i + 1,
			UserNombre: r.Nombre,
			CertCount:  r.Total,
		})
	}

	c.JSON(http.StatusOK, dto.RankingResponse{Ranking: items})
}
