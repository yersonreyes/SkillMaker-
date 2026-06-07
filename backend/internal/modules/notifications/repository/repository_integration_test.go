//go:build integration

// Package repository — integration tests for the notifications repository.
// Covers migration 0015: table created, indexes, CHECK constraint round-trip.
// Run with: make backend-test-integration
package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// ── helpers ────────────────────────────────────────────────────────────────────

func seedUser(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO "user" (id, google_sub, email, nombre, activo)
		 VALUES (?, ?, ?, ?, true)`,
		id, "sub-"+id, id+"@example.com", "Test User "+id[:8],
	).Error
	require.NoError(t, err, "seedUser: failed to insert user")
	return id
}

// ── TestMigration0015RoundTrip ─────────────────────────────────────────────────

// TestMigration0015RoundTrip verifies that migration 0015 creates the notification
// table with both indexes. m.Steps(-1) drops it cleanly (HEAD-relative, stable).
func TestMigration0015RoundTrip(t *testing.T) {
	db, m, teardown := testutil.SetupPostgresWithMigrate(t)
	defer func() { m.Close(); teardown() }()

	ctx := context.Background()

	// Verify notification table exists after 0015 up.
	var tableCount int64
	err := db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_name = 'notification'`,
	).Scan(&tableCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), tableCount, "notification table must exist after 0015 up")

	// Verify partial index (user_id) WHERE leida=false exists.
	var partialIdxCount int64
	err = db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM pg_indexes
		 WHERE schemaname = 'public'
		   AND tablename = 'notification'
		   AND indexname = 'idx_notification_user_unread'`,
	).Scan(&partialIdxCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), partialIdxCount, "idx_notification_user_unread partial index must exist after 0015 up")

	// Verify composite index (user_id, creado_en DESC) exists.
	var compositeIdxCount int64
	err = db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM pg_indexes
		 WHERE schemaname = 'public'
		   AND tablename = 'notification'
		   AND indexname = 'idx_notification_user_created'`,
	).Scan(&compositeIdxCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), compositeIdxCount, "idx_notification_user_created composite index must exist after 0015 up")

	// Verify tipo CHECK constraint rejects an invalid value.
	userID := seedUser(t, db)
	err = db.WithContext(ctx).Exec(
		`INSERT INTO notification (id, user_id, tipo, titulo, cuerpo, leida, creado_en)
		 VALUES (gen_random_uuid(), ?, 'tipo_invalido', 'T', 'C', false, now())`,
		userID,
	).Error
	assert.Error(t, err, "tipo CHECK constraint must reject 'tipo_invalido'")

	// Roll back 1 step — drops 0015 (HEAD-relative; stable because this test owns 0015).
	require.NoError(t, m.Steps(-1), "m.Steps(-1) must roll back 0015 (notification table) without error")

	// Verify table gone after down.
	err = db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_name = 'notification'`,
	).Scan(&tableCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), tableCount, "notification table must be gone after 0015 down")
}

// ── TestNotificationRepository_Create ────────────────────────────────────────

func TestNotificationRepository_Create(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)

	// Create with nil refID.
	n := &domain.Notification{
		ID:       uuid.New().String(),
		UserID:   userID,
		Tipo:     domain.TipoCursoAprobado,
		Titulo:   "Curso aprobado",
		Cuerpo:   "Tu curso fue aprobado.",
		Leida:    false,
		RefID:    nil,
		CreadoEn: time.Now().UTC(),
	}
	err := repo.Create(ctx, n)
	require.NoError(t, err, "Create with nil refID must succeed")

	// Verify round-trip.
	var count int64
	err = db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM notification WHERE id = ?`, n.ID,
	).Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "created notification must be persisted")

	// Create with non-nil refID.
	refID := uuid.New().String()
	n2 := &domain.Notification{
		ID:       uuid.New().String(),
		UserID:   userID,
		Tipo:     domain.TipoCertificadoEmitido,
		Titulo:   "Certificado emitido",
		Cuerpo:   "Tu certificado fue emitido.",
		Leida:    false,
		RefID:    &refID,
		CreadoEn: time.Now().UTC(),
	}
	err = repo.Create(ctx, n2)
	require.NoError(t, err, "Create with non-nil refID must succeed")
}

// ── TestNotificationRepository_ListByUser_CallerScoped ──────────────────────

