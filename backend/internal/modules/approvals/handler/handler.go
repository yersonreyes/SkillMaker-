// Package handler contains the Gin HTTP handlers for the approvals module.
// Handlers are intentionally thin: parse → call service → errors.Is → render.
// No domain logic lives here.
//
// CRITICAL: ErrNotOwner maps to DIFFERENT HTTP statuses depending on the route:
//   - GET routes (ListHistory) → 404 via renderReadError (hides existence)
//   - POST routes (Submit) → 403 via renderWriteError (signals authz failure)
//
// CRITICAL Gin param convention (ADR-7, design §5):
//   - POST tree under /courses uses :courseId
//   - GET tree under /courses uses :id
//     Mixing these in the same method tree panics at boot.
//
// adminID is read from JWT claims (middleware.UserIDFrom) — NEVER from request body (SEC-8).
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"slices"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
)

// Handler holds the service dependency injected at registration time.
type Handler struct {
	svc service.Service
}

// RegisterCreator mounts the creator-gated approvals routes.
// creatorGrp must carry JWT + RequireRole("creador") middleware.
// CRITICAL param: POST tree under /courses = :courseId
func RegisterCreator(creatorGrp *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}
	creatorGrp.POST("/courses/:courseId/submit", h.Submit)
}

// RegisterAdmin mounts the admin-gated approvals routes.
// adminGrp must carry JWT + RequireRole("administrador") middleware.
// CRITICAL param: POST tree under /courses = :courseId
func RegisterAdmin(adminGrp *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}
	adminGrp.GET("/approvals/pending", h.ListPending)
	adminGrp.POST("/courses/:courseId/approve", h.Approve)
	adminGrp.POST("/courses/:courseId/reject", h.Reject)
}

// RegisterHistory mounts the JWT-only history route (owner-or-admin, in-handler authz).
// protected must carry JWT middleware only (no RequireRole).
// CRITICAL param: GET tree under /courses = :id
func RegisterHistory(protected *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}
	protected.GET("/courses/:id/approvals", h.ListHistory)
}

// ── Submit ─────────────────────────────────────────────────────────────────────

// Submit godoc
// @Summary     Enviar curso a revision
// @Description Transiciona el curso de borrador/rechazado a en_revision. Valida propiedad, estado, contenido y evaluacion.
// @Tags        approvals
// @Produce     json
// @Security    BearerAuth
// @Param       courseId path string true "UUID del curso"
// @Success     200 {object} dto.SubmitReviewResponse
// @Failure     400 {object} httperr.Error "request invalido"
// @Failure     403 {object} httperr.Error "no es propietario del curso"
// @Failure     404 {object} httperr.Error "curso no encontrado"
// @Failure     409 {object} httperr.Error "estado invalido / sin contenido / evaluacion incompleta"
// @Router      /courses/{courseId}/submit [post]
func (h *Handler) Submit(c *gin.Context) {
	courseID := c.Param("courseId")
	callerID := middleware.UserIDFrom(c)

	if err := h.svc.SubmitToReview(c.Request.Context(), courseID, callerID); err != nil {
		h.renderWriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SubmitReviewResponse{
		CourseID: courseID,
		Estado:   "en_revision",
	})
}

// ── ListPending ────────────────────────────────────────────────────────────────

