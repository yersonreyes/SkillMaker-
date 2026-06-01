// Package testutil provides shared test helpers for the backend modules.
// mocks.go contains testify/mock implementations of the auth interfaces.
// No build tag — compiles in both unit and integration builds.
package testutil

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"google.golang.org/api/idtoken"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users"
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
type MockUsersService struct {
	mock.Mock
}

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

// ValidateTokenStub returns a stub function that always returns the given payload and error.
// Assign to service.validateToken in white-box unit tests.
func ValidateTokenStub(payload *idtoken.Payload, err error) func(ctx context.Context, idToken, audience string) (*idtoken.Payload, error) {
	return func(_ context.Context, _, _ string) (*idtoken.Payload, error) {
		return payload, err
	}
}
