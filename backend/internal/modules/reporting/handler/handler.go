// Package handler contains the Gin HTTP handlers for the reporting module.
// Handlers are intentionally thin: parse → authz check → call service → render.
// No domain logic lives here; business logic is in service.go.
//
// Authorization model (REQ-SEC):
//   - /reports/global, /reports/courses → adminGrp (JWT + RequireRole("administrador"))
//   - /reports/team → supervisorGrp (JWT + RequireRole("supervisor"))
//   - /reports/users/:id/progress → protected (JWT only) + IN-HANDLER admin-or-self authz
//
// In-handler admin-or-self pattern (from design §4, same as approvals/handler ListHistory):
//
//	callerID := middleware.UserIDFrom(c)
//	roles    := middleware.RolesFrom(c)
//	isAdmin  := slices.Contains(roles, "administrador")
//	if !isAdmin && callerID != targetID { httperr.Render(c, httperr.Forbidden(...)) }
package handler

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
)

// Handler holds the service dependency injected at registration time.
type Handler struct {
	svc service.Service
}

// RegisterAdminRoutes mounts the admin-gated reporting routes.
// adminGrp must carry JWT + RequireRole("administrador") middleware.
func RegisterAdminRoutes(adminGrp *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}
	adminGrp.GET("/reports/global", h.Global)
	adminGrp.GET("/reports/courses", h.Courses)
}

// RegisterSupervisorRoutes mounts the supervisor-gated reporting routes.
// supervisorGrp must carry JWT + RequireRole("supervisor") middleware.
func RegisterSupervisorRoutes(supervisorGrp *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}
	supervisorGrp.GET("/reports/team", h.Team)
}

// RegisterSelfRoutes mounts the JWT-only route with in-handler authz.
// protected must carry JWT middleware only (no RequireRole).
func RegisterSelfRoutes(protected *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}
	protected.GET("/reports/users/:id/progress", h.UserProgress)
}

// ── Global ─────────────────────────────────────────────────────────────────────

