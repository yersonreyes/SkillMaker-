//go:build integration

// Package repository — integration tests for the certificates repository.
// Covers migration 0010: tables created, seed badges, idempotency, CRUD,
// Ranking, badge award idempotency.
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

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/repository"
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

func seedCourse(t *testing.T, db *gorm.DB, creadorID string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO course (id, creador_id, titulo, descripcion, estado)
		 VALUES (?, ?, ?, ?, 'aprobado')`,
		id, creadorID, "Test Course "+id[:8], "Desc",
	).Error
	require.NoError(t, err, "seedCourse: failed to insert course")
	return id
}

func seedCertificate(t *testing.T, repo repository.Repository, userID, courseID string) *domain.Certificate {
	t.Helper()
	cert := &domain.Certificate{
		ID:         uuid.New().String(),
		UserID:     userID,
		CourseID:   courseID,
		Codigo:     uuid.New().String()[:13],
		StorageKey: "certificates/" + uuid.New().String() + ".pdf",
		EmitidoEn:  time.Now().UTC(),
	}
	require.NoError(t, repo.Create(context.Background(), cert), "seedCertificate: failed")
	return cert
}

// ── TestMigration0010RoundTrip ─────────────────────────────────────────────────

// TestMigration0010RoundTrip verifies that migration 0010 creates certificate, badge, user_badge
// tables and seeds 3 badge rows. Also verifies the down migration removes all 0010 objects.
func TestMigration0010RoundTrip(t *testing.T) {
	db, m, teardown := testutil.SetupPostgresWithMigrate(t)
	defer func() { m.Close(); teardown() }()

	ctx := context.Background()

	// Verify certificate table exists after 0010 up.
	var tableCount int64
	err := db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_name = 'certificate'`,
	).Scan(&tableCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), tableCount, "certificate table must exist after 0010 up")

	// Verify badge table exists.
	err = db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_name = 'badge'`,
	).Scan(&tableCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), tableCount, "badge table must exist after 0010 up")

	// Verify user_badge table exists.
	err = db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_name = 'user_badge'`,
	).Scan(&tableCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), tableCount, "user_badge table must exist after 0010 up")

	// Verify seed badges = 3 rows.
	var badgeCount int64
	err = db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM badge`).Scan(&badgeCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(3), badgeCount, "badge seed must produce exactly 3 rows after 0010 up")

	// Verify idempotent re-seed (re-run INSERT ON CONFLICT DO NOTHING).
	err = db.WithContext(ctx).Exec(
		`INSERT INTO badge (nombre, descripcion, umbral) VALUES
		 ('Primer curso completado', 'Completaste tu primer curso', 1),
		 ('5 cursos completados', 'Completaste 5 cursos', 5),
		 ('10 cursos completados', 'Completaste 10 cursos', 10)
		 ON CONFLICT (nombre) DO NOTHING`,
	).Error
	require.NoError(t, err, "re-seeding must not fail")
	err = db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM badge`).Scan(&badgeCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(3), badgeCount, "re-seed must not create duplicates")

	// Roll back 6 steps — drops 0015+0014+0013+0012+0011+0010.
	// NOTE (course-structure-v2): migrations 0011+0012+0013 added; +3 → need -4.
	// NOTE (course-player-progress): migration 0014 added; +1 → need -5.
	// NOTE (notifications-inapp): migration 0015 added; +1 → need -6.
	require.NoError(t, m.Steps(-6), "m.Steps(-6) must roll back 0015+0014+0013+0012+0011+0010 without error")

	// Verify tables gone.
	err = db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_name IN ('certificate', 'badge', 'user_badge')`,
	).Scan(&tableCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), tableCount, "all 0010 tables must be dropped after 0010 down")
}

// ── TestCertificateRepository_CreateIdempotent ───────────────────────────────

// TestCertificateRepository_CreateIdempotent verifies that creating the same
// (user_id, course_id) pair twice produces exactly 1 row (UNIQUE constraint / DO NOTHING).
func TestCertificateRepository_CreateIdempotent(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creadorID := seedUser(t, db)
	studentID := seedUser(t, db)
	courseID := seedCourse(t, db, creadorID)

	// First create.
	cert1 := seedCertificate(t, repo, studentID, courseID)

	// Second create with same (user, course) — different ID/codigo but same constraint.
	cert2 := &domain.Certificate{
		ID:         uuid.New().String(),
		UserID:     studentID,
		CourseID:   courseID,
		Codigo:     uuid.New().String()[:13],
		StorageKey: "certificates/other.pdf",
		EmitidoEn:  time.Now().UTC(),
	}
	err := repo.Create(ctx, cert2)
	// The ON CONFLICT DO NOTHING means either no error OR the first row is returned.
	// Either way, there must be exactly 1 row.
	_ = err // idempotent create — may or may not return error depending on implementation

	var count int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM certificate WHERE user_id = ? AND course_id = ?`,
		studentID, courseID,
	).Scan(&count).Error)
	assert.Equal(t, int64(1), count, "second create must not produce a duplicate row (UNIQUE constraint)")
	_ = cert1
}

