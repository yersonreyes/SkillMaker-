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
	"github.com/stretchr/testify/require"
	"google.golang.org/api/idtoken"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/repository"
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

			resp, err := svc.LoginWithGoogle(context.Background(), "fake-id-token", "1.2.3.4", "Chrome/120")

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
			_, err := svc.Refresh(context.Background(), "plain-token", "1.2.3.4", "Chrome/120")

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

// ── ListActiveSessions ────────────────────────────────────────────────────────

// TestListActiveSessions verifies that the service maps repo rows to DTOs and
// always returns a non-nil slice (even for zero rows).
func TestListActiveSessions(t *testing.T) {
	now := time.Now().UTC()
	ip := "10.0.0.1"
	ua := "Chrome/120"

	row1 := testutil.RefreshTokenRow(
		testutil.WithID("rt-1"),
		testutil.WithUserID("user-123"),
		testutil.WithExpiresAt(now.Add(7*24*time.Hour)),
	)
	row1.IP = &ip
	row1.UserAgent = &ua

	row2 := testutil.RefreshTokenRow(
		testutil.WithID("rt-2"),
		testutil.WithUserID("user-123"),
		testutil.WithExpiresAt(now.Add(24*time.Hour)),
	)

	t.Run("two_rows_mapped", func(t *testing.T) {
		mockRepo := &testutil.MockRepository{}
		mockUsers := &testutil.MockUsersService{}

		rows := []*domain.RefreshToken{row1, row2}
		mockRepo.On("ListActiveByUser", mock.Anything, "user-123").Return(rows, nil)

		svc := newTestService(defaultConfig(""), mockUsers, mockRepo)
		result, err := svc.ListActiveSessions(context.Background(), "user-123")

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "rt-1", result[0].ID)
		assert.Equal(t, &ip, result[0].IP)
		assert.Equal(t, &ua, result[0].UserAgent)
		assert.Equal(t, "rt-2", result[1].ID)
		// token_hash and parent_id must NOT appear on the DTO
		mockRepo.AssertCalled(t, "ListActiveByUser", mock.Anything, "user-123")
	})

	t.Run("empty_rows_returns_non_nil_slice", func(t *testing.T) {
		mockRepo := &testutil.MockRepository{}
		mockUsers := &testutil.MockUsersService{}

		mockRepo.On("ListActiveByUser", mock.Anything, "user-empty").Return([]*domain.RefreshToken{}, nil)

		svc := newTestService(defaultConfig(""), mockUsers, mockRepo)
		result, err := svc.ListActiveSessions(context.Background(), "user-empty")

		require.NoError(t, err)
		require.NotNil(t, result, "must return non-nil empty slice, not nil")
		assert.Len(t, result, 0)
	})
}

// ── RevokeSession ─────────────────────────────────────────────────────────────

