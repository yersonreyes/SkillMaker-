// Package service contains the business logic for the reporting module.
// It is HTTP-agnostic and read-only. Handlers map its results to HTTP responses.
//
// Responsibilities (design §3):
//   - Call repository methods and aggregate results into response models.
//   - Fill missing estados (borrador/en_revision/aprobado/rechazado) with 0.
//   - Format MonthCountRow.Month (time.Time from date_trunc) → ISO "2006-01" string.
//   - Pass-through for CourseStats, TeamProgress, UserProgress.
package service

import (
	"context"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting/repository"
)

// ── Read models ───────────────────────────────────────────────────────────────

// GlobalReportModel is the aggregate read model for /reports/global.
type GlobalReportModel struct {
	ActiveUsers             int64
	CoursesByEstado         []EstadoItem
	TotalAttempts           int64
	CertificatesIssued      int64
	TopCreators             []CreatorItem
	UsersPerMonth           []MonthItem
	ApprovedCoursesPerMonth []MonthItem
}

// EstadoItem is a single estado + count pair (after service-level 0-fill).
type EstadoItem struct {
	Estado string
	Total  int64
}

// CreatorItem is a single creator + count pair.
type CreatorItem struct {
	Nombre string
	Total  int64
}

// MonthItem is a month label (ISO "2006-01") + count.
type MonthItem struct {
	Month string
	Total int64
}

// CourseStatModel is the read model for a single course in /reports/courses.
type CourseStatModel struct {
	ID           string
	Titulo       string
	Estado       string
	Enrollments  int64
	Attempts     int64
	ApprovalRate float64
}

// TeamMemberModel is the read model for a single employee in /reports/team.
// LastAttemptDate is nil when the employee has no finalized attempts (spec REQ-TEAM).
type TeamMemberModel struct {
	EmpleadoID      string
	EmpleadoNombre  string
	Enrolled        int64
	Completed       int64
	LastAttemptDate *string
}

// UserProgressModel is the read model for /reports/users/:id/progress.
type UserProgressModel struct {
	Enrolled       int64
	Completed      int64
	Attempts       int64
	PassedAttempts int64
	Certificates   int64
}

// ── Service interface ──────────────────────────────────────────────────────────

// Service is the public interface of the reporting domain.
type Service interface {
	// GlobalReport returns system-wide aggregate metrics.
	// Fills all 4 course estados with 0 when absent from repo.
	GlobalReport(ctx context.Context) (*GlobalReportModel, error)

	// CourseReport returns per-course aggregate stats for all courses.
	CourseReport(ctx context.Context) ([]CourseStatModel, error)

	// UserProgressReport returns the 5 progress counters for userID.
	UserProgressReport(ctx context.Context, userID string) (UserProgressModel, error)

	// TeamReport returns the supervised employees for supervisorID.
	TeamReport(ctx context.Context, supervisorID string) ([]TeamMemberModel, error)
}

// ── reportingService implementation ──────────────────────────────────────────

type reportingService struct {
	repo repository.Repository
}

// New constructs a Service backed by the given Repository.
func New(repo repository.Repository) Service {
	return &reportingService{repo: repo}
}

// allEstados is the canonical set of course estados.
// The service fills any missing ones with 0 (design §3).
var allEstados = []string{"borrador", "en_revision", "aprobado", "rechazado"}

