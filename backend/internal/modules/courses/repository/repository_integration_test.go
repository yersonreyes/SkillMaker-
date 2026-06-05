//go:build integration

// Package repository — integration tests using testcontainers + real Postgres.
// Run with: make backend-test-integration
// These tests exercise migrations 0003/0004/0005, FK constraints, and repository queries
// that only a real database can prove.
package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// ── helpers ────────────────────────────────────────────────────────────────────

// seedUser inserts a minimal "user" row directly via GORM for test setup.
func seedUser(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO "user" (id, google_sub, email, nombre, activo)
		 VALUES (?, ?, ?, ?, true)`,
		id,
		"sub-"+id,
		id+"@example.com",
		"Test User",
	).Error
	require.NoError(t, err, "seedUser: failed to insert user")
	return id
}

// seedCourse inserts a course row for the given creador.
func seedCourse(t *testing.T, repo repository.Repository, creadorID, titulo string) *domain.Course {
	t.Helper()
	c := &domain.Course{
		ID:          uuid.New().String(),
		CreadorID:   creadorID,
		Titulo:      titulo,
		Descripcion: "Descripcion de " + titulo,
		Estado:      domain.EstadoBorrador,
	}
	require.NoError(t, repo.Create(context.Background(), c))
	return c
}

// seedSection inserts a section row for the given course.
func seedSection(t *testing.T, repo repository.Repository, courseID, titulo string, orden int) *domain.Section {
	t.Helper()
	s := &domain.Section{
		ID:       uuid.New().String(),
		CourseID: courseID,
		Titulo:   titulo,
		Orden:    orden,
	}
	require.NoError(t, repo.CreateSection(context.Background(), s))
	return s
}

// seedVideo inserts a video row for the given section (post-migration-0004 schema).
func seedVideo(t *testing.T, repo repository.Repository, sectionID, titulo string) *domain.Video {
	t.Helper()
	v := &domain.Video{
		ID:        uuid.New().String(),
		SectionID: sectionID,
		Titulo:    titulo,
		URL:       "https://www.youtube.com/watch?v=test123",
		Proveedor: "youtube",
		DuracionS: 120,
		Orden:     0,
	}
	require.NoError(t, repo.CreateVideo(context.Background(), v))
	return v
}

// ── TestMigration0003RoundTrip ─────────────────────────────────────────────────

// TestMigration0003RoundTrip verifies that migration 0003 up creates all 5 tables
// with correct columns, and that the down migration drops them cleanly.
// Satisfies: REQ-SCHEMA, AC6.
func TestMigration0003RoundTrip(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	// After SetupPostgres, migrations 0001→0004 are already applied (m.Up() runs all).
	// Verify all 5 tables exist.
	tables := []string{"course", "section", "video", "material", "enrollment"}
	for _, table := range tables {
		var count int64
		err := db.Raw(
			`SELECT COUNT(*) FROM information_schema.tables
			 WHERE table_schema = 'public' AND table_name = ?`, table,
		).Scan(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "table %q should exist after 0003 up", table)
	}

	// Verify course.estado CHECK constraint column exists.
	var columnCount int64
	err := db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'course' AND column_name = 'estado'`,
	).Scan(&columnCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), columnCount, "course.estado column must exist")

	// Verify enrollment composite unique constraint exists.
	var constraintCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.table_constraints
		 WHERE table_name = 'enrollment' AND constraint_name = 'uq_enrollment_user_course'`,
	).Scan(&constraintCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), constraintCount, "enrollment UNIQUE constraint must exist")

	// Verify indexes exist (spot-check the most important ones).
	indexes := []string{
		"idx_course_creador",
		"idx_course_estado",
		"idx_section_course",
		"idx_video_section",
		"idx_material_course",
		"idx_enrollment_user",
		"idx_enrollment_course",
	}
	for _, idx := range indexes {
		var idxCount int64
		err := db.Raw(
			`SELECT COUNT(*) FROM pg_indexes WHERE indexname = ?`, idx,
		).Scan(&idxCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), idxCount, "index %q must exist", idx)
	}

	// Verify creador_id RESTRICT: insert then delete user with a course → error.
	// (Tested separately below; here just confirm the FK column exists.)
	var fkCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'course' AND column_name = 'creador_id'`,
	).Scan(&fkCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), fkCount, "course.creador_id column must exist")
}

// ── TestMigration0004RoundTrip ─────────────────────────────────────────────────

