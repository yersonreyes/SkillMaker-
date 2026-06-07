// Package service — unit tests for the notifications service.
// No build tag: runs with standard `make backend-test`.
//
// Strategy: inject a mock Repository so no real DB is needed.
// TDD: tests written BEFORE the service implementation (Strict TDD).
package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// ── mock repository ────────────────────────────────────────────────────────────

type mockRepo struct {
	CreateFn      func(ctx context.Context, n *domain.Notification) error
	ListByUserFn  func(ctx context.Context, userID string, p pagination.Params) ([]domain.Notification, int64, error)
	CountUnreadFn func(ctx context.Context, userID string) (int64, error)
	MarkReadFn    func(ctx context.Context, id, userID string) error
	MarkAllReadFn func(ctx context.Context, userID string) error

	createCalls []domain.Notification
}

func (m *mockRepo) Create(ctx context.Context, n *domain.Notification) error {
	m.createCalls = append(m.createCalls, *n)
	if m.CreateFn != nil {
		return m.CreateFn(ctx, n)
	}
	return nil
}

func (m *mockRepo) ListByUser(ctx context.Context, userID string, p pagination.Params) ([]domain.Notification, int64, error) {
	if m.ListByUserFn != nil {
		return m.ListByUserFn(ctx, userID, p)
	}
	return nil, 0, nil
}

func (m *mockRepo) CountUnread(ctx context.Context, userID string) (int64, error) {
	if m.CountUnreadFn != nil {
		return m.CountUnreadFn(ctx, userID)
	}
	return 0, nil
}

func (m *mockRepo) MarkRead(ctx context.Context, id, userID string) error {
	if m.MarkReadFn != nil {
		return m.MarkReadFn(ctx, id, userID)
	}
	return nil
}

func (m *mockRepo) MarkAllRead(ctx context.Context, userID string) error {
	if m.MarkAllReadFn != nil {
		return m.MarkAllReadFn(ctx, userID)
	}
	return nil
}

// ── Notify tests ───────────────────────────────────────────────────────────────

// TestNotify_BuildsCorrectDomainObject verifies that Notify builds the correct
// domain.Notification with tipo passthrough; refID nil when empty string.
func TestNotify_BuildsCorrectDomainObject(t *testing.T) {
	repo := &mockRepo{}
	svc := New(repo)

	err := svc.Notify(
		context.Background(),
		"user-1",
		domain.TipoCursoAprobado,
		"Curso aprobado",
		"Tu curso fue aprobado.",
		"", // empty refID → nil
	)
	require.NoError(t, err)
	require.Len(t, repo.createCalls, 1)

	n := repo.createCalls[0]
	assert.Equal(t, "user-1", n.UserID)
	assert.Equal(t, domain.TipoCursoAprobado, n.Tipo)
	assert.Equal(t, "Curso aprobado", n.Titulo)
	assert.Equal(t, "Tu curso fue aprobado.", n.Cuerpo)
	assert.Nil(t, n.RefID, "empty refID must produce nil pointer")
	assert.NotEmpty(t, n.ID, "ID must be set")
	assert.False(t, n.CreadoEn.IsZero(), "CreadoEn must be set")
}

// TestNotify_NonEmptyRefID verifies that a non-empty refID produces a non-nil pointer.
func TestNotify_NonEmptyRefID(t *testing.T) {
	repo := &mockRepo{}
	svc := New(repo)

	refID := "cert-123"
	err := svc.Notify(
		context.Background(),
		"user-1",
		domain.TipoCertificadoEmitido,
		"Cert",
		"Cuerpo",
		refID,
	)
	require.NoError(t, err)
	require.Len(t, repo.createCalls, 1)

	n := repo.createCalls[0]
	require.NotNil(t, n.RefID, "non-empty refID must produce non-nil pointer")
	assert.Equal(t, refID, *n.RefID)
}

// ── ListMine tests ─────────────────────────────────────────────────────────────

// TestListMine_MapsPagination verifies that ListMine maps repo results to a pagination.Page.
func TestListMine_MapsPagination(t *testing.T) {
	now := time.Now().UTC()
	refID := "ref-abc"
	repo := &mockRepo{
		ListByUserFn: func(_ context.Context, _ string, _ pagination.Params) ([]domain.Notification, int64, error) {
			return []domain.Notification{
				{
					ID:       "n1",
					UserID:   "user-1",
					Tipo:     domain.TipoCursoAprobado,
					Titulo:   "T",
					Cuerpo:   "C",
					Leida:    false,
					RefID:    &refID,
					CreadoEn: now,
				},
			}, 1, nil
		},
	}
	svc := New(repo)

	page, err := svc.ListMine(context.Background(), "user-1", pagination.Params{Page: 1, Size: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	assert.Len(t, page.Items, 1)
	assert.Equal(t, "n1", page.Items[0].ID)
	assert.Equal(t, refID, page.Items[0].RefID) // non-nil pointer → string
}

// ── MarkRead tests ─────────────────────────────────────────────────────────────

// TestMarkRead_RepoNotFound_MapsToServiceErrNotFound verifies that when the
// repository returns ErrNotFound (re-exported from service), MarkRead surfaces
// service.ErrNotFound to the caller.
func TestMarkRead_RepoNotFound_MapsToServiceErrNotFound(t *testing.T) {
	repo := &mockRepo{
		MarkReadFn: func(_ context.Context, _, _ string) error {
			// Simulate what repository.MarkRead returns on 0 rows:
			// repository.ErrNotFound == service.ErrNotFound (same pointer via re-export).
			return ErrNotFound
		},
	}
	svc := New(repo)

	err := svc.MarkRead(context.Background(), "some-id", "user-1")
	assert.ErrorIs(t, err, ErrNotFound, "repo ErrNotFound must be mapped to service ErrNotFound")
}

// ── UnreadCount + MarkAllRead delegation tests ─────────────────────────────────

// TestUnreadCount_Delegates verifies that UnreadCount calls repo.CountUnread.
func TestUnreadCount_Delegates(t *testing.T) {
	called := false
	repo := &mockRepo{
		CountUnreadFn: func(_ context.Context, userID string) (int64, error) {
			called = true
			assert.Equal(t, "user-1", userID)
			return 5, nil
		},
	}
	svc := New(repo)

	count, err := svc.UnreadCount(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	assert.True(t, called)
}

// TestMarkAllRead_Delegates verifies that MarkAllRead calls repo.MarkAllRead.
func TestMarkAllRead_Delegates(t *testing.T) {
	called := false
	repo := &mockRepo{
		MarkAllReadFn: func(_ context.Context, userID string) error {
			called = true
			assert.Equal(t, "user-1", userID)
			return nil
		},
	}
	svc := New(repo)

	err := svc.MarkAllRead(context.Background(), "user-1")
	require.NoError(t, err)
	assert.True(t, called)
}
