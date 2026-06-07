// Package service contains the business logic for the notifications module.
// It is HTTP-agnostic: returns domain sentinels and read-models.
// This is a LEAF module — it imports NOTHING from other domain modules.
//
// Cross-module contract (structural, NOT imported):
//
//	The Notify method signature satisfies the Notifier interfaces declared
//	independently in approvals/service and certificates/service (duck typing).
//	No import of those packages here; the acyclic constraint is maintained.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// ── Read model ─────────────────────────────────────────────────────────────────

// NotificationModel is the service-layer read model returned to handlers and DTOs.
type NotificationModel struct {
	ID       string
	Tipo     string
	Titulo   string
	Cuerpo   string
	Leida    bool
	RefID    string // empty string when domain.Notification.RefID is nil
	CreadoEn time.Time
}

// ── Repository interface ───────────────────────────────────────────────────────

// Repository is the data-access seam for the notifications service.
type Repository interface {
	Create(ctx context.Context, n *domain.Notification) error
	ListByUser(ctx context.Context, userID string, p pagination.Params) ([]domain.Notification, int64, error)
	CountUnread(ctx context.Context, userID string) (int64, error)
	MarkRead(ctx context.Context, id, userID string) error
	MarkAllRead(ctx context.Context, userID string) error
}

// ── Service interface ──────────────────────────────────────────────────────────

// Service is the public interface of the notifications domain.
//
// The Notify method matches the Notifier interface declared in approvals/service
// and certificates/service (structural duck typing — no import cycle).
type Service interface {
	// Notify creates a new notification for userID.
	// Signature is frozen: satisfies both consumer Notifier seams structurally.
	Notify(ctx context.Context, userID, tipo, titulo, cuerpo, refID string) error

	// ListMine returns paginated notifications for userID, ordered by creado_en DESC.
	ListMine(ctx context.Context, userID string, p pagination.Params) (pagination.Page[NotificationModel], error)

	// UnreadCount returns the number of unread notifications for userID.
	UnreadCount(ctx context.Context, userID string) (int64, error)

	// MarkRead marks a single notification as read (caller-scoped).
	// Returns ErrNotFound if the notification doesn't exist or belongs to another user.
	MarkRead(ctx context.Context, id, userID string) error

	// MarkAllRead marks all of userID's notifications as read.
	MarkAllRead(ctx context.Context, userID string) error
}

// ── concrete implementation ────────────────────────────────────────────────────

type serviceImpl struct {
	repo Repository
}

// New creates a Service backed by the given Repository.
// No functional options needed — notifications is a pure leaf.
func New(repo Repository) Service {
	return &serviceImpl{repo: repo}
}

// Notify creates a new notification.
// refID="" → nil pointer in domain; non-empty → pointer to refID.
func (s *serviceImpl) Notify(ctx context.Context, userID, tipo, titulo, cuerpo, refID string) error {
	n := &domain.Notification{
		ID:       uuid.New().String(),
		UserID:   userID,
		Tipo:     tipo,
		Titulo:   titulo,
		Cuerpo:   cuerpo,
		Leida:    false,
		CreadoEn: time.Now().UTC(),
	}
	if refID != "" {
		r := refID
		n.RefID = &r
	}
	return s.repo.Create(ctx, n)
}

// ListMine returns paginated notifications for userID.
func (s *serviceImpl) ListMine(ctx context.Context, userID string, p pagination.Params) (pagination.Page[NotificationModel], error) {
	rows, total, err := s.repo.ListByUser(ctx, userID, p)
	if err != nil {
		return pagination.Page[NotificationModel]{}, err
	}

	items := make([]NotificationModel, 0, len(rows))
	for i := range rows {
		items = append(items, toModel(&rows[i]))
	}

	return pagination.NewPage(items, total, p), nil
}

// UnreadCount returns the number of unread notifications for userID.
func (s *serviceImpl) UnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.repo.CountUnread(ctx, userID)
}

// MarkRead marks a single notification as read (caller-scoped).
// The repository returns ErrNotFound (same sentinel via re-export) when the
// (id, user_id) pair matches no row — either absent or cross-user attempt.
func (s *serviceImpl) MarkRead(ctx context.Context, id, userID string) error {
	err := s.repo.MarkRead(ctx, id, userID)
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	return err
}

// MarkAllRead marks all of the user's notifications as read.
func (s *serviceImpl) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// ── helpers ────────────────────────────────────────────────────────────────────

func toModel(n *domain.Notification) NotificationModel {
	refID := ""
	if n.RefID != nil {
		refID = *n.RefID
	}
	return NotificationModel{
		ID:       n.ID,
		Tipo:     n.Tipo,
		Titulo:   n.Titulo,
		Cuerpo:   n.Cuerpo,
		Leida:    n.Leida,
		RefID:    refID,
		CreadoEn: n.CreadoEn,
	}
}