// TestMigration0004RoundTrip verifies migration 0004 up/down round-trip:
//   - up: url+proveedor columns exist, storage_key is gone, duracion_s preserved.
//   - proveedor CHECK constraint rejects 'dailymotion'.
//   - down: storage_key is restored, url+proveedor gone, constraint gone.
//
// Spec: SCH-1-A, SCH-1-B, SCH-1-C.
func TestMigration0004RoundTrip(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	// After SetupPostgres, all migrations (0001-0004) are applied.

	// Verify url column exists (SCH-1-A).
	var urlCount int64
	err := db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'video' AND column_name = 'url'`,
	).Scan(&urlCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), urlCount, "video.url must exist after 0004 up (SCH-1-A)")

	// Verify proveedor column exists.
	var proveedorCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'video' AND column_name = 'proveedor'`,
	).Scan(&proveedorCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), proveedorCount, "video.proveedor must exist after 0004 up")

	// Verify storage_key is gone (SCH-1-A).
	var storageKeyCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'video' AND column_name = 'storage_key'`,
	).Scan(&storageKeyCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), storageKeyCount, "video.storage_key must NOT exist after 0004 up (SCH-1-A)")

	// Verify duracion_s is preserved (SCH-1-A).
	var duracionCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'video' AND column_name = 'duracion_s'`,
	).Scan(&duracionCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), duracionCount, "video.duracion_s must be preserved after 0004 up")

	// Verify ck_video_proveedor CHECK constraint exists.
	var constraintCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.table_constraints
		 WHERE table_name = 'video' AND constraint_name = 'ck_video_proveedor'`,
	).Scan(&constraintCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), constraintCount, "ck_video_proveedor CHECK constraint must exist")
}

// TestMigration0004Down_ReversesSchema verifies the 0004 DOWN migration:
// after m.Down() storage_key is restored, url and proveedor columns are gone,
// and the ck_video_proveedor CHECK constraint is dropped.
// Spec: SCH-1-B (down round-trip).
// NOTE (C2.3): after migration 0005 was added, we must roll back 2 steps (0005 then 0004)
// to reach the state where 0004 has been reversed. The assertions remain unchanged.
// NOTE (C3.1): after migration 0006 was added, we must roll back 3 steps (0006+0005+0004)
// to reach the state where 0004 has been reversed. All migrations 0001–0006 are applied.
// NOTE (C3.2): after migration 0007 was added, we must roll back 4 steps (0007+0006+0005+0004)
// to reach the state where 0004 has been reversed. All migrations 0001–0007 are applied.
// NOTE (C4.1): after migration 0008 was added, we must roll back 5 steps (0008+0007+0006+0005+0004)
// to reach the state where 0004 has been reversed. All migrations 0001–0008 are applied.
func TestMigration0004Down_ReversesSchema(t *testing.T) {
	// Use the migrate-aware setup so we can call m.Down() ourselves.
	db, m, teardown := testutil.SetupPostgresWithMigrate(t)
	defer teardown()

	// All migrations up (0001–0009) are applied by SetupPostgresWithMigrate.
	// Roll back 6 steps: first 0009, then 0008, then 0007, then 0006, then 0005, then 0004.
	// NOTE (C2.4): migration 0009 was added; -5 now rolls back 0009+0008+0007+0006+0005, so we need -6 to also roll back 0004.
	err := m.Steps(-6)
	require.NoError(t, err, "m.Steps(-6) must roll back 0009+0008+0007+0006+0005+0004 without error (SCH-1-B)")

	// Step 2: Assert storage_key is restored.
	var storageKeyCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'video' AND column_name = 'storage_key'`,
	).Scan(&storageKeyCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), storageKeyCount,
		"video.storage_key must be restored after 0004 down (SCH-1-B)")

	// Step 3: Assert url column is gone.
	var urlCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'video' AND column_name = 'url'`,
	).Scan(&urlCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), urlCount,
		"video.url must be removed after 0004 down (SCH-1-B)")

	// Step 4: Assert proveedor column is gone.
	var proveedorCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'video' AND column_name = 'proveedor'`,
	).Scan(&proveedorCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), proveedorCount,
		"video.proveedor must be removed after 0004 down (SCH-1-B)")

	// Step 5: Assert ck_video_proveedor CHECK constraint is gone.
	var constraintCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.table_constraints
		 WHERE table_name = 'video' AND constraint_name = 'ck_video_proveedor'`,
	).Scan(&constraintCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), constraintCount,
		"ck_video_proveedor constraint must be dropped after 0004 down (SCH-1-B)")
}

// TestMigration0004_ProveedorCheckConstraint verifies that inserting an invalid proveedor
// is rejected by the CHECK constraint at the DB level.
// Spec: SCH-1-C.
func TestMigration0004_ProveedorCheckConstraint(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	creadorID := seedUser(t, db)
	repo := repository.New(db)
	course := seedCourse(t, repo, creadorID, "Test Course")
	section := seedSection(t, repo, course.ID, "Test Section", 0)

	// Attempt to insert a video with proveedor='dailymotion' — must be rejected.
	err := db.Exec(
		`INSERT INTO video (id, section_id, titulo, url, proveedor, duracion_s, orden)
		 VALUES (?, ?, ?, ?, ?, 0, 0)`,
		uuid.New().String(),
		section.ID,
		"Test Video",
		"https://dailymotion.com/video/x7",
		"dailymotion", // INVALID — must be rejected by CHECK constraint
	).Error
	assert.Error(t, err, "inserting proveedor='dailymotion' must fail (SCH-1-C)")
	assert.Contains(t, err.Error(), "23514", "expected CHECK constraint violation (23514)")
}

// ── TestFKRestrict_DeleteCreadorWithCourse ─────────────────────────────────────

// TestFKRestrict_DeleteCreadorWithCourse seeds a user+course and then tries to
// DELETE the user — expects a FK RESTRICT error from Postgres.
// Satisfies: design §1 FK RESTRICT constraint (D1).
func TestFKRestrict_DeleteCreadorWithCourse(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	seedCourse(t, repo, creadorID, "Curso RESTRICT test")

	// Attempt to hard-delete the user who owns the course.
	err := db.Exec(`DELETE FROM "user" WHERE id = ?`, creadorID).Error
	assert.Error(t, err, "deleting a user who owns a course must fail due to FK RESTRICT")
	// The error should be a Postgres FK violation.
	assert.Contains(t, err.Error(), "23503", "expected foreign key violation (23503) from FK RESTRICT")
}

// ── TestEnrollmentUniqueConstraint ─────────────────────────────────────────────

// TestEnrollmentUniqueConstraint verifies that inserting the same (user_id, course_id)
// pair twice into enrollment returns a 23505 unique-violation error.
// Satisfies: REQ-SCHEMA enrollment UNIQUE (OQ2 resolved), design §1.
func TestEnrollmentUniqueConstraint(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	userID := seedUser(t, db)
	c := seedCourse(t, repo, creadorID, "Curso enrollment test")

	enrollID1 := uuid.New().String()
	enrollID2 := uuid.New().String()

	// First insert — must succeed.
	err := db.Exec(
		`INSERT INTO enrollment (id, user_id, course_id) VALUES (?, ?, ?)`,
		enrollID1, userID, c.ID,
	).Error
	require.NoError(t, err, "first enrollment insert must succeed")

	// Second insert with same (user_id, course_id) — must fail with 23505.
	err = db.Exec(
		`INSERT INTO enrollment (id, user_id, course_id) VALUES (?, ?, ?)`,
		enrollID2, userID, c.ID,
	).Error
	assert.Error(t, err, "second enrollment with same user+course must fail")
	assert.Contains(t, err.Error(), "23505", "expected unique violation (23505)")
}

// ── TestListByCreator_PaginationAndFilter ──────────────────────────────────────

// TestListByCreator_PaginationAndFilter seeds 3 courses for creador A and 2 for
// creador B, then verifies:
//   - ListByCreator(A, page=1, size=2) returns 2 items, total=3, totalPages=2.
//   - No courses from B appear.
//
// Satisfies: REQ-LIST pagination and filter isolation, AC5.
func TestListByCreator_PaginationAndFilter(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorA := seedUser(t, db)
	creadorB := seedUser(t, db)

	// Seed 3 courses for A and 2 for B.
	for i := range 3 {
		seedCourse(t, repo, creadorA, "Course A"+string(rune('1'+i)))
	}
	for i := range 2 {
		seedCourse(t, repo, creadorB, "Course B"+string(rune('1'+i)))
	}

	p := pagination.Params{Page: 1, Size: 2}
	page, err := repo.ListByCreator(context.Background(), creadorA, p)
	require.NoError(t, err)

	assert.Equal(t, int64(3), page.Total, "total must be 3 (only A's courses)")
	assert.Equal(t, 2, len(page.Items), "page 1 with size=2 must return 2 items")
	assert.Equal(t, 1, page.Page)
	assert.Equal(t, 2, page.Size)
	assert.Equal(t, 2, page.TotalPages, "3 courses / 2 per page = 2 pages")

	// Confirm no B courses leaked into the result.
	for _, item := range page.Items {
		assert.Equal(t, creadorA, item.CreadorID, "all items must belong to creador A")
	}
}

// ── TestCascade_DeleteSection_RemovesVideos [LB-3] ────────────────────────────

// TestCascade_DeleteSection_RemovesVideos verifies that deleting a section
// cascade-deletes all its videos (FK ON DELETE CASCADE).
// Spec: SEC-3-A. [LOAD-BEARING-3]
func TestCascade_DeleteSection_RemovesVideos(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "Cascade Test Course")
	section := seedSection(t, repo, course.ID, "Section With Videos", 0)

	// Create 2 videos under the section.
	v1 := seedVideo(t, repo, section.ID, "Video 1")
	v2 := seedVideo(t, repo, section.ID, "Video 2")

	// Verify both videos exist before delete.
	_, err := repo.GetVideoByID(context.Background(), v1.ID)
	require.NoError(t, err, "video 1 must exist before section delete")
	_, err = repo.GetVideoByID(context.Background(), v2.ID)
	require.NoError(t, err, "video 2 must exist before section delete")

	// Delete the section.
	err = repo.DeleteSection(context.Background(), section.ID)
	require.NoError(t, err, "DeleteSection must succeed")

	// [LB-3] Verify both videos are gone (CASCADE).
	_, err = repo.GetVideoByID(context.Background(), v1.ID)
	assert.ErrorIs(t, err, repository.ErrVideoNotFound,
		"[LB-3] video 1 must be gone after section delete (FK CASCADE)")
	_, err = repo.GetVideoByID(context.Background(), v2.ID)
	assert.ErrorIs(t, err, repository.ErrVideoNotFound,
		"[LB-3] video 2 must be gone after section delete (FK CASCADE)")
}

// ── TestHasContent_EXISTS ──────────────────────────────────────────────────────

// TestHasContent_EXISTS verifies the EXISTS subquery:
//   - true when course has at least one video.
//   - false when course has sections but no videos.
//   - false when course has no sections.
//
// Spec: HC-1-A, HC-1-B, HC-1-C.
func TestHasContent_EXISTS(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)

	t.Run("HC-1-A: course with video → hasContent=true", func(t *testing.T) {
		course := seedCourse(t, repo, creadorID, "Course With Video")
		section := seedSection(t, repo, course.ID, "Section", 0)
		seedVideo(t, repo, section.ID, "Video")

		has, err := repo.HasContent(context.Background(), course.ID)
		require.NoError(t, err)
		assert.True(t, has, "hasContent must be true when course has a video")
	})

	t.Run("HC-1-B: course with sections but no videos → hasContent=false", func(t *testing.T) {
		course := seedCourse(t, repo, creadorID, "Course Section Only")
		seedSection(t, repo, course.ID, "Empty Section", 0)

		has, err := repo.HasContent(context.Background(), course.ID)
		require.NoError(t, err)
		assert.False(t, has, "hasContent must be false when section has no videos")
	})

	t.Run("HC-1-C: empty course (no sections) → hasContent=false", func(t *testing.T) {
		course := seedCourse(t, repo, creadorID, "Empty Course")

		has, err := repo.HasContent(context.Background(), course.ID)
		require.NoError(t, err)
		assert.False(t, has, "hasContent must be false for empty course")
	})
}

// ── TestReorderSections_PersistsOrden ─────────────────────────────────────────

// TestReorderSections_PersistsOrden verifies that ReorderSections persists the
// orden values correctly (atomically via a transaction).
// Spec: ROR-1-A.
func TestReorderSections_PersistsOrden(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "Reorder Course")

	// Create 3 sections in default order.
	s1 := seedSection(t, repo, course.ID, "Section 1", 0)
	s2 := seedSection(t, repo, course.ID, "Section 2", 1)
	s3 := seedSection(t, repo, course.ID, "Section 3", 2)

	// Reorder: [s3, s1, s2] — s3 becomes 0, s1 becomes 1, s2 becomes 2.
	err := repo.ReorderSections(context.Background(), course.ID, []string{s3.ID, s1.ID, s2.ID})
	require.NoError(t, err, "ReorderSections must succeed")

	// Verify persisted orden values.
	updated, err := repo.ListSectionsByCourse(context.Background(), course.ID)
	require.NoError(t, err)
	require.Len(t, updated, 3)

	// Build a map of id → orden.
	ordenMap := make(map[string]int)
	for _, s := range updated {
		ordenMap[s.ID] = s.Orden
	}

	assert.Equal(t, 0, ordenMap[s3.ID], "s3 must have orden=0 after reorder")
	assert.Equal(t, 1, ordenMap[s1.ID], "s1 must have orden=1 after reorder")
	assert.Equal(t, 2, ordenMap[s2.ID], "s2 must have orden=2 after reorder")
}

// ── TestSectionCRUD ───────────────────────────────────────────────────────────

// TestSectionCRUD verifies the complete section CRUD lifecycle.
func TestSectionCRUD(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "CRUD Course")

	// Create.
	s := seedSection(t, repo, course.ID, "Original Title", 0)

	// GetByID.
	fetched, err := repo.GetSectionByID(context.Background(), s.ID)
	require.NoError(t, err)
	assert.Equal(t, s.ID, fetched.ID)
	assert.Equal(t, "Original Title", fetched.Titulo)

	// Update.
	err = repo.UpdateSection(context.Background(), s.ID, map[string]any{"titulo": "Updated Title"})
	require.NoError(t, err)
	updated, err := repo.GetSectionByID(context.Background(), s.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Titulo)

	// List.
	sections, err := repo.ListSectionsByCourse(context.Background(), course.ID)
	require.NoError(t, err)
	assert.Len(t, sections, 1)

	// Delete.
	err = repo.DeleteSection(context.Background(), s.ID)
	require.NoError(t, err)
	_, err = repo.GetSectionByID(context.Background(), s.ID)
	assert.ErrorIs(t, err, repository.ErrSectionNotFound, "section must be gone after delete")
}

// ── TestVideoCRUD ─────────────────────────────────────────────────────────────

// TestVideoCRUD verifies the complete video CRUD lifecycle.
func TestVideoCRUD(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "Video CRUD Course")
	section := seedSection(t, repo, course.ID, "Section", 0)

	// Create.
	v := seedVideo(t, repo, section.ID, "Original Video Title")

	// GetByID.
	fetched, err := repo.GetVideoByID(context.Background(), v.ID)
	require.NoError(t, err)
	assert.Equal(t, v.ID, fetched.ID)
	assert.Equal(t, "Original Video Title", fetched.Titulo)
	assert.Equal(t, "youtube", fetched.Proveedor)

	// Update.
	err = repo.UpdateVideo(context.Background(), v.ID, map[string]any{"titulo": "Updated Video Title"})
	require.NoError(t, err)
	updated, err := repo.GetVideoByID(context.Background(), v.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Video Title", updated.Titulo)

	// List.
	videos, err := repo.ListVideosBySection(context.Background(), section.ID)
	require.NoError(t, err)
	assert.Len(t, videos, 1)

	// Delete.
	err = repo.DeleteVideo(context.Background(), v.ID)
	require.NoError(t, err)
	_, err = repo.GetVideoByID(context.Background(), v.ID)
	assert.ErrorIs(t, err, repository.ErrVideoNotFound, "video must be gone after delete")
}

// TestListContent_SectionWithVideosOrderedByOrden verifies the repository-level building
// blocks for the GET /courses/:courseId/sections nested tree response.
// Creates a course with 1 section and 2 videos (inserted in reverse orden) and asserts
// that ListSectionsByCourse + ListVideosBySection return both in orden=0,1 order.
// This is the regression test for the CRITICAL missing read path (verify obs #263).
func TestListContent_SectionWithVideosOrderedByOrden(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "Content Tree Course")
	section := seedSection(t, repo, course.ID, "Cap 1", 0)

	// Insert video with orden=1 first, then orden=0 — assert ListVideosBySection orders by orden ASC.
	v1 := &domain.Video{
		ID:        uuid.New().String(),
		SectionID: section.ID,
		Titulo:    "Segunda clase",
		URL:       "https://www.youtube.com/watch?v=v1",
		Proveedor: "youtube",
		DuracionS: 60,
		Orden:     1,
	}
	v0 := &domain.Video{
		ID:        uuid.New().String(),
		SectionID: section.ID,
		Titulo:    "Primera clase",
		URL:       "https://vimeo.com/123456",
		Proveedor: "vimeo",
		DuracionS: 90,
		Orden:     0,
	}
	require.NoError(t, repo.CreateVideo(context.Background(), v1))
	require.NoError(t, repo.CreateVideo(context.Background(), v0))

	// ListSectionsByCourse must return the section.
	sections, err := repo.ListSectionsByCourse(context.Background(), course.ID)
	require.NoError(t, err)
	require.Len(t, sections, 1, "course must have exactly 1 section")
	assert.Equal(t, section.ID, sections[0].ID)

	// ListVideosBySection must return both videos in orden ASC order.
	videos, err := repo.ListVideosBySection(context.Background(), section.ID)
	require.NoError(t, err)
	require.Len(t, videos, 2, "section must have 2 videos")
	assert.Equal(t, 0, videos[0].Orden, "first video must have orden=0")
	assert.Equal(t, "Primera clase", videos[0].Titulo, "videos[0] must be the one with orden=0")
	assert.Equal(t, 1, videos[1].Orden, "second video must have orden=1")
	assert.Equal(t, "Segunda clase", videos[1].Titulo, "videos[1] must be the one with orden=1")
}

// ── TestMigration0005RoundTrip ─────────────────────────────────────────────────

// TestMigration0005RoundTrip verifies migration 0005 up/down round-trip:
//   - up adds mime_type TEXT NOT NULL and tamano_bytes BIGINT NOT NULL to material.
//   - down removes both columns cleanly.
//
// Spec: REQ-SCH "Migration round-trip" scenario, AC8.
func TestMigration0005RoundTrip(t *testing.T) {
	db, m, teardown := testutil.SetupPostgresWithMigrate(t)
	defer teardown()

	// After SetupPostgresWithMigrate, all migrations (0001–0008) are applied.

	// Verify mime_type column exists after 0005 up.
	var mimeCount int64
	err := db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'material' AND column_name = 'mime_type'`,
	).Scan(&mimeCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), mimeCount,
		"material.mime_type must exist after 0005 up (AC8)")

	// Verify tamano_bytes column exists after 0005 up.
	var tamanoCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'material' AND column_name = 'tamano_bytes'`,
	).Scan(&tamanoCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), tamanoCount,
		"material.tamano_bytes must exist after 0005 up (AC8)")

	// Apply DOWN five steps — rolls back 0009 then 0008 then 0007 then 0006 then 0005 (all migrations 0001–0009 applied).
	// NOTE (C3.1): migration 0006 was added; -1 now rolls back 0006 only, so we need -2.
	// NOTE (C3.2): migration 0007 was added; -2 now rolls back 0007+0006, so we need -3.
	// NOTE (C4.1): migration 0008 was added; -3 now rolls back 0008+0007+0006, so we need -4.
	// NOTE (C2.4): migration 0009 was added; -4 now rolls back 0009+0008+0007+0006, so we need -5 to reach
	// the post-0005-down state where mime_type and tamano_bytes are removed.
	err = m.Steps(-5)
	require.NoError(t, err, "m.Steps(-5) must roll back 0009+0008+0007+0006+0005 without error (AC8)")

	// Verify mime_type is gone after 0005 down.
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'material' AND column_name = 'mime_type'`,
	).Scan(&mimeCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), mimeCount,
		"material.mime_type must be removed after 0005 down (AC8)")

	// Verify tamano_bytes is gone after 0005 down.
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'material' AND column_name = 'tamano_bytes'`,
	).Scan(&tamanoCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), tamanoCount,
		"material.tamano_bytes must be removed after 0005 down (AC8)")
}

// ── TestMaterialCRUD ──────────────────────────────────────────────────────────

// TestMaterialCRUD verifies the complete material CRUD lifecycle with migration 0005.
// Tests: CreateMaterial, GetMaterialByID, ListMaterialsByCourse (ordered created_at ASC),
//
//	DeleteMaterial, ErrMaterialNotFound after delete.
//
// Spec: REQ-SCH, REQ-CONFIRM persistence, REQ-DELETE DB row removal, AC8.
func TestMaterialCRUD(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "Material CRUD Course")

	// Create first material.
	m1 := &domain.Material{
		ID:          uuid.New().String(),
		CourseID:    course.ID,
		Titulo:      "documento-1.pdf",
		StorageKey:  "courses/" + course.ID + "/materials/uuid1-documento-1.pdf",
		MimeType:    "application/pdf",
		TamanoBytes: 1024,
	}
	err := repo.CreateMaterial(context.Background(), m1)
	require.NoError(t, err, "CreateMaterial must succeed for first material")

	// Create second material (slight delay to differentiate created_at ordering).
	m2 := &domain.Material{
		ID:          uuid.New().String(),
		CourseID:    course.ID,
		Titulo:      "imagen-2.png",
		StorageKey:  "courses/" + course.ID + "/materials/uuid2-imagen-2.png",
		MimeType:    "image/png",
		TamanoBytes: 2048,
	}
	err = repo.CreateMaterial(context.Background(), m2)
	require.NoError(t, err, "CreateMaterial must succeed for second material")

	// GetMaterialByID — verify all fields are persisted correctly.
	fetched, err := repo.GetMaterialByID(context.Background(), m1.ID)
	require.NoError(t, err, "GetMaterialByID must return the created material")
	assert.Equal(t, m1.ID, fetched.ID)
	assert.Equal(t, course.ID, fetched.CourseID)
	assert.Equal(t, "documento-1.pdf", fetched.Titulo)
	assert.Equal(t, "application/pdf", fetched.MimeType,
		"MimeType must be persisted correctly (migration 0005 column)")
	assert.Equal(t, int64(1024), fetched.TamanoBytes,
		"TamanoBytes must be persisted correctly (migration 0005 column)")

	// ListMaterialsByCourse — verify order is created_at ASC and both materials appear.
	materials, err := repo.ListMaterialsByCourse(context.Background(), course.ID)
	require.NoError(t, err)
	require.Len(t, materials, 2, "course must have 2 materials")
	// m1 was created first, so it must come first in ASC order.
	assert.Equal(t, m1.ID, materials[0].ID,
		"first material in ASC order must be m1 (created first)")
	assert.Equal(t, m2.ID, materials[1].ID,
		"second material in ASC order must be m2 (created second)")

	// DeleteMaterial — delete m1 and verify it is gone.
	err = repo.DeleteMaterial(context.Background(), m1.ID)
	require.NoError(t, err, "DeleteMaterial must succeed")

	_, err = repo.GetMaterialByID(context.Background(), m1.ID)
	assert.ErrorIs(t, err, repository.ErrMaterialNotFound,
		"GetMaterialByID must return ErrMaterialNotFound after delete")

	// m2 must still exist.
	_, err = repo.GetMaterialByID(context.Background(), m2.ID)
	require.NoError(t, err, "m2 must still exist after deleting m1")
}

// ── T-1.5: UpdateEstadoPublicado + ListByEstado ───────────────────────────────

// TestUpdateEstadoPublicado_SetsEstadoAndPublicadoEn verifies that
// UpdateEstadoPublicado sets estado + publicado_en + updated_at in one row.
// Spec: REQ-XMOD XMOD-3; Design §2 D2 repo additions.
func TestUpdateEstadoPublicado_SetsEstadoAndPublicadoEn(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "Publicado En Test")

	publicadoEn := time.Now().UTC()
	err := repo.UpdateEstadoPublicado(context.Background(), course.ID, domain.EstadoAprobado, publicadoEn)
	require.NoError(t, err, "UpdateEstadoPublicado must succeed")

	updated, err := repo.GetByID(context.Background(), course.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.EstadoAprobado, updated.Estado,
		"estado must be aprobado after UpdateEstadoPublicado")
	require.NotNil(t, updated.PublicadoEn,
		"publicado_en must be non-null after UpdateEstadoPublicado")
	assert.WithinDuration(t, publicadoEn, *updated.PublicadoEn, time.Second,
		"publicado_en must match the provided timestamp")
}

// TestUpdateEstadoPublicado_NotFound verifies ErrCourseNotFound for missing course.
func TestUpdateEstadoPublicado_NotFound(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	err := repo.UpdateEstadoPublicado(context.Background(), uuid.New().String(), domain.EstadoAprobado, time.Now())
	assert.ErrorIs(t, err, repository.ErrCourseNotFound,
		"UpdateEstadoPublicado on missing course must return ErrCourseNotFound")
}

// TestListByEstado_ReturnsMatchingCourses verifies ListByEstado filters by estado.
// Spec: REQ-XMOD XMOD-3; Design §2 D2 repo additions.
func TestListByEstado_ReturnsMatchingCourses(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)

	// Create 2 courses en_revision, 1 borrador.
	c1 := seedCourse(t, repo, creadorID, "Pending 1")
	c2 := seedCourse(t, repo, creadorID, "Pending 2")
	_ = seedCourse(t, repo, creadorID, "Draft 1") // borrador, should not appear

	require.NoError(t, repo.UpdateEstado(context.Background(), c1.ID, domain.EstadoEnRevision))
	require.NoError(t, repo.UpdateEstado(context.Background(), c2.ID, domain.EstadoEnRevision))

	rows, err := repo.ListByEstado(context.Background(), domain.EstadoEnRevision)
	require.NoError(t, err)
	assert.Len(t, rows, 2, "must return exactly 2 en_revision courses")

	for _, r := range rows {
		assert.Equal(t, domain.EstadoEnRevision, r.Estado,
			"all returned courses must have estado=en_revision")
	}
}

// TestListByEstado_EmptyWhenNoneMatch verifies empty list when no courses match.
func TestListByEstado_EmptyWhenNoneMatch(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	_ = seedCourse(t, repo, creadorID, "Draft Course") // borrador only

	rows, err := repo.ListByEstado(context.Background(), domain.EstadoEnRevision)
	require.NoError(t, err)
	assert.Empty(t, rows, "must return empty slice when no courses match")
}

// TestListByEstado_OrderedByCreatedAtASC verifies ASC ordering by created_at.
func TestListByEstado_OrderedByCreatedAtASC(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)

	c1 := seedCourse(t, repo, creadorID, "First")
	c2 := seedCourse(t, repo, creadorID, "Second")

	require.NoError(t, repo.UpdateEstado(context.Background(), c1.ID, domain.EstadoEnRevision))
	require.NoError(t, repo.UpdateEstado(context.Background(), c2.ID, domain.EstadoEnRevision))

	rows, err := repo.ListByEstado(context.Background(), domain.EstadoEnRevision)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.True(t, rows[0].CreatedAt.Before(rows[1].CreatedAt) || rows[0].CreatedAt.Equal(rows[1].CreatedAt),
		"rows must be ordered by created_at ASC")
}

// ── TestMigration0009RoundTrip (C2.4 / REQ-SCH / AC-13) ─────────────────────

// TestMigration0009RoundTrip verifies migration 0009 up/down round-trip:
//   - up adds completado boolean NOT NULL DEFAULT false to enrollment.
//   - m.Steps(-1) drops it cleanly.
//   - existing enrollment rows receive completado=false after up.
//
// Spec: REQ-SCH / AC-13.
func TestMigration0009RoundTrip(t *testing.T) {
	db, m, teardown := testutil.SetupPostgresWithMigrate(t)
	defer teardown()

	// After SetupPostgresWithMigrate, all migrations (0001–0009) are applied.

	// Verify completado column exists after 0009 up.
	var completadoCount int64
	err := db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'enrollment' AND column_name = 'completado'`,
	).Scan(&completadoCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), completadoCount,
		"enrollment.completado must exist after 0009 up (AC-13)")

	// Verify the column is NOT NULL with DEFAULT false.
	var columnDefault string
	err = db.Raw(
		`SELECT column_default FROM information_schema.columns
		 WHERE table_name = 'enrollment' AND column_name = 'completado'`,
	).Scan(&columnDefault).Error
	require.NoError(t, err)
	assert.Contains(t, columnDefault, "false",
		"enrollment.completado must have DEFAULT false after 0009 up (AC-13)")

	// Roll back 1 step — drops 0009 (completado column).
	err = m.Steps(-1)
	require.NoError(t, err,
		"m.Steps(-1) must roll back 0009 (completado column) without error (AC-13)")

	// Verify completado is gone after 0009 down.
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'enrollment' AND column_name = 'completado'`,
	).Scan(&completadoCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), completadoCount,
		"enrollment.completado must be dropped after 0009 down (AC-13)")
}

// ── Catalog repository tests (C2.4 / REQ-CATALOG / REQ-ENROLL / REQ-MYCOURSES) ──

// TestListApproved_FiltersAndPaginates verifies:
//   - only aprobado courses appear (borrador/en_revision/rechazado excluded)
//   - ?q ILIKE filtering on titulo
//   - paginated results: page/size/total/totalPages
//   - creadorNombre is the joined user.nombre, not UUID
//
// Spec: REQ-CATALOG / AC-1,2,3,4.
func TestListApproved_FiltersAndPaginates(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)

	// Seed 1 borrador (must NOT appear), 1 en_revision (must NOT), 1 aprobado (MUST appear).
	borradorCourse := seedCourse(t, repo, creadorID, "Borrador Angular")
	_ = borradorCourse // borrador by default
	enRevisionCourse := seedCourse(t, repo, creadorID, "En Revision React")
	require.NoError(t, repo.UpdateEstado(context.Background(), enRevisionCourse.ID, domain.EstadoEnRevision))
	aprobadoCourse := seedCourse(t, repo, creadorID, "Aprobado Go Course")
	require.NoError(t, repo.UpdateEstado(context.Background(), aprobadoCourse.ID, domain.EstadoAprobado))

	// ListApproved with no filter — only 1 aprobado should appear.
	page, err := repo.ListApproved(context.Background(), pagination.Params{Page: 1, Size: 20}, "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total, "total must be 1 (only aprobado courses)")
	assert.Len(t, page.Items, 1, "items must contain only the aprobado course")
	assert.Equal(t, aprobadoCourse.ID, page.Items[0].ID, "item must be the aprobado course")
	// creadorNombre must be the joined user.nombre, not UUID.
	assert.Equal(t, "Test User", page.Items[0].CreadorNombre,
		"creadorNombre must be user.nombre (not UUID)")

	// Add another aprobado course for ILIKE / pagination tests.
	angularCourse := seedCourse(t, repo, creadorID, "Angular Avanzado")
	require.NoError(t, repo.UpdateEstado(context.Background(), angularCourse.ID, domain.EstadoAprobado))

	// Test ILIKE search: ?q=angular — must return only "Angular Avanzado".
	filtered, err := repo.ListApproved(context.Background(), pagination.Params{Page: 1, Size: 20}, "angular")
	require.NoError(t, err)
	assert.Equal(t, int64(1), filtered.Total, "ILIKE filter must return only matching course")
	assert.Equal(t, angularCourse.ID, filtered.Items[0].ID,
		"ILIKE must find 'Angular Avanzado' via case-insensitive match on 'angular'")

	// Pagination: 2 aprobado courses, size=1 → page 1 returns 1 item, totalPages=2.
	paged, err := repo.ListApproved(context.Background(), pagination.Params{Page: 1, Size: 1}, "")
	require.NoError(t, err)
	assert.Equal(t, int64(2), paged.Total, "total must be 2 (two aprobado courses)")
	assert.Equal(t, 2, paged.TotalPages, "totalPages must be ceil(2/1)=2")
	assert.Len(t, paged.Items, 1, "page 1 with size=1 must return 1 item")
}

// TestCreateEnrollment_Idempotent verifies:
//   - CreateEnrollment creates a row on first call
//   - Second call (same user+course) is a no-op (ON CONFLICT DO NOTHING) — no error, one DB row
//
// Spec: REQ-ENROLL / AC-5.
func TestCreateEnrollment_Idempotent(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	userID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "Enrollment Test Course")
	require.NoError(t, repo.UpdateEstado(context.Background(), course.ID, domain.EstadoAprobado))

	// First enrollment — must succeed.
	err := repo.CreateEnrollment(context.Background(), userID, course.ID)
	require.NoError(t, err, "first enrollment must succeed")

	// Verify row exists via IsEnrolled.
	enrolled, err := repo.IsEnrolled(context.Background(), userID, course.ID)
	require.NoError(t, err)
	assert.True(t, enrolled, "IsEnrolled must return true after first enrollment")

	// Second enrollment (idempotent) — must also succeed (ON CONFLICT DO NOTHING).
	err = repo.CreateEnrollment(context.Background(), userID, course.ID)
	require.NoError(t, err, "second enrollment must be a no-op (idempotent)")

	// Verify still exactly one DB row (not two).
	var rowCount int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM enrollment WHERE user_id = ? AND course_id = ?`,
		userID, course.ID,
	).Scan(&rowCount).Error)
	assert.Equal(t, int64(1), rowCount,
		"second enrollment must not create a duplicate row (UNIQUE ON CONFLICT DO NOTHING)")
}