// ── TestCertificateRepository_GetByID ─────────────────────────────────────────

func TestCertificateRepository_GetByID(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creadorID := seedUser(t, db)
	studentID := seedUser(t, db)
	courseID := seedCourse(t, db, creadorID)
	cert := seedCertificate(t, repo, studentID, courseID)

	got, err := repo.GetByID(ctx, cert.ID)
	require.NoError(t, err)
	assert.Equal(t, cert.ID, got.ID)
	assert.Equal(t, studentID, got.UserID)
	assert.Equal(t, courseID, got.CourseID)
}

// ── TestCertificateRepository_GetByID_NotFound ───────────────────────────────

func TestCertificateRepository_GetByID_NotFound(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New().String())
	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrCertificateNotFound,
		"GetByID on missing cert must return ErrCertificateNotFound")
}

// ── TestCertificateRepository_ListByUser ──────────────────────────────────────

func TestCertificateRepository_ListByUser(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creadorID := seedUser(t, db)
	userA := seedUser(t, db)
	userB := seedUser(t, db)
	course1 := seedCourse(t, db, creadorID)
	course2 := seedCourse(t, db, creadorID)

	// userA has 2 certs, userB has 1.
	seedCertificate(t, repo, userA, course1)
	time.Sleep(2 * time.Millisecond) // ensure ordering
	seedCertificate(t, repo, userA, course2)
	seedCertificate(t, repo, userB, course1)

	certs, err := repo.ListByUser(ctx, userA)
	require.NoError(t, err)
	assert.Len(t, certs, 2, "userA must have 2 certs")
	for _, c := range certs {
		assert.Equal(t, userA, c.UserID, "all certs must belong to userA")
	}

	// Ordering: emitido_en DESC (newest first).
	if len(certs) == 2 {
		assert.True(t, !certs[0].EmitidoEn.Before(certs[1].EmitidoEn),
			"certs must be ordered emitido_en DESC")
	}
}

// ── TestCertificateRepository_CountByUser ────────────────────────────────────

func TestCertificateRepository_CountByUser(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creadorID := seedUser(t, db)
	studentID := seedUser(t, db)
	course1 := seedCourse(t, db, creadorID)
	course2 := seedCourse(t, db, creadorID)

	count0, err := repo.CountByUser(ctx, studentID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count0, "new user must have 0 certs")

	seedCertificate(t, repo, studentID, course1)
	count1, err := repo.CountByUser(ctx, studentID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count1)

	seedCertificate(t, repo, studentID, course2)
	count2, err := repo.CountByUser(ctx, studentID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count2)
}

// ── TestCertificateRepository_BadgesWorkflow ─────────────────────────────────

