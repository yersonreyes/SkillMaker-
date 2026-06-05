// Package service_test — unit tests for the reporting service.
// Uses a mock repository to verify:
//   - GlobalReport fills all 4 estados (borrador/en_revision/aprobado/rechazado) with 0 when absent
//   - Month formatting: time.Time → "2006-01" ISO string
//   - Thin pass-through for Courses, Team, UserProgress
//
// No build tag: runs with standard `make backend-test`.
package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting/service"
)

// ── Mock repository ────────────────────────────────────────────────────────────

type mockRepo struct {
	activeUsers           int64
	activeUsersErr        error
	coursesByEstado       []repository.EstadoCountRow
	coursesByEstadoErr    error
	totalAttempts         int64
	totalAttemptsErr      error
	certificatesIssued    int64
	certificatesIssuedErr error
	topCreators           []repository.CreatorCountRow
	topCreatorsErr        error
	usersPerMonth         []repository.MonthCountRow
	usersPerMonthErr      error
	approvedPerMonth      []repository.MonthCountRow
	approvedPerMonthErr   error
	courseStats           []repository.CourseStatRow
	courseStatsErr        error
	userProgress          repository.UserProgressRow
	userProgressErr       error
	teamProgress          []repository.TeamMemberRow
	teamProgressErr       error
}

func (m *mockRepo) ActiveUsers(_ context.Context) (int64, error) {
	return m.activeUsers, m.activeUsersErr
}
func (m *mockRepo) CoursesByEstado(_ context.Context) ([]repository.EstadoCountRow, error) {
	return m.coursesByEstado, m.coursesByEstadoErr
}
func (m *mockRepo) TotalAttempts(_ context.Context) (int64, error) {
	return m.totalAttempts, m.totalAttemptsErr
}
func (m *mockRepo) CertificatesIssued(_ context.Context) (int64, error) {
	return m.certificatesIssued, m.certificatesIssuedErr
}
func (m *mockRepo) TopCreators(_ context.Context, _ int) ([]repository.CreatorCountRow, error) {
	return m.topCreators, m.topCreatorsErr
}
func (m *mockRepo) UsersPerMonth(_ context.Context) ([]repository.MonthCountRow, error) {
	return m.usersPerMonth, m.usersPerMonthErr
}
func (m *mockRepo) ApprovedCoursesPerMonth(_ context.Context) ([]repository.MonthCountRow, error) {
	return m.approvedPerMonth, m.approvedPerMonthErr
}
func (m *mockRepo) CourseStats(_ context.Context) ([]repository.CourseStatRow, error) {
	return m.courseStats, m.courseStatsErr
}
func (m *mockRepo) UserProgress(_ context.Context, _ string) (repository.UserProgressRow, error) {
	return m.userProgress, m.userProgressErr
}
func (m *mockRepo) TeamProgress(_ context.Context, _ string) ([]repository.TeamMemberRow, error) {
	return m.teamProgress, m.teamProgressErr
}

// ── Tests ──────────────────────────────────────────────────────────────────────

// TestGlobalReport_EstadoFill verifies that when the repo returns only {aprobado:2},
// the service GlobalReport output contains all 4 estados, with the missing ones at Total=0.
func TestGlobalReport_EstadoFill(t *testing.T) {
	repo := &mockRepo{
		activeUsers:     10,
		coursesByEstado: []repository.EstadoCountRow{{Estado: "aprobado", Total: 2}},
	}
	svc := service.New(repo)

	result, err := svc.GlobalReport(context.Background())
	require.NoError(t, err)

	// Build a map to check all 4 estados.
	estadoMap := make(map[string]int64)
	for _, item := range result.CoursesByEstado {
		estadoMap[item.Estado] = item.Total
	}

	assert.Equal(t, int64(2), estadoMap["aprobado"], "aprobado must be 2")
	assert.Equal(t, int64(0), estadoMap["borrador"], "borrador must be 0 (filled by service)")
	assert.Equal(t, int64(0), estadoMap["en_revision"], "en_revision must be 0 (filled by service)")
	assert.Equal(t, int64(0), estadoMap["rechazado"], "rechazado must be 0 (filled by service)")
	assert.Len(t, result.CoursesByEstado, 4, "must have exactly 4 estado items")

	assert.Equal(t, int64(10), result.ActiveUsers)
}

// TestGlobalReport_MonthFormatting verifies that a time.Time from the repo
// is formatted to "2006-01" ISO string in the DTO.
func TestGlobalReport_MonthFormatting(t *testing.T) {
	marchTime := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	repo := &mockRepo{
		usersPerMonth:    []repository.MonthCountRow{{Month: marchTime, Total: 5}},
		approvedPerMonth: []repository.MonthCountRow{{Month: marchTime, Total: 3}},
	}
	svc := service.New(repo)

	result, err := svc.GlobalReport(context.Background())
	require.NoError(t, err)

	require.Len(t, result.UsersPerMonth, 1)
	assert.Equal(t, "2026-03", result.UsersPerMonth[0].Month, "UsersPerMonth month must be ISO '2026-03'")

	require.Len(t, result.ApprovedCoursesPerMonth, 1)
	assert.Equal(t, "2026-03", result.ApprovedCoursesPerMonth[0].Month, "ApprovedCoursesPerMonth month must be ISO '2026-03'")
}

// TestCourseReport_PassThrough verifies CourseReport returns repo data directly.
func TestCourseReport_PassThrough(t *testing.T) {
	repo := &mockRepo{
		courseStats: []repository.CourseStatRow{
			{ID: "course-1", Titulo: "Go Basics", Estado: "aprobado", Enrollments: 10, Attempts: 20, ApprovalRate: 0.75},
		},
	}
	svc := service.New(repo)

	items, err := svc.CourseReport(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "course-1", items[0].ID)
	assert.Equal(t, "Go Basics", items[0].Titulo)
	assert.InDelta(t, 0.75, items[0].ApprovalRate, 0.001)
}

// TestTeamReport_PassThrough verifies TeamReport returns repo data directly.
func TestTeamReport_PassThrough(t *testing.T) {
	repo := &mockRepo{
		teamProgress: []repository.TeamMemberRow{
			{EmpleadoID: "e1", EmpleadoNombre: "Alice", Enrolled: 3, Completed: 1},
		},
	}
	svc := service.New(repo)

	items, err := svc.TeamReport(context.Background(), "supervisor-1")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "e1", items[0].EmpleadoID)
	assert.Equal(t, "Alice", items[0].EmpleadoNombre)
}

// TestUserProgressReport_PassThrough verifies UserProgressReport delegates to repo.
func TestUserProgressReport_PassThrough(t *testing.T) {
	repo := &mockRepo{
		userProgress: repository.UserProgressRow{Enrolled: 2, Completed: 1, Attempts: 3, PassedAttempts: 2, Certificates: 1},
	}
	svc := service.New(repo)

	row, err := svc.UserProgressReport(context.Background(), "user-id")
	require.NoError(t, err)
	assert.Equal(t, int64(2), row.Enrolled)
	assert.Equal(t, int64(1), row.Completed)
	assert.Equal(t, int64(3), row.Attempts)
}
