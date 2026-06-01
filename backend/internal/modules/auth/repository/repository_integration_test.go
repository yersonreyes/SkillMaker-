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
