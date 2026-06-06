// Package repository handles all persistence for refresh tokens.
// Only this package is allowed to use *gorm.DB directly — the service
// never accesses the DB directly.
package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/domain"
)

// ErrNotAffected is returned by RevokeByID when the UPDATE matched zero rows.
// This happens when the session does not exist, belongs to a different user,
// or is already revoked. The service maps this to ErrSessionNotFound (no
// existence leak between the three cases).
var ErrNotAffected = errors.New("no rows affected")

// Repository defines the persistence contract for refresh tokens.
// The service depends on this interface, never on gormRepository directly.
type Repository interface {
	// Insert persists a new refresh token record.
	Insert(ctx context.Context, rt *domain.RefreshToken) error

	// FindByHash looks up a refresh token by its SHA-256 hash.
	// Returns (nil, nil) when not found — the service treats nil as "not found"
	// and a non-nil error as a DB failure. This avoids leaking gorm.ErrRecordNotFound
	// across the boundary.
	FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)

	// MarkUsed sets used_at on the token with the given ID.
	MarkUsed(ctx context.Context, id string, at time.Time) error

	// Revoke sets revoked_at on the token with the given ID.
	Revoke(ctx context.Context, id string, at time.Time) error

	// RevokeChain revokes an entire rotation chain starting from rootID.
	// MVP trade-off: the current implementation calls RevokeAllForUser which is
	// broader (revokes ALL user tokens, not just the chain). This is safe because
	// OWASP reuse detection already implies all sessions should be invalidated.
	// A stricter implementation would follow parent_id via a recursive CTE:
	//   WITH RECURSIVE chain AS (SELECT id FROM refresh_token WHERE id = rootID
	//     UNION ALL SELECT rt.id FROM refresh_token rt JOIN chain c ON rt.parent_id = c.id)
	//   UPDATE refresh_token SET revoked_at = now() WHERE id IN (SELECT id FROM chain)
	RevokeChain(ctx context.Context, rootID string) error

	// RevokeAllForUser revokes all active refresh tokens for a given user.
	// Called during OWASP reuse detection when a replay attack is suspected.
	RevokeAllForUser(ctx context.Context, userID string) error

	// ListActiveByUser returns all non-expired, non-revoked sessions for the
	// given user, ordered by created_at DESC (newest first).
	// SECURITY: caller-scoped at SQL layer — user_id = callerID.
	ListActiveByUser(ctx context.Context, userID string) ([]*domain.RefreshToken, error)

	// RevokeByID sets revoked_at = now() on the row matching id AND user_id.
	// Returns ErrNotAffected when zero rows are updated (not found, wrong user,
	// or already revoked). SECURITY: WHERE user_id = callerID prevents cross-user revoke.
	RevokeByID(ctx context.Context, id, userID string) error
}

type gormRepository struct {
	db *gorm.DB
}

// New constructs a GORM-backed Repository.
func New(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Insert(ctx context.Context, rt *domain.RefreshToken) error {
	return r.db.WithContext(ctx).Create(rt).Error
}

func (r *gormRepository) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var rt domain.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", hash).
		First(&rt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // caller distinguishes nil from error
		}
		return nil, err
	}
	return &rt, nil
}

func (r *gormRepository) MarkUsed(ctx context.Context, id string, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("id = ?", id).
		Update("used_at", at).Error
}

func (r *gormRepository) Revoke(ctx context.Context, id string, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("id = ?", id).
		Update("revoked_at", at).Error
}

// RevokeChain — MVP: revokes all tokens for the user that owns rootID.
// See interface comment for the stricter recursive-CTE alternative.
func (r *gormRepository) RevokeChain(ctx context.Context, rootID string) error {
	var rt domain.RefreshToken
	if err := r.db.WithContext(ctx).
		Select("user_id").
		Where("id = ?", rootID).
		First(&rt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return r.RevokeAllForUser(ctx, rt.UserID)
}

func (r *gormRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now().UTC()).Error
}

// ListActiveByUser returns the caller's active (not revoked, not expired) sessions
// ordered newest-first. SECURITY: user_id = callerID enforced at SQL layer.
func (r *gormRepository) ListActiveByUser(ctx context.Context, userID string) ([]*domain.RefreshToken, error) {
	var rows []*domain.RefreshToken
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > NOW()", userID).
		Order("created_at DESC").
		Find(&rows).Error
	return rows, err
}

// RevokeByID sets revoked_at = now() on the matching row, enforcing caller
// scoping at the SQL layer. Returns ErrNotAffected when zero rows are updated.
// SECURITY: WHERE user_id = callerID prevents cross-user revoke.
func (r *gormRepository) RevokeByID(ctx context.Context, id, userID string) error {
	res := r.db.WithContext(ctx).
		Model(&domain.RefreshToken{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", id, userID).
		Update("revoked_at", time.Now().UTC())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotAffected
	}
	return nil
}
