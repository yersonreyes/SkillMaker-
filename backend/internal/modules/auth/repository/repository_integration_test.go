//go:build integration

// Package repository — integration tests.
// Build tag: integration — requires Docker and a running daemon.
// Run with: make backend-test-integration
package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// hashPlainToken returns the SHA-256 hex hash used by the repository.
func hashPlainToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// seedUser inserts a minimal "user" row and returns its UUID.
// The refresh_token table has a FK to "user"(id).
func seedUser(t *testing.T, repo Repository) string {
	t.Helper()
	// We can't use the users repository here (wrong package), so insert raw SQL
	// via the underlying *gorm.DB. Access it through the gormRepository field —
	// white-box test in same package.
	r := repo.(*gormRepository)
	id := uuid.NewString()
	err := r.db.Exec(`INSERT INTO "user" (id, google_sub, email, nombre)
		VALUES (?, ?, ?, ?)`, id, uuid.NewString(), uuid.NewString()+"@test.com", "Test User").Error
	require.NoError(t, err)
	return id
}

func TestInsertAndFindByHash(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := New(db)
	ctx := context.Background()
	userID := seedUser(t, repo)

	plain := "test-token-insert-find"
	hash := hashPlainToken(plain)

	rt := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}

	err := repo.Insert(ctx, rt)
	require.NoError(t, err)

	found, err := repo.FindByHash(ctx, hash)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, rt.ID, found.ID)
	assert.Equal(t, rt.UserID, found.UserID)
	assert.Equal(t, rt.TokenHash, found.TokenHash)
	assert.Nil(t, found.UsedAt)
	assert.Nil(t, found.RevokedAt)
}