func TestCertificateRepository_BadgesWorkflow(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	studentID := seedUser(t, db)

	// Initially no badges earned.
	badges, err := repo.ListBadgesByUser(ctx, studentID)
	require.NoError(t, err)
	assert.Empty(t, badges, "new user must have no badges")

	// Fetch the "Primer curso completado" badge from DB (seeded by migration 0010).
	var badgeID string
	require.NoError(t, db.Raw(
		`SELECT id FROM badge WHERE nombre = 'Primer curso completado'`,
	).Scan(&badgeID).Error)
	require.NotEmpty(t, badgeID, "migration 0010 must seed 'Primer curso completado' badge")

	// Award once.
	require.NoError(t, repo.AwardBadge(ctx, studentID, badgeID))

	// Verify award.
	badges, err = repo.ListBadgesByUser(ctx, studentID)
	require.NoError(t, err)
	require.Len(t, badges, 1, "user must have 1 badge after AwardBadge")
	assert.Equal(t, "Primer curso completado", badges[0].Nombre)

	// Award again — idempotent (ON CONFLICT DO NOTHING via PK).
	require.NoError(t, repo.AwardBadge(ctx, studentID, badgeID))

	badges, err = repo.ListBadgesByUser(ctx, studentID)
	require.NoError(t, err)
	assert.Len(t, badges, 1, "second AwardBadge must be idempotent (PK conflict)")
}

// ── TestCertificateRepository_ListBadgesUpToThreshold ────────────────────────

func TestCertificateRepository_ListBadgesUpToThreshold(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	// Threshold 1 → only "Primer curso completado" (umbral=1).
	badges1, err := repo.ListBadgesUpToThreshold(ctx, 1)
	require.NoError(t, err)
	require.Len(t, badges1, 1, "threshold 1 must return 1 badge")
	assert.Equal(t, "Primer curso completado", badges1[0].Nombre)

	// Threshold 5 → "Primer" + "5 cursos".
	badges5, err := repo.ListBadgesUpToThreshold(ctx, 5)
	require.NoError(t, err)
	assert.Len(t, badges5, 2, "threshold 5 must return 2 badges")

	// Threshold 10 → all 3.
	badges10, err := repo.ListBadgesUpToThreshold(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, badges10, 3, "threshold 10 must return 3 badges")
}

// ── TestCertificateRepository_Ranking ────────────────────────────────────────

func TestCertificateRepository_Ranking(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creadorID := seedUser(t, db)

	// userD has 0 certs — must be excluded from ranking.
	userD := seedUser(t, db)
	_ = userD

	// userA has 3 certs.
	userA := seedUser(t, db)
	for i := 0; i < 3; i++ {
		c := seedCourse(t, db, creadorID)
		seedCertificate(t, repo, userA, c)
	}

	// userB has 1 cert.
	userB := seedUser(t, db)
	courseB := seedCourse(t, db, creadorID)
	seedCertificate(t, repo, userB, courseB)

	rows, err := repo.Ranking(ctx, 10)
	require.NoError(t, err)

	// Verify at least 2 users in ranking (userA and userB).
	require.GreaterOrEqual(t, len(rows), 2, "ranking must include at least 2 users")

	// Verify userD (0 certs) is excluded.
	for _, r := range rows {
		assert.NotEqual(t, userD, r.UserID, "user with 0 certs must be excluded from ranking")
		assert.Greater(t, r.Total, int64(0), "ranking must only include users with certs")
	}

	// Verify ordering (DESC by total).
	for i := 1; i < len(rows); i++ {
		assert.GreaterOrEqual(t, rows[i-1].Total, rows[i].Total,
			"ranking must be ordered DESC by total")
	}
}

// ── TestCertificateRepository_GetByUserCourse ────────────────────────────────

func TestCertificateRepository_GetByUserCourse(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creadorID := seedUser(t, db)
	studentID := seedUser(t, db)
	courseID := seedCourse(t, db, creadorID)

	// Not found before creating.
	_, err := repo.GetByUserCourse(ctx, studentID, courseID)
	require.Error(t, err)
	assert.ErrorIs(t, err, repository.ErrCertificateNotFound)

	// Create and then find.
	cert := seedCertificate(t, repo, studentID, courseID)
	got, err := repo.GetByUserCourse(ctx, studentID, courseID)
	require.NoError(t, err)
	assert.Equal(t, cert.ID, got.ID)
}
