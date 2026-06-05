// Package testutil provides shared test helpers for the backend modules.
// mocks.go contains testify/mock implementations of the auth and users interfaces.
// No build tag — compiles in both unit and integration builds.
package testutil

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"google.golang.org/api/idtoken"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// MockRepository is a testify/mock implementation of repository.Repository.
// Use it in unit tests to avoid database dependencies.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Insert(ctx context.Context, rt *domain.RefreshToken) error {
	args := m.Called(ctx, rt)
	return args.Error(0)
}

func (m *MockRepository) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.RefreshToken), args.Error(1)
}

func (m *MockRepository) MarkUsed(ctx context.Context, id string, at time.Time) error {
	args := m.Called(ctx, id, at)
	return args.Error(0)
}

func (m *MockRepository) Revoke(ctx context.Context, id string, at time.Time) error {
	args := m.Called(ctx, id, at)
	return args.Error(0)
}

func (m *MockRepository) RevokeChain(ctx context.Context, rootID string) error {
	args := m.Called(ctx, rootID)
	return args.Error(0)
}

func (m *MockRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockUsersService is a testify/mock implementation of users.Service.
// It covers ALL methods of the Service interface, including the C1.1 additions.
// This is required to keep auth/service/service_test.go compiling after the
// users.Service interface was expanded.
type MockUsersService struct {
	mock.Mock
}

// ── existing methods ──────────────────────────────────────────────────────────

func (m *MockUsersService) UpsertFromGoogle(ctx context.Context, profile users.GoogleProfile) (*users.UserSummary, error) {
	args := m.Called(ctx, profile)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*users.UserSummary), args.Error(1)
}

func (m *MockUsersService) GetByID(ctx context.Context, id string) (*users.UserSummary, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*users.UserSummary), args.Error(1)
}

// ── C1.1 additions ─────────────────────────────────────────────────────────────

func (m *MockUsersService) List(ctx context.Context, f users.ListFilters, p pagination.Params) (pagination.Page[users.UserDetailModel], error) {
	args := m.Called(ctx, f, p)
	return args.Get(0).(pagination.Page[users.UserDetailModel]), args.Error(1)
}

func (m *MockUsersService) GetDetail(ctx context.Context, id string) (*users.UserDetailModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*users.UserDetailModel), args.Error(1)
}

func (m *MockUsersService) PatchRoles(ctx context.Context, id string, add, remove []string) (*users.UserDetailModel, error) {
	args := m.Called(ctx, id, add, remove)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*users.UserDetailModel), args.Error(1)
}

func (m *MockUsersService) SetActive(ctx context.Context, id string, active bool) (*users.UserDetailModel, error) {
	args := m.Called(ctx, id, active)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*users.UserDetailModel), args.Error(1)
}

func (m *MockUsersService) CreateSupervision(ctx context.Context, supervisorID, empleadoID string) (*users.SupervisionModel, error) {
	args := m.Called(ctx, supervisorID, empleadoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*users.SupervisionModel), args.Error(1)
}

func (m *MockUsersService) ListSupervisions(ctx context.Context) ([]users.SupervisionModel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]users.SupervisionModel), args.Error(1)
}

func (m *MockUsersService) DeleteSupervision(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ValidateTokenStub returns a stub function that always returns the given payload and error.
// Assign to service.validateToken in white-box unit tests.
func ValidateTokenStub(payload *idtoken.Payload, err error) func(ctx context.Context, idToken, audience string) (*idtoken.Payload, error) {
	return func(_ context.Context, _, _ string) (*idtoken.Payload, error) {
		return payload, err
	}
}

// MockCoursesChecker is a testify/mock implementation of the evaluations.CoursesChecker interface.
// Used in evaluations service unit tests to avoid importing the courses module.
// Mirrors MockUsersService — a narrow single-method mock for the cross-module seam (C3.1, ADR-9).
type MockCoursesChecker struct {
	mock.Mock
}

// GetCourseOwnership returns the mocked creadorID, estado, and error for a courseID.
func (m *MockCoursesChecker) GetCourseOwnership(ctx context.Context, courseID string) (creadorID, estado string, err error) {
	args := m.Called(ctx, courseID)
	return args.String(0), args.String(1), args.Error(2)
}