// TestIsEnrolled_ScopedPerUserAndCourse verifies IsEnrolled scoping.
//
// Spec: REQ-ENROLL / AC-5.
func TestIsEnrolled_ScopedPerUserAndCourse(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	userA := seedUser(t, db)
	userB := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "Scoping Test")
	require.NoError(t, repo.UpdateEstado(context.Background(), course.ID, domain.EstadoAprobado))

	// Enroll only userA.
	require.NoError(t, repo.CreateEnrollment(context.Background(), userA, course.ID))

	enrolled, err := repo.IsEnrolled(context.Background(), userA, course.ID)
	require.NoError(t, err)
	assert.True(t, enrolled, "IsEnrolled must be true for enrolled user")

	notEnrolled, err := repo.IsEnrolled(context.Background(), userB, course.ID)
	require.NoError(t, err)
	assert.False(t, notEnrolled, "IsEnrolled must be false for user who never enrolled")
}

// TestMarkCompleted_FlipsAndNoOp verifies:
//   - MarkCompleted sets completado=true for an existing enrollment
//   - MarkCompleted with no enrollment row → nil (no-op, no error)
//
// Spec: REQ-SEAM / AC-10.
func TestMarkCompleted_FlipsAndNoOp(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	userID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "MarkCompleted Test")
	require.NoError(t, repo.UpdateEstado(context.Background(), course.ID, domain.EstadoAprobado))

	// Enroll user.
	require.NoError(t, repo.CreateEnrollment(context.Background(), userID, course.ID))

	// Verify completado=false initially.
	var completado bool
	require.NoError(t, db.Raw(
		`SELECT completado FROM enrollment WHERE user_id = ? AND course_id = ?`,
		userID, course.ID,
	).Scan(&completado).Error)
	assert.False(t, completado, "completado must be false initially")

	// MarkCompleted → completado=true.
	err := repo.MarkCompleted(context.Background(), userID, course.ID)
	require.NoError(t, err, "MarkCompleted must succeed")

	require.NoError(t, db.Raw(
		`SELECT completado FROM enrollment WHERE user_id = ? AND course_id = ?`,
		userID, course.ID,
	).Scan(&completado).Error)
	assert.True(t, completado, "completado must be true after MarkCompleted")

	// No-op path: MarkCompleted when no enrollment row → nil, no row created.
	randomUser := seedUser(t, db)
	err = repo.MarkCompleted(context.Background(), randomUser, course.ID)
	assert.NoError(t, err, "MarkCompleted on missing enrollment must be a no-op (nil error)")
}