// ListPending godoc
// @Summary     Lista cursos en revision
// @Description Retorna todos los cursos con estado=en_revision. Solo administradores.
// @Tags        approvals
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array}  dto.PendingItemDTO
// @Failure     403 {object} httperr.Error "no es administrador"
// @Router      /approvals/pending [get]
func (h *Handler) ListPending(c *gin.Context) {
	rows, err := h.svc.ListPending(c.Request.Context())
	if err != nil {
		slog.Error("approvals: unexpected error (ListPending)", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.ToPending(rows))
}

// ── Approve ────────────────────────────────────────────────────────────────────

// Approve godoc
// @Summary     Aprobar un curso
// @Description Aprueba un curso en revision. Crea fila de auditoria y actualiza estado a aprobado.
// @Tags        approvals
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       courseId path   string              true  "UUID del curso"
// @Param       body     body   dto.ApproveRequest  false "Comentario opcional"
// @Success     200
// @Failure     403 {object} httperr.Error "no es administrador"
// @Failure     404 {object} httperr.Error "curso no encontrado"
// @Failure     409 {object} httperr.Error "curso no en revision"
// @Router      /courses/{courseId}/approve [post]
func (h *Handler) Approve(c *gin.Context) {
	courseID := c.Param("courseId")
	adminID := middleware.UserIDFrom(c) // SEC-8: from JWT, never from body

	var req dto.ApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Binding is optional for approve (comentario is optional).
		req = dto.ApproveRequest{}
	}

	if err := h.svc.Approve(c.Request.Context(), courseID, adminID, req.Comentario); err != nil {
		h.renderWriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

// ── Reject ─────────────────────────────────────────────────────────────────────

// Reject godoc
// @Summary     Rechazar un curso
// @Description Rechaza un curso en revision con un comentario obligatorio.
// @Tags        approvals
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       courseId path   string             true "UUID del curso"
// @Param       body     body   dto.RejectRequest  true "Comentario requerido"
// @Success     200
// @Failure     400 {object} httperr.Error "comentario requerido"
// @Failure     403 {object} httperr.Error "no es administrador"
// @Failure     404 {object} httperr.Error "curso no encontrado"
// @Failure     409 {object} httperr.Error "curso no en revision"
// @Router      /courses/{courseId}/reject [post]
func (h *Handler) Reject(c *gin.Context) {
	courseID := c.Param("courseId")
	adminID := middleware.UserIDFrom(c) // SEC-8: from JWT, never from body

	var req dto.RejectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Body missing or unparseable → pass empty comentario to service → ErrCommentRequired.
		req = dto.RejectRequest{}
	}

	if err := h.svc.Reject(c.Request.Context(), courseID, adminID, req.Comentario); err != nil {
		h.renderWriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

// ── ListHistory ────────────────────────────────────────────────────────────────

// ListHistory godoc
// @Summary     Historial de revisiones de un curso
// @Description Retorna las filas de aprobacion/rechazo de un curso. Accessible para el propietario o administrador.
// @Tags        approvals
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID del curso"
// @Success     200 {array}  dto.ApprovalHistoryDTO
// @Failure     403 {object} httperr.Error "no es propietario ni administrador"
// @Failure     404 {object} httperr.Error "curso no encontrado"
// @Router      /courses/{id}/approvals [get]
func (h *Handler) ListHistory(c *gin.Context) {
	courseID := c.Param("id")
	callerID := middleware.UserIDFrom(c)
	roles := middleware.RolesFrom(c)
	isAdmin := slices.Contains(roles, "administrador")

	rows, err := h.svc.ListHistory(c.Request.Context(), courseID, callerID, isAdmin)
	if err != nil {
		h.renderReadError(c, err) // ErrNotOwner → 404 (hide existence on read)
		return
	}
	c.JSON(http.StatusOK, dto.ToHistory(rows))
}

// ── Error render helpers ───────────────────────────────────────────────────────

// renderReadError maps service sentinels to HTTP statuses for READ routes (GET).
// ErrNotOwner → 404 (hides existence of the resource from non-owners).
func (h *Handler) renderReadError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCourseNotFound):
		httperr.Render(c, httperr.NotFound("COURSE_NOT_FOUND", "course not found"))
	case errors.Is(err, service.ErrNotOwner):
		httperr.Render(c, httperr.NotFound("COURSE_NOT_FOUND", "course not found")) // 404: hide existence
	case errors.Is(err, evaluations.ErrEvaluationNotFound):
		httperr.Render(c, httperr.Conflict("EVALUATION_NOT_FOUND", "evaluation not found"))
	case errors.Is(err, evaluations.ErrNoCorrectOption):
		httperr.Render(c, httperr.Conflict("EVALUATION_INCOMPLETE", "evaluation is incomplete"))
	default:
		slog.Error("approvals: unexpected error (read)", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
	}
}

// renderWriteError maps service sentinels to HTTP statuses for WRITE routes (POST).
// ErrNotOwner → 403 (signals authz failure for write operations).
func (h *Handler) renderWriteError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCourseNotFound):
		httperr.Render(c, httperr.NotFound("COURSE_NOT_FOUND", "course not found"))
	case errors.Is(err, service.ErrNotOwner):
		httperr.Render(c, httperr.Forbidden("NOT_OWNER", "you do not own this course")) // 403
	case errors.Is(err, service.ErrCourseNotSubmittable):
		httperr.Render(c, httperr.Conflict("COURSE_NOT_SUBMITTABLE", "course estado does not permit submission"))
	case errors.Is(err, service.ErrNoContent):
		httperr.Render(c, httperr.Conflict("NO_CONTENT", "course has no video content"))
	case errors.Is(err, service.ErrNotInReview):
		httperr.Render(c, httperr.Conflict("NOT_IN_REVIEW", "course is not in review"))
	case errors.Is(err, service.ErrCommentRequired):
		httperr.Render(c, httperr.BadRequest("COMMENT_REQUIRED", "rejection comment is required"))
	case errors.Is(err, evaluations.ErrEvaluationNotFound):
		httperr.Render(c, httperr.Conflict("EVALUATION_NOT_FOUND", "evaluation not found for course"))
	case errors.Is(err, evaluations.ErrNoCorrectOption):
		httperr.Render(c, httperr.Conflict("EVALUATION_INCOMPLETE", "evaluation is incomplete"))
	default:
		slog.Error("approvals: unexpected error (write)", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
	}
}