// TestRevokeSession verifies the service's session-revoke behaviour:
// (a) success, (b) ErrNotAffected → ErrSessionNotFound, (c) other error propagated.
func TestRevokeSession(t *testing.T) {
	t.Run("success_no_error", func(t *testing.T) {
		mockRepo := &testutil.MockRepository{}
		mockUsers := &testutil.MockUsersService{}

		mockRepo.On("RevokeByID", mock.Anything, "sess-id", "user-123").Return(nil)

		svc := newTestService(defaultConfig(""), mockUsers, mockRepo)
		err := svc.RevokeSession(context.Background(), "user-123", "sess-id")

		assert.NoError(t, err)
		mockRepo.AssertCalled(t, "RevokeByID", mock.Anything, "sess-id", "user-123")
	})

	t.Run("not_affected_maps_to_session_not_found", func(t *testing.T) {
		mockRepo := &testutil.MockRepository{}
		mockUsers := &testutil.MockUsersService{}

		mockRepo.On("RevokeByID", mock.Anything, "bad-id", "user-123").Return(repository.ErrNotAffected)

		svc := newTestService(defaultConfig(""), mockUsers, mockRepo)
		err := svc.RevokeSession(context.Background(), "user-123", "bad-id")

		assert.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("other_error_propagated", func(t *testing.T) {
		mockRepo := &testutil.MockRepository{}
		mockUsers := &testutil.MockUsersService{}

		dbErr := errors.New("database unavailable")
		mockRepo.On("RevokeByID", mock.Anything, "some-id", "user-123").Return(dbErr)

		svc := newTestService(defaultConfig(""), mockUsers, mockRepo)
		err := svc.RevokeSession(context.Background(), "user-123", "some-id")

		assert.ErrorIs(t, err, dbErr)
		assert.NotErrorIs(t, err, ErrSessionNotFound)
	})
}

// ── W1: ip/ua reach repo.Insert at service layer ─────────────────────────────

// TestLoginWithGoogle_InsertReceivesIPAndUserAgent asserts that the
// ip and userAgent passed to LoginWithGoogle reach the domain.RefreshToken
// persisted via repo.Insert (service-layer threading proof, W1).
func TestLoginWithGoogle_InsertReceivesIPAndUserAgent(t *testing.T) {
	const wantIP = "1.2.3.4"
	const wantUA = "TestAgent/1.0"

	var capturedRT *domain.RefreshToken

	mockRepo := &testutil.MockRepository{}
	mockUsers := &testutil.MockUsersService{}

	mockRepo.On("Insert", mock.Anything, mock.MatchedBy(func(rt *domain.RefreshToken) bool {
		capturedRT = rt
		return true
	})).Return(nil)
	mockUsers.On("UpsertFromGoogle", mock.Anything, mock.Anything).
		Return(defaultUserSummary(), nil)

	svc := newTestService(defaultConfig(""), mockUsers, mockRepo)
	svc.validateToken = testutil.ValidateTokenStub(
		testutil.ValidPayload("sub-123", "test@example.com", "Test User", ""),
		nil,
	)

	_, err := svc.LoginWithGoogle(context.Background(), "fake-token", wantIP, wantUA)

	require.NoError(t, err)
	require.NotNil(t, capturedRT, "repo.Insert must have been called")
	require.NotNil(t, capturedRT.IP, "rt.IP must be non-nil for non-empty ip")
	assert.Equal(t, wantIP, *capturedRT.IP, "rt.IP must equal the ip passed to LoginWithGoogle")
	require.NotNil(t, capturedRT.UserAgent, "rt.UserAgent must be non-nil for non-empty ua")
	assert.Equal(t, wantUA, *capturedRT.UserAgent, "rt.UserAgent must equal the userAgent passed to LoginWithGoogle")
}

// TestRefresh_InsertReceivesIPAndUserAgent asserts that the ip and userAgent
// passed to Refresh reach the new domain.RefreshToken persisted via repo.Insert
// after token rotation (service-layer threading proof, W1).
func TestRefresh_InsertReceivesIPAndUserAgent(t *testing.T) {
	const wantIP = "5.6.7.8"
	const wantUA = "RefreshAgent/2.0"

	now := time.Now().UTC()
	validToken := testutil.RefreshTokenRow(
		testutil.WithID("rt-id"),
		testutil.WithUserID("user-123"),
		testutil.WithTokenHash("the-hash"),
		testutil.WithExpiresAt(now.Add(7*24*time.Hour)),
	)

	var capturedRT *domain.RefreshToken

	mockRepo := &testutil.MockRepository{}
	mockUsers := &testutil.MockUsersService{}

	mockRepo.On("FindByHash", mock.Anything, mock.Anything).Return(validToken, nil)
	mockRepo.On("MarkUsed", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("Revoke", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockUsers.On("GetByID", mock.Anything, mock.Anything).Return(defaultUserSummary(), nil)
	mockRepo.On("Insert", mock.Anything, mock.MatchedBy(func(rt *domain.RefreshToken) bool {
		capturedRT = rt
		return true
	})).Return(nil)

	svc := newTestService(defaultConfig(""), mockUsers, mockRepo)
	// Use a real hash of "plain-token" so crypto math works.
	_, err := svc.Refresh(context.Background(), "plain-token", wantIP, wantUA)

	require.NoError(t, err)
	require.NotNil(t, capturedRT, "repo.Insert must have been called for the rotated token")
	require.NotNil(t, capturedRT.IP, "rt.IP must be non-nil for non-empty ip")
	assert.Equal(t, wantIP, *capturedRT.IP, "rt.IP must equal the ip passed to Refresh")
	require.NotNil(t, capturedRT.UserAgent, "rt.UserAgent must be non-nil for non-empty ua")
	assert.Equal(t, wantUA, *capturedRT.UserAgent, "rt.UserAgent must equal the userAgent passed to Refresh")
}

// Ensure the idtoken import is used (Payload type).
var _ *idtoken.Payload
