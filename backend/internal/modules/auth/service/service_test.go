// Package service — white-box unit tests.
// No build tag: runs with standard `make backend-test`.
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/idtoken"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func defaultConfig(hostedDomain string) Config {
	return Config{
		JWTSecret:             "test-secret-32-bytes-at-minimum!!",
		JWTExpiresIn:          time.Hour,
		RefreshTokenExpiresIn: 7 * 24 * time.Hour,
		GoogleClientID:        "test-client-id",
		GoogleHostedDomain:    hostedDomain,
	}
}

// newTestService returns the concrete *service so fields can be swapped.
func newTestService(cfg Config, u users.Service, r *testutil.MockRepository) *service {
	svc := New(cfg, u, r)
	return svc.(*service)
}

func defaultUserSummary() *users.UserSummary {
	return &users.UserSummary{
		ID:     "user-123",
		Email:  "test@example.com",
		Nombre: "Test User",
		Roles:  []string{"alumno"},
	}
}

// ── LoginWithGoogle ───────────────────────────────────────────────────────────

func TestLoginWithGoogle(t *testing.T) {
	tests := []struct {
		name           string
		hostedDomain   string
		payloadHD      string
		tokenErr       error
		wantErr        error
		wantUpsertCall bool
	}{
		{
			name:           "hd_empty_skip_domain_check",
			hostedDomain:   "",
			payloadHD:      "any.com",
			tokenErr:       nil,
			wantErr:        nil,
			wantUpsertCall: true,
		},
		{
			name:           "hd_set_and_matching",
			hostedDomain:   "acme.com",
			payloadHD:      "acme.com",
			tokenErr:       nil,
			wantErr:        nil,
			wantUpsertCall: true,
		},
		{
			name:           "hd_set_but_mismatched",
			hostedDomain:   "acme.com",
			payloadHD:      "other.com",
			tokenErr:       nil,
			wantErr:        ErrUnauthorizedDomain,
			wantUpsertCall: false,
		},
		{
			name:           "invalid_google_token",
			hostedDomain:   "",
			payloadHD:      "",
			tokenErr:       errors.New("invalid token"),
			wantErr:        ErrInvalidGoogleToken,
			wantUpsertCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &testutil.MockRepository{}
			mockUsers := &testutil.MockUsersService{}

			// Always stub Insert so issueRefreshToken succeeds when login proceeds.
			mockRepo.On("Insert", mock.Anything, mock.Anything).Return(nil).Maybe()

			if tt.wantUpsertCall {
				mockUsers.On("UpsertFromGoogle", mock.Anything, mock.Anything).
					Return(defaultUserSummary(), nil)
			}

			svc := newTestService(defaultConfig(tt.hostedDomain), mockUsers, mockRepo)
			svc.validateToken = testutil.ValidateTokenStub(
				testutil.ValidPayload("sub-123", "test@example.com", "Test User", tt.payloadHD),
				tt.tokenErr,
			)

			resp, err := svc.LoginWithGoogle(context.Background(), "fake-id-token")

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
			}

			if tt.wantUpsertCall {
				mockUsers.AssertCalled(t, "UpsertFromGoogle", mock.Anything, mock.Anything)
			} else {
				mockUsers.AssertNotCalled(t, "UpsertFromGoogle", mock.Anything, mock.Anything)
			}
		})
	}
}

// ── Refresh ───────────────────────────────────────────────────────────────────

