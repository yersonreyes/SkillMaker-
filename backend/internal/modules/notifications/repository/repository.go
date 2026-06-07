// Package repository contains the GORM-backed data-access layer for the notifications module.
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/domain"
	notifService "github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// ErrNotFound is re-exported from service so that errors.Is works across the
// repository → service boundary without requiring callers to import repository.
var ErrNotFound = notifService.ErrNotFound

// Repository defines the data-access contract for the notifications module.
type Repository interface {
	Create(ctx context.Context, n *domain.Notification) error
	ListByUser(ctx context.Context, userID string, p pagination.Params) ([]domain.Notification, int64, error)
	CountUnread(ctx context.Context, userID string) (int64, error)
	// MarkRead is CALLER-SCOPED: WHERE id=? AND user_id=?. Returns ErrNotFound
	// when RowsAffected==0 (notification absent or belongs to another user).
	MarkRead(ctx context.Context, id, userID string) error
	// MarkAllRead sets leida=true for all of userID's notifications.
	MarkAllRead(ctx context.Context, userID string) error
}

type gormRepository struct {
	db *gorm.DB
}

// New creates a GORM-backed Repository.
func New(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

// Create persists a new notification row.
func (r *gormRepository) Create(ctx context.Context, n *domain.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

// ListByUser returns the caller's notifications paginated and ordered by creado_en DESC.
func (r *gormRepository) ListByUser(ctx context.Context, userID string, p pagination.Params) ([]domain.Notification, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []domain.Notification
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("creado_en DESC").
		Offset(p.Offset()).
		Limit(p.Limit()).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

// CountUnread returns the number of unread notifications for userID.
func (r *gormRepository) CountUnread(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Notification{}).
		Where("user_id = ? AND leida = false", userID).
		Count(&count).Error
	return count, err
}

// MarkRead marks a single notification as read.
// SECURITY: Uses WHERE id=? AND user_id=? so a user cannot mark another user's notification.
// Returns ErrNotFound when RowsAffected == 0.
func (r *gormRepository) MarkRead(ctx context.Context, id, userID string) error {
	res := r.db.WithContext(ctx).Exec(
		"UPDATE notification SET leida = true WHERE id = ? AND user_id = ?",
		id, userID,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkAllRead marks all of the user's unread notifications as read.
// No-op (no error) when there are no unread notifications.
func (r *gormRepository) MarkAllRead(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Exec(
		"UPDATE notification SET leida = true WHERE user_id = ? AND leida = false",
		userID,
	).Error
}