// TestNotificationRepository_ListByUser_CallerScoped verifies that ListByUser
// returns ONLY the caller's notifications, paginated, ordered by creado_en DESC.
func TestNotificationRepository_ListByUser_CallerScoped(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	userA := seedUser(t, db)
	userB := seedUser(t, db)

	// Seed 3 notifications for A (with different timestamps).
	for i := 0; i < 3; i++ {
		n := &domain.Notification{
			ID:       uuid.New().String(),
			UserID:   userA,
			Tipo:     domain.TipoCursoAprobado,
			Titulo:   "Notif A",
			Cuerpo:   "Cuerpo A",
			Leida:    false,
			CreadoEn: time.Now().UTC().Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, repo.Create(ctx, n))
	}
	// Seed 2 notifications for B.
	for i := 0; i < 2; i++ {
		n := &domain.Notification{
			ID:       uuid.New().String(),
			UserID:   userB,
			Tipo:     domain.TipoCursoAprobado,
			Titulo:   "Notif B",
			Cuerpo:   "Cuerpo B",
			Leida:    false,
			CreadoEn: time.Now().UTC(),
		}
		require.NoError(t, repo.Create(ctx, n))
	}

	// ListByUser for A — must return exactly A's 3, ordered DESC.
	rows, total, err := repo.ListByUser(ctx, userA, pagination.Params{Page: 1, Size: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total, "ListByUser for A must report total=3")
	assert.Len(t, rows, 3, "ListByUser for A must return exactly 3 rows")
	for _, r := range rows {
		assert.Equal(t, userA, r.UserID, "all returned notifications must belong to user A")
	}
	// Verify DESC order (each creado_en >= the next).
	for i := 0; i < len(rows)-1; i++ {
		assert.True(t, !rows[i].CreadoEn.Before(rows[i+1].CreadoEn),
			"results must be ordered by creado_en DESC")
	}
}

// ── TestNotificationRepository_CountUnread ────────────────────────────────────

func TestNotificationRepository_CountUnread(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	userA := seedUser(t, db)

	// Seed 2 unread + 1 read for A.
	for i := 0; i < 2; i++ {
		n := &domain.Notification{
			ID:       uuid.New().String(),
			UserID:   userA,
			Tipo:     domain.TipoCursoAprobado,
			Titulo:   "T",
			Cuerpo:   "C",
			Leida:    false,
			CreadoEn: time.Now().UTC(),
		}
		require.NoError(t, repo.Create(ctx, n))
	}
	readNotif := &domain.Notification{
		ID:       uuid.New().String(),
		UserID:   userA,
		Tipo:     domain.TipoCursoAprobado,
		Titulo:   "T",
		Cuerpo:   "C",
		Leida:    true,
		CreadoEn: time.Now().UTC(),
	}
	require.NoError(t, repo.Create(ctx, readNotif))

	count, err := repo.CountUnread(ctx, userA)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count, "CountUnread must return 2 for user A (2 unread)")

	// After MarkAllRead, count must be 0.
	require.NoError(t, repo.MarkAllRead(ctx, userA))
	count, err = repo.CountUnread(ctx, userA)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "CountUnread must return 0 after MarkAllRead")
}

// ── TestNotificationRepository_MarkRead_CallerScoped ────────────────────────

// TestNotificationRepository_MarkRead_CallerScoped is the adversarial test.
// User A marking user B's notification → ErrNotFound + B's row leida stays false.
func TestNotificationRepository_MarkRead_CallerScoped(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	userA := seedUser(t, db)
	userB := seedUser(t, db)

	// Seed 1 notification for B.
	bNotifID := uuid.New().String()
	bNotif := &domain.Notification{
		ID:       bNotifID,
		UserID:   userB,
		Tipo:     domain.TipoCursoAprobado,
		Titulo:   "T",
		Cuerpo:   "C",
		Leida:    false,
		CreadoEn: time.Now().UTC(),
	}
	require.NoError(t, repo.Create(ctx, bNotif))

	// A tries to mark B's notification → must return ErrNotFound.
	err := repo.MarkRead(ctx, bNotifID, userA)
	assert.ErrorIs(t, err, repository.ErrNotFound, "MarkRead cross-user must return ErrNotFound")

	// Verify B's row is still leida=false (non-vacuous security assertion).
	var leida bool
	dbErr := db.WithContext(ctx).Raw(
		`SELECT leida FROM notification WHERE id = ?`, bNotifID,
	).Scan(&leida).Error
	require.NoError(t, dbErr)
	assert.False(t, leida, "B's notification leida must still be false after A's cross-user MarkRead attempt")
}

// ── TestNotificationRepository_MarkAllRead_CallerScoped ──────────────────────

func TestNotificationRepository_MarkAllRead_CallerScoped(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	userA := seedUser(t, db)
	userB := seedUser(t, db)

	// Seed 2 unread for A, 1 unread for B.
	for i := 0; i < 2; i++ {
		n := &domain.Notification{
			ID:       uuid.New().String(),
			UserID:   userA,
			Tipo:     domain.TipoCursoAprobado,
			Titulo:   "T",
			Cuerpo:   "C",
			Leida:    false,
			CreadoEn: time.Now().UTC(),
		}
		require.NoError(t, repo.Create(ctx, n))
	}
	bNotifID := uuid.New().String()
	bNotif := &domain.Notification{
		ID:       bNotifID,
		UserID:   userB,
		Tipo:     domain.TipoCursoAprobado,
		Titulo:   "T",
		Cuerpo:   "C",
		Leida:    false,
		CreadoEn: time.Now().UTC(),
	}
	require.NoError(t, repo.Create(ctx, bNotif))

	// MarkAllRead for A.
	require.NoError(t, repo.MarkAllRead(ctx, userA))

	// A's unread count must be 0.
	countA, err := repo.CountUnread(ctx, userA)
	require.NoError(t, err)
	assert.Equal(t, int64(0), countA, "A's unread count must be 0 after MarkAllRead")

	// B's notification must still be leida=false.
	var bLeida bool
	dbErr := db.WithContext(ctx).Raw(
		`SELECT leida FROM notification WHERE id = ?`, bNotifID,
	).Scan(&bLeida).Error
	require.NoError(t, dbErr)
	assert.False(t, bLeida, "B's notification must remain leida=false after A's MarkAllRead")
}