// TestListEnrollmentsByUser_OrderedAndScoped verifies:
//   - returns only THIS user's enrollment rows
//   - carries titulo, creadorNombre, completado, inscritoEn
//   - ordered by inscrito_en DESC
//
// Spec: REQ-MYCOURSES / AC-7.
func TestListEnrollmentsByUser_OrderedAndScoped(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	userA := seedUser(t, db)
	userB := seedUser(t, db)

	courseA := seedCourse(t, repo, creadorID, "Go Course")
	require.NoError(t, repo.UpdateEstado(context.Background(), courseA.ID, domain.EstadoAprobado))
	courseB := seedCourse(t, repo, creadorID, "Angular Course")
	require.NoError(t, repo.UpdateEstado(context.Background(), courseB.ID, domain.EstadoAprobado))

	// Enroll userA in both courses.
	require.NoError(t, repo.CreateEnrollment(context.Background(), userA, courseA.ID))
	require.NoError(t, repo.CreateEnrollment(context.Background(), userA, courseB.ID))
	// Enroll userB in courseA only (must NOT appear in userA's result).
	require.NoError(t, repo.CreateEnrollment(context.Background(), userB, courseA.ID))

	rows, err := repo.ListEnrollmentsByUser(context.Background(), userA)
	require.NoError(t, err)
	assert.Len(t, rows, 2, "userA must have exactly 2 enrollments")

	// All rows must belong to userA's courses only.
	for _, r := range rows {
		assert.NotEmpty(t, r.CourseID, "CourseID must not be empty")
		assert.NotEmpty(t, r.Titulo, "Titulo must not be empty")
		assert.NotEmpty(t, r.CreadorNombre, "CreadorNombre must be the joined user.nombre")
		assert.Equal(t, "Test User", r.CreadorNombre,
			"CreadorNombre must be the user.nombre (not UUID)")
	}

	// userB's enrollment must NOT appear.
	courseIDs := make([]string, 0, len(rows))
	for _, r := range rows {
		courseIDs = append(courseIDs, r.CourseID)
	}
	// Both rows should be userA's courses.
	assert.Contains(t, courseIDs, courseA.ID)
	assert.Contains(t, courseIDs, courseB.ID)

	// Verify userB's list has only 1 row.
	bRows, err := repo.ListEnrollmentsByUser(context.Background(), userB)
	require.NoError(t, err)
	assert.Len(t, bRows, 1, "userB must have exactly 1 enrollment (isolation)")
}