func TestRefresh(t *testing.T) {
	now := time.Now().UTC()

	validToken := testutil.RefreshTokenRow(
		testutil.WithID("rt-id"),
		testutil.WithUserID("user-123"),
		testutil.WithTokenHash("the-hash"),
		testutil.WithExpiresAt(now.Add(7*24*time.Hour)),
	)

	revokedToken := testutil.RefreshTokenRow(
		testutil.WithRevokedAt(now.Add(-time.Hour)),
	)

	expiredToken := testutil.RefreshTokenRow(
		testutil.WithExpiresAt(now.Add(-time.Minute)),
	)

	usedToken := testutil.RefreshTokenRow(
		testutil.WithUsedAt(now.Add(-time.Minute)),
	)

	tests := []struct {
		name              string
		findReturn        *domain.RefreshToken
		findErr           error
		wantErr           error
		wantMarkUsed      bool
		wantRevoke        bool
		wantRevokeAllUser bool
	}{
		{
			name:         "valid_rotation",
			findReturn:   validToken,
			wantErr:      nil,
			wantMarkUsed: true,
			wantRevoke:   true,
		},
		{
			name:       "token_not_found",
			findReturn: nil,
			findErr:    nil,
			wantErr:    ErrInvalidRefreshToken,
		},
		{
			name:       "token_already_revoked",
			findReturn: revokedToken,
			wantErr:    ErrInvalidRefreshToken,
		},
		{
			name:       "token_expired",
			findReturn: expiredToken,
			wantErr:    ErrInvalidRefreshToken,
		},
		{
			name:              "token_reuse_replay",
			findReturn:        usedToken,
			wantErr:           ErrRefreshTokenReused,
			wantRevokeAllUser: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &testutil.MockRepository{}
			mockUsers := &testutil.MockUsersService{}

			// FindByHash returns whatever the test case specifies.
			mockRepo.On("FindByHash", mock.Anything, mock.Anything).
				Return(tt.findReturn, tt.findErr)

			if tt.wantMarkUsed {
				mockRepo.On("MarkUsed", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			}
			if tt.wantRevoke {
				mockRepo.On("Revoke", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockUsers.On("GetByID", mock.Anything, mock.Anything).Return(defaultUserSummary(), nil)
				// Insert for the new refresh token issued after rotation.
				mockRepo.On("Insert", mock.Anything, mock.Anything).Return(nil)
			}
			if tt.wantRevokeAllUser {
				mockRepo.On("RevokeAllForUser", mock.Anything, mock.Anything).Return(nil)
			}

			svc := newTestService(defaultConfig(""), mockUsers, mockRepo)
			// Use a real hash of "plain-token" so crypto math works.
			_, err := svc.Refresh(context.Background(), "plain-token")

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantMarkUsed {
				mockRepo.AssertCalled(t, "MarkUsed", mock.Anything, mock.Anything, mock.Anything)
			}
			if tt.wantRevoke {
				mockRepo.AssertCalled(t, "Revoke", mock.Anything, mock.Anything, mock.Anything)
			}
			if tt.wantRevokeAllUser {
				mockRepo.AssertCalled(t, "RevokeAllForUser", mock.Anything, mock.Anything)
			}
		})
	}
}

// ── Logout ────────────────────────────────────────────────────────────────────

func TestLogout(t *testing.T) {
	now := time.Now().UTC()

	validToken := testutil.RefreshTokenRow(
		testutil.WithID("rt-id"),
		testutil.WithExpiresAt(now.Add(time.Hour)),
	)

	tests := []struct {
		name           string
		plainToken     string
		findReturn     *domain.RefreshToken
		findErr        error
		wantErr        bool
		wantFindCall   bool
		wantRevokeCall bool
	}{
		{
			name:         "empty_token_noop",
			plainToken:   "",
			wantErr:      false,
			wantFindCall: false,
		},
		{
			name:           "token_found_revoke",
			plainToken:     "some-plain-token",
			findReturn:     validToken,
			wantErr:        false,
			wantFindCall:   true,
			wantRevokeCall: true,
		},
		{
			name:           "token_not_found_idempotent",
			plainToken:     "some-plain-token",
			findReturn:     nil,
			findErr:        nil,
			wantErr:        false,
			wantFindCall:   true,
			wantRevokeCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &testutil.MockRepository{}
			mockUsers := &testutil.MockUsersService{}

			if tt.wantFindCall {
				mockRepo.On("FindByHash", mock.Anything, mock.Anything).
					Return(tt.findReturn, tt.findErr)
			}
			if tt.wantRevokeCall {
				mockRepo.On("Revoke", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			}

			svc := newTestService(defaultConfig(""), mockUsers, mockRepo)
			err := svc.Logout(context.Background(), tt.plainToken)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantFindCall {
				mockRepo.AssertCalled(t, "FindByHash", mock.Anything, mock.Anything)
			} else {
				mockRepo.AssertNotCalled(t, "FindByHash", mock.Anything, mock.Anything)
			}

			if tt.wantRevokeCall {
				mockRepo.AssertCalled(t, "Revoke", mock.Anything, mock.Anything, mock.Anything)
			} else {
				mockRepo.AssertNotCalled(t, "Revoke", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

// Ensure the idtoken import is used (Payload type).
var _ *idtoken.Payload