// GlobalReport aggregates all system-wide metrics.
// It calls 7 repo methods and performs two shaping operations:
//  1. Estado-fill: any absent estado gets Total=0.
//  2. Month formatting: time.Time → "2006-01" ISO string.
func (s *reportingService) GlobalReport(ctx context.Context) (*GlobalReportModel, error) {
	activeUsers, err := s.repo.ActiveUsers(ctx)
	if err != nil {
		return nil, err
	}

	estadoRows, err := s.repo.CoursesByEstado(ctx)
	if err != nil {
		return nil, err
	}

	totalAttempts, err := s.repo.TotalAttempts(ctx)
	if err != nil {
		return nil, err
	}

	certsIssued, err := s.repo.CertificatesIssued(ctx)
	if err != nil {
		return nil, err
	}

	topCreatorRows, err := s.repo.TopCreators(ctx, 10)
	if err != nil {
		return nil, err
	}

	usersPerMonthRows, err := s.repo.UsersPerMonth(ctx)
	if err != nil {
		return nil, err
	}

	approvedPerMonthRows, err := s.repo.ApprovedCoursesPerMonth(ctx)
	if err != nil {
		return nil, err
	}

	// ── Shape: fill all 4 estados (including absent ones with Total=0) ──────
	estadoMap := make(map[string]int64, len(estadoRows))
	for _, r := range estadoRows {
		estadoMap[r.Estado] = r.Total
	}
	coursesByEstado := make([]EstadoItem, 0, len(allEstados))
	for _, e := range allEstados {
		coursesByEstado = append(coursesByEstado, EstadoItem{Estado: e, Total: estadoMap[e]})
	}

	// ── Shape: creators ──────────────────────────────────────────────────────
	topCreators := make([]CreatorItem, 0, len(topCreatorRows))
	for _, r := range topCreatorRows {
		topCreators = append(topCreators, CreatorItem{Nombre: r.Nombre, Total: r.Total})
	}

	// ── Shape: month formatting ──────────────────────────────────────────────
	usersPerMonth := make([]MonthItem, 0, len(usersPerMonthRows))
	for _, r := range usersPerMonthRows {
		usersPerMonth = append(usersPerMonth, MonthItem{
			Month: r.Month.Format("2006-01"),
			Total: r.Total,
		})
	}

	approvedPerMonth := make([]MonthItem, 0, len(approvedPerMonthRows))
	for _, r := range approvedPerMonthRows {
		approvedPerMonth = append(approvedPerMonth, MonthItem{
			Month: r.Month.Format("2006-01"),
			Total: r.Total,
		})
	}

	return &GlobalReportModel{
		ActiveUsers:             activeUsers,
		CoursesByEstado:         coursesByEstado,
		TotalAttempts:           totalAttempts,
		CertificatesIssued:      certsIssued,
		TopCreators:             topCreators,
		UsersPerMonth:           usersPerMonth,
		ApprovedCoursesPerMonth: approvedPerMonth,
	}, nil
}

// CourseReport returns per-course aggregate stats. Thin pass-through with model mapping.
func (s *reportingService) CourseReport(ctx context.Context) ([]CourseStatModel, error) {
	rows, err := s.repo.CourseStats(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CourseStatModel, 0, len(rows))
	for _, r := range rows {
		out = append(out, CourseStatModel{
			ID:           r.ID,
			Titulo:       r.Titulo,
			Estado:       r.Estado,
			Enrollments:  r.Enrollments,
			Attempts:     r.Attempts,
			ApprovalRate: r.ApprovalRate,
		})
	}
	return out, nil
}

// UserProgressReport returns the 5 progress counters for userID.
// Non-existent userID returns all-zero UserProgressModel (no error).
func (s *reportingService) UserProgressReport(ctx context.Context, userID string) (UserProgressModel, error) {
	row, err := s.repo.UserProgress(ctx, userID)
	if err != nil {
		return UserProgressModel{}, err
	}
	return UserProgressModel{
		Enrolled:       row.Enrolled,
		Completed:      row.Completed,
		Attempts:       row.Attempts,
		PassedAttempts: row.PassedAttempts,
		Certificates:   row.Certificates,
	}, nil
}

// TeamReport returns the supervised employees for supervisorID.
func (s *reportingService) TeamReport(ctx context.Context, supervisorID string) ([]TeamMemberModel, error) {
	rows, err := s.repo.TeamProgress(ctx, supervisorID)
	if err != nil {
		return nil, err
	}
	out := make([]TeamMemberModel, 0, len(rows))
	for _, r := range rows {
		out = append(out, TeamMemberModel{
			EmpleadoID:      r.EmpleadoID,
			EmpleadoNombre:  r.EmpleadoNombre,
			Enrolled:        r.Enrolled,
			Completed:       r.Completed,
			LastAttemptDate: r.LastAttemptDate,
		})
	}
	return out, nil
}