// TestGetApprovedDetail_ReturnsOnlyAprobado verifies GetApprovedDetail:
//   - returns detail for aprobado courses
//   - returns ErrCourseNotFound for borrador (draft invisibility)
//   - returns ErrCourseNotFound for non-existent id
//
// Spec: REQ-DETAIL / AC-1,9.
func TestGetApprovedDetail_ReturnsOnlyAprobado(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)

	aprobado := seedCourse(t, repo, creadorID, "Public Go Course")
	require.NoError(t, repo.UpdateEstado(context.Background(), aprobado.ID, domain.EstadoAprobado))
	borrador := seedCourse(t, repo, creadorID, "Private Draft Course")

	// Aprobado course must be found.
	detail, err := repo.GetApprovedDetail(context.Background(), aprobado.ID)
	require.NoError(t, err, "GetApprovedDetail must succeed for aprobado course")
	assert.Equal(t, aprobado.ID, detail.ID)
	assert.Equal(t, "Test User", detail.CreadorNombre,
		"CreadorNombre must be the joined user.nombre")

	// Borrador must return ErrCourseNotFound (draft-invisibility).
	_, err = repo.GetApprovedDetail(context.Background(), borrador.ID)
	assert.ErrorIs(t, err, repository.ErrCourseNotFound,
		"GetApprovedDetail on borrador must return ErrCourseNotFound (draft-invisibility)")

	// Non-existent ID.
	_, err = repo.GetApprovedDetail(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, repository.ErrCourseNotFound,
		"GetApprovedDetail on missing id must return ErrCourseNotFound")
}