func TestMarkUsed_SetsUsedAt(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := New(db)
	ctx := context.Background()
	userID := seedUser(t, repo)

	plain := "test-token-mark-used"
	rt := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: hashPlainToken(plain),
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	require.NoError(t, repo.Insert(ctx, rt))

	usedAt := time.Now().UTC().Truncate(time.Millisecond)
	require.NoError(t, repo.MarkUsed(ctx, rt.ID, usedAt))

	found, err := repo.FindByHash(ctx, rt.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.NotNil(t, found.UsedAt)
	// Timestamps may differ by sub-millisecond due to DB rounding — compare at second precision.
	assert.WithinDuration(t, usedAt, *found.UsedAt, time.Second)
}

func TestRevoke_SetsRevokedAt(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := New(db)
	ctx := context.Background()
	userID := seedUser(t, repo)

	rt := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: hashPlainToken("test-token-revoke"),
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	require.NoError(t, repo.Insert(ctx, rt))

	revokedAt := time.Now().UTC()
	require.NoError(t, repo.Revoke(ctx, rt.ID, revokedAt))

	found, err := repo.FindByHash(ctx, rt.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.NotNil(t, found.RevokedAt)
}

func TestRevokeAllForUser_RevokesAllActiveTokens(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := New(db)
	ctx := context.Background()
	userID := seedUser(t, repo)

	rt1 := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: hashPlainToken("token-a"),
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	rt2 := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: hashPlainToken("token-b"),
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}

	require.NoError(t, repo.Insert(ctx, rt1))
	require.NoError(t, repo.Insert(ctx, rt2))

	require.NoError(t, repo.RevokeAllForUser(ctx, userID))

	f1, err := repo.FindByHash(ctx, rt1.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, f1)
	assert.NotNil(t, f1.RevokedAt, "rt1 should be revoked")

	f2, err := repo.FindByHash(ctx, rt2.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, f2)
	assert.NotNil(t, f2.RevokedAt, "rt2 should be revoked")
}

func TestFindByHash_ReturnsNilForUnknownHash(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := New(db)
	ctx := context.Background()

	found, err := repo.FindByHash(ctx, "unknown-hash-that-does-not-exist")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

// ── C8.1: ip/ua round-trip + caller-scoped list/revoke ────────────────────────

// TestInsert_WithIPAndUserAgent_RoundTrip verifies that ip and user_agent fields
// are stored and retrieved correctly (inet round-trip, text round-trip).
func TestInsert_WithIPAndUserAgent_RoundTrip(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := New(db)
	ctx := context.Background()
	userID := seedUser(t, repo)

	ip := "10.0.0.1"
	ua := "TestAgent/1.0"
	rt := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userID,
		TokenHash: hashPlainToken("token-ip-ua"),
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
		IP:        &ip,
		UserAgent: &ua,
	}
	require.NoError(t, repo.Insert(ctx, rt))

	found, err := repo.FindByHash(ctx, rt.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.NotNil(t, found.IP, "IP should round-trip through inet column")
	require.NotNil(t, found.UserAgent, "UserAgent should round-trip through text column")
	assert.Equal(t, ip, *found.IP)
	assert.Equal(t, ua, *found.UserAgent)
}

// TestListActiveByUser_OnlyActiveAndCallerScoped verifies:
// - Only active (not revoked, not expired) sessions are returned.
// - Only the caller's sessions are returned (user B's row never appears).
// - Results are ordered created_at DESC.
// SECURITY-CRITICAL: cross-user exclusion is the primary authz boundary.
func TestListActiveByUser_OnlyActiveAndCallerScoped(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := New(db)
	ctx := context.Background()

	userA := seedUser(t, repo)
	userB := seedUser(t, repo)

	now := time.Now().UTC()

	// userA: 1 active, 1 revoked, 1 expired
	activeA := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userA,
		TokenHash: hashPlainToken("active-a"),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	require.NoError(t, repo.Insert(ctx, activeA))

	revokedA := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userA,
		TokenHash: hashPlainToken("revoked-a"),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	require.NoError(t, repo.Insert(ctx, revokedA))
	require.NoError(t, repo.Revoke(ctx, revokedA.ID, now))

	expiredA := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userA,
		TokenHash: hashPlainToken("expired-a"),
		ExpiresAt: now.Add(-time.Minute), // already expired
	}
	require.NoError(t, repo.Insert(ctx, expiredA))

	// userB: 1 active
	activeB := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userB,
		TokenHash: hashPlainToken("active-b"),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	require.NoError(t, repo.Insert(ctx, activeB))

	rows, err := repo.ListActiveByUser(ctx, userA)
	require.NoError(t, err)

	// Must return exactly 1 row — the active A row only.
	require.Len(t, rows, 1, "ListActiveByUser must return exactly 1 active row for userA")
	assert.Equal(t, activeA.ID, rows[0].ID)

	// userB's session must never appear.
	for _, r := range rows {
		assert.NotEqual(t, userB, r.UserID, "userB's sessions must never appear in userA's list")
	}
}

// TestRevokeByID_CallerScoped verifies:
// - A can revoke their own session (RowsAffected 1).
// - A cannot revoke B's session (ErrNotAffected AND B's row.revoked_at stays NULL).
// - Already-revoked session → ErrNotAffected.
// SECURITY-CRITICAL: cross-user revoke no-op is the primary authz boundary.
func TestRevokeByID_CallerScoped(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := New(db)
	ctx := context.Background()

	userA := seedUser(t, repo)
	userB := seedUser(t, repo)

	now := time.Now().UTC()

	rowA := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userA,
		TokenHash: hashPlainToken("row-a"),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	require.NoError(t, repo.Insert(ctx, rowA))

	rowB := &domain.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    userB,
		TokenHash: hashPlainToken("row-b"),
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	require.NoError(t, repo.Insert(ctx, rowB))

	// A revokes their own row — must succeed.
	err := repo.RevokeByID(ctx, rowA.ID, userA)
	assert.NoError(t, err, "A revoking their own session must succeed")

	// Verify A's row is now revoked.
	foundA, err := repo.FindByHash(ctx, rowA.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, foundA)
	assert.NotNil(t, foundA.RevokedAt, "A's row must be revoked after RevokeByID")

	// A tries to revoke B's row — must return ErrNotAffected.
	err = repo.RevokeByID(ctx, rowB.ID, userA)
	assert.ErrorIs(t, err, ErrNotAffected, "A revoking B's session must return ErrNotAffected")

	// Verify B's row is NOT modified (revoked_at still NULL).
	foundB, err := repo.FindByHash(ctx, rowB.TokenHash)
	require.NoError(t, err)
	require.NotNil(t, foundB)
	assert.Nil(t, foundB.RevokedAt, "B's session must remain unrevoked after A's failed cross-user revoke attempt")

	// Already-revoked A row → ErrNotAffected (idempotent-safe).
	err = repo.RevokeByID(ctx, rowA.ID, userA)
	assert.ErrorIs(t, err, ErrNotAffected, "revoking an already-revoked session must return ErrNotAffected")
}