// Global godoc
// @Summary     Reporte global del sistema
// @Description Retorna métricas agregadas del sistema. Solo administradores.
// @Tags        reporting
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} dto.GlobalReportResponse
// @Failure     401 {object} httperr.Error "sin autenticación"
// @Failure     403 {object} httperr.Error "no es administrador"
// @Router      /reports/global [get]
func (h *Handler) Global(c *gin.Context) {
	result, err := h.svc.GlobalReport(c.Request.Context())
	if err != nil {
		slog.Error("reporting: GlobalReport", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}

	// Map service model → DTO.
	estadoItems := make([]dto.CoursesByEstadoItem, 0, len(result.CoursesByEstado))
	for _, e := range result.CoursesByEstado {
		estadoItems = append(estadoItems, dto.CoursesByEstadoItem{Estado: e.Estado, Total: e.Total})
	}

	creatorItems := make([]dto.TopCreatorItem, 0, len(result.TopCreators))
	for _, c := range result.TopCreators {
		creatorItems = append(creatorItems, dto.TopCreatorItem{Nombre: c.Nombre, Total: c.Total})
	}

	usersPerMonth := make([]dto.MonthCountItem, 0, len(result.UsersPerMonth))
	for _, m := range result.UsersPerMonth {
		usersPerMonth = append(usersPerMonth, dto.MonthCountItem{Month: m.Month, Total: m.Total})
	}

	approvedPerMonth := make([]dto.MonthCountItem, 0, len(result.ApprovedCoursesPerMonth))
	for _, m := range result.ApprovedCoursesPerMonth {
		approvedPerMonth = append(approvedPerMonth, dto.MonthCountItem{Month: m.Month, Total: m.Total})
	}

	c.JSON(http.StatusOK, dto.GlobalReportResponse{
		ActiveUsers:             result.ActiveUsers,
		CoursesByEstado:         estadoItems,
		TotalAttempts:           result.TotalAttempts,
		CertificatesIssued:      result.CertificatesIssued,
		TopCreators:             creatorItems,
		UsersPerMonth:           usersPerMonth,
		ApprovedCoursesPerMonth: approvedPerMonth,
	})
}

// ── Courses ─────────────────────────────────────────────────────────────────────

// Courses godoc
// @Summary     Reporte de cursos
// @Description Retorna estadísticas por curso para todos los cursos. Solo administradores.
// @Tags        reporting
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} dto.CourseReportItem
// @Failure     401 {object} httperr.Error "sin autenticación"
// @Failure     403 {object} httperr.Error "no es administrador"
// @Router      /reports/courses [get]
func (h *Handler) Courses(c *gin.Context) {
	items, err := h.svc.CourseReport(c.Request.Context())
	if err != nil {
		slog.Error("reporting: CourseReport", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}

	out := make([]dto.CourseReportItem, 0, len(items))
	for _, item := range items {
		out = append(out, dto.CourseReportItem{
			ID:           item.ID,
			Titulo:       item.Titulo,
			Estado:       item.Estado,
			Enrollments:  item.Enrollments,
			Attempts:     item.Attempts,
			ApprovalRate: item.ApprovalRate,
		})
	}
	c.JSON(http.StatusOK, out)
}

// ── Team ───────────────────────────────────────────────────────────────────────

// Team godoc
// @Summary     Avance del equipo supervisado
// @Description Retorna los empleados supervisados por el caller con sus métricas de progreso. Solo supervisors.
// @Tags        reporting
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} dto.TeamReportItem
// @Failure     401 {object} httperr.Error "sin autenticación"
// @Failure     403 {object} httperr.Error "no es supervisor"
// @Router      /reports/team [get]
func (h *Handler) Team(c *gin.Context) {
	callerID := middleware.UserIDFrom(c)

	items, err := h.svc.TeamReport(c.Request.Context(), callerID)
	if err != nil {
		slog.Error("reporting: TeamReport", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}

	out := make([]dto.TeamReportItem, 0, len(items))
	for _, item := range items {
		out = append(out, dto.TeamReportItem{
			EmpleadoID:      item.EmpleadoID,
			EmpleadoNombre:  item.EmpleadoNombre,
			Enrolled:        item.Enrolled,
			Completed:       item.Completed,
			LastAttemptDate: item.LastAttemptDate,
		})
	}
	c.JSON(http.StatusOK, out)
}

// ── UserProgress ──────────────────────────────────────────────────────────────

// UserProgress godoc
// @Summary     Progreso de un usuario
// @Description Retorna las métricas de progreso de un usuario. Accesible para el propio usuario o un administrador.
// @Tags        reporting
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID del usuario"
// @Success     200 {object} dto.UserProgressResponse
// @Failure     401 {object} httperr.Error "sin autenticación"
// @Failure     403 {object} httperr.Error "no es el usuario ni administrador"
// @Router      /reports/users/{id}/progress [get]
func (h *Handler) UserProgress(c *gin.Context) {
	targetID := c.Param("id")
	callerID := middleware.UserIDFrom(c)
	roles := middleware.RolesFrom(c)
	isAdmin := slices.Contains(roles, "administrador")

	// In-handler admin-or-self authz (design §4).
	// A supervisor calling for an employee → 403 (they must use /reports/team).
	if !isAdmin && callerID != targetID {
		httperr.Render(c, httperr.Forbidden("FORBIDDEN", "you can only view your own progress"))
		return
	}

	row, err := h.svc.UserProgressReport(c.Request.Context(), targetID)
	if err != nil {
		slog.Error("reporting: UserProgressReport", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.UserProgressResponse{
		Enrolled:       row.Enrolled,
		Completed:      row.Completed,
		Attempts:       row.Attempts,
		PassedAttempts: row.PassedAttempts,
		Certificates:   row.Certificates,
	})
}
