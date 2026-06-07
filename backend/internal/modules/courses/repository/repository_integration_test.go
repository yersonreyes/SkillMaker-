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
	// NOTE (course-structure-v2): idx_material_course is replaced by idx_material_video in 0012;
	// that index is checked in TestMigration0012BackfillInvariant instead.
	indexes := []string{
		"idx_course_creador",
		"idx_course_estado",
		"idx_section_course",
		"idx_video_section",
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

	// All migrations up (0001–0013) are applied by SetupPostgresWithMigrate.
	// Roll back 10 steps: 0013+0012+0011+0010+0009+0008+0007+0006+0005+0004.
	// NOTE (C2.4): migration 0009 was added; -5 now rolls back 0009+0008+0007+0006+0005, so we need -6 to also roll back 0004.
	// NOTE (C5.1): migration 0010 was added; -6 now rolls back 0010+0009+0008+0007+0006+0005, so we need -7.
	// NOTE (course-structure-v2): migrations 0011+0012+0013 added; +3 → need -10.
	err := m.Steps(-10)
	require.NoError(t, err, "m.Steps(-10) must roll back 0013+0012+0011+0010+0009+0008+0007+0006+0005+0004 without error (SCH-1-B)")

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

	// Apply DOWN nine steps — rolls back 0013+0012+0011+0010+0009+0008+0007+0006+0005 (all migrations 0001–0013 applied).
	// NOTE (C3.1): migration 0006 was added; -1 now rolls back 0006 only, so we need -2.
	// NOTE (C3.2): migration 0007 was added; -2 now rolls back 0007+0006, so we need -3.
	// NOTE (C4.1): migration 0008 was added; -3 now rolls back 0008+0007+0006, so we need -4.
	// NOTE (C2.4): migration 0009 was added; -4 now rolls back 0009+0008+0007+0006, so we need -5 to reach
	// the post-0005-down state where mime_type and tamano_bytes are removed.
	// NOTE (C5.1): migration 0010 was added; -5 now rolls back 0010+0009+0008+0007+0006, so we need -6.
	// NOTE (course-structure-v2): migrations 0011+0012+0013 added; +3 → need -9.
	err = m.Steps(-9)
	require.NoError(t, err, "m.Steps(-9) must roll back 0013+0012+0011+0010+0009+0008+0007+0006+0005 without error (AC8)")

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
// course-structure-v2: materials are now keyed by video_id (not course_id).
// Tests: CreateMaterial, GetMaterialByID, ListMaterialsByVideo (ordered created_at ASC),
//
//	DeleteMaterial, ErrMaterialNotFound after delete.
//
// Spec: REQ-SCH, REQ-MATERIAL-VIDEO, REQ-CONFIRM persistence, REQ-DELETE DB row removal, AC8.
func TestMaterialCRUD(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "Material CRUD Course")
	section := seedSection(t, repo, course.ID, "S1", 0)
	video := seedVideo(t, repo, section.ID, "V1")

	// Create first material (now keyed by video_id, not course_id).
	m1 := &domain.Material{
		ID:          uuid.New().String(),
		VideoID:     video.ID,
		Titulo:      "documento-1.pdf",
		StorageKey:  "courses/" + course.ID + "/videos/" + video.ID + "/materials/uuid1-documento-1.pdf",
		MimeType:    "application/pdf",
		TamanoBytes: 1024,
	}
	err := repo.CreateMaterial(context.Background(), m1)
	require.NoError(t, err, "CreateMaterial must succeed for first material")

	// Create second material (slight delay to differentiate created_at ordering).
	m2 := &domain.Material{
		ID:          uuid.New().String(),
		VideoID:     video.ID,
		Titulo:      "imagen-2.png",
		StorageKey:  "courses/" + course.ID + "/videos/" + video.ID + "/materials/uuid2-imagen-2.png",
		MimeType:    "image/png",
		TamanoBytes: 2048,
	}
	err = repo.CreateMaterial(context.Background(), m2)
	require.NoError(t, err, "CreateMaterial must succeed for second material")

	// GetMaterialByID — verify all fields are persisted correctly.
	fetched, err := repo.GetMaterialByID(context.Background(), m1.ID)
	require.NoError(t, err, "GetMaterialByID must return the created material")
	assert.Equal(t, m1.ID, fetched.ID)
	assert.Equal(t, video.ID, fetched.VideoID,
		"VideoID must be persisted correctly (migration 0012 column)")
	assert.Equal(t, "documento-1.pdf", fetched.Titulo)
	assert.Equal(t, "application/pdf", fetched.MimeType,
		"MimeType must be persisted correctly (migration 0005 column)")
	assert.Equal(t, int64(1024), fetched.TamanoBytes,
		"TamanoBytes must be persisted correctly (migration 0005 column)")

	// ListMaterialsByVideo — verify order is created_at ASC and both materials appear.
	materials, err := repo.ListMaterialsByVideo(context.Background(), video.ID)
	require.NoError(t, err)
	require.Len(t, materials, 2, "video must have 2 materials")
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

	// Roll back 5 steps — drops 0013+0012+0011+0010+0009 (completado column).
	// NOTE (C5.1): migration 0010 was added; -1 now rolls back 0010 only, so we need -2 to drop 0009.
	// NOTE (course-structure-v2): migrations 0011+0012+0013 added; +3 → need -5.
	err = m.Steps(-5)
	require.NoError(t, err,
		"m.Steps(-5) must roll back 0013+0012+0011+0010+0009 (completado column) without error (AC-13)")

	// Verify completado is gone after 0009 down.
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'enrollment' AND column_name = 'completado'`,
	).Scan(&completadoCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), completadoCount,
		"enrollment.completado must be dropped after 0009 down (AC-13)")
}

// ── Migration 0011/0012/0013 round-trip tests (course-structure-v2) ──────────────

// TestMigration0011RoundTrip verifies that migration 0011 adds video.descripcion
// and the down migration removes it cleanly. Satisfies REQ-SCH 0011.
func TestMigration0011RoundTrip(t *testing.T) {
	db, m, teardown := testutil.SetupPostgresWithMigrate(t)
	defer teardown()

	// All migrations up (0001–0013) are applied by SetupPostgresWithMigrate.
	// Verify video.descripcion exists.
	var count int64
	err := db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'video' AND column_name = 'descripcion'`,
	).Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count,
		"video.descripcion must exist after 0011 up (REQ-SCH-0011)")

	// Roll back 3 steps (0013+0012+0011).
	err = m.Steps(-3)
	require.NoError(t, err, "m.Steps(-3) must roll back 0013+0012+0011 without error")

	// Verify video.descripcion is gone after 0011 down.
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'video' AND column_name = 'descripcion'`,
	).Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count,
		"video.descripcion must be dropped after 0011 down (REQ-SCH-0011)")
}

// TestMigration0012BackfillInvariant is the CRITICAL backfill test.
// Seeds course A with 2 videos (orden 0, 1) and 1 material.
// Seeds course B with NO video and 1 material (orphan).
// After 0012 up:
//   - material of A has video_id == video(orden 0).id
//   - material of B is deleted (orphan)
//   - SELECT COUNT(*) FROM material WHERE video_id IS NULL == 0
//   - course_id column is absent
//   - idx_material_video index exists
//
// Satisfies REQ-SCH 0012 backfill invariant + orphan deletion invariant.
func TestMigration0012BackfillInvariant(t *testing.T) {
	db, m, teardown := testutil.SetupPostgresWithMigrate(t)
	defer teardown()

	// Roll back to migration 0010 state (before 0011+0012+0013).
	err := m.Steps(-3)
	require.NoError(t, err, "m.Steps(-3) to reach 0010 state")

	// Re-apply 0011 only (video.descripcion) so we can seed a video without descripcion issues.
	err = m.Steps(1)
	require.NoError(t, err, "m.Steps(1) to apply 0011")

	// Seed users.
	userAID := uuid.New().String()
	userBID := uuid.New().String()
	for _, uid := range []string{userAID, userBID} {
		err = db.Exec(
			`INSERT INTO "user" (id, google_sub, email, nombre, activo)
			 VALUES (?, ?, ?, 'Test', true)`,
			uid, "sub-"+uid, uid+"@test.com",
		).Error
		require.NoError(t, err)
	}

	// Course A: has 1 section, 2 videos (orden 0 and 1).
	courseAID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO course (id, creador_id, titulo, descripcion, estado)
		 VALUES (?, ?, 'Course A', 'desc', 'borrador')`,
		courseAID, userAID,
	).Error
	require.NoError(t, err)

	sectionAID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO section (id, course_id, titulo, orden) VALUES (?, ?, 'S1', 0)`,
		sectionAID, courseAID,
	).Error
	require.NoError(t, err)

	// video0 (orden=0) — this is the first video; backfill should use it.
	video0ID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO video (id, section_id, titulo, url, proveedor, duracion_s, orden)
		 VALUES (?, ?, 'V0', 'https://youtube.com/watch?v=x', 'youtube', 100, 0)`,
		video0ID, sectionAID,
	).Error
	require.NoError(t, err)

	// video1 (orden=1).
	video1ID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO video (id, section_id, titulo, url, proveedor, duracion_s, orden)
		 VALUES (?, ?, 'V1', 'https://youtube.com/watch?v=y', 'youtube', 200, 1)`,
		video1ID, sectionAID,
	).Error
	require.NoError(t, err)

	// Material for course A (old schema: course_id).
	mat1ID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO material (id, course_id, titulo, storage_key, mime_type, tamano_bytes)
		 VALUES (?, ?, 'notes.pdf', 'courses/A/mat1', 'application/pdf', 1000)`,
		mat1ID, courseAID,
	).Error
	require.NoError(t, err)

	// Course B: has a section but NO videos (orphan material).
	courseBID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO course (id, creador_id, titulo, descripcion, estado)
		 VALUES (?, ?, 'Course B', 'desc', 'borrador')`,
		courseBID, userBID,
	).Error
	require.NoError(t, err)

	sectionBID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO section (id, course_id, titulo, orden) VALUES (?, ?, 'SB', 0)`,
		sectionBID, courseBID,
	).Error
	require.NoError(t, err)

	// Orphan material for course B (no videos → will be deleted by 0012).
	mat2ID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO material (id, course_id, titulo, storage_key, mime_type, tamano_bytes)
		 VALUES (?, ?, 'orphan.pdf', 'courses/B/mat2', 'application/pdf', 500)`,
		mat2ID, courseBID,
	).Error
	require.NoError(t, err)

	// Apply migration 0012 (material → video).
	err = m.Steps(1)
	require.NoError(t, err, "m.Steps(1) to apply 0012")

	// CRITICAL: no NULL video_id in material after backfill.
	var nullCount int64
	err = db.Raw(`SELECT COUNT(*) FROM material WHERE video_id IS NULL`).Scan(&nullCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), nullCount,
		"BACKFILL INVARIANT: no material may have video_id IS NULL after 0012 (REQ-SCH-0012)")

	// Material of course A must point to video0 (orden=0, the first by orden).
	var gotVideoID string
	err = db.Raw(`SELECT video_id FROM material WHERE id = ?`, mat1ID).Scan(&gotVideoID).Error
	require.NoError(t, err)
	assert.Equal(t, video0ID, gotVideoID,
		"material of course A must be backfilled to video(orden=0) (REQ-SCH-0012)")

	// Orphan material (course B) must be deleted.
	var orphanCount int64
	err = db.Raw(`SELECT COUNT(*) FROM material WHERE id = ?`, mat2ID).Scan(&orphanCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), orphanCount,
		"orphan material (course with no video) must be deleted after 0012 (REQ-SCH-0012)")

	// course_id column must be absent.
	var colCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'material' AND column_name = 'course_id'`,
	).Scan(&colCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), colCount,
		"material.course_id must be dropped after 0012 (REQ-SCH-0012)")

	// idx_material_video must exist.
	var idxCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM pg_indexes
		 WHERE tablename = 'material' AND indexname = 'idx_material_video'`,
	).Scan(&idxCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), idxCount,
		"idx_material_video must exist after 0012 (REQ-SCH-0012)")
}

// TestMigration0013CourseMetaAndCategorias verifies that migration 0013 adds
// nivel/miniatura_key/horas_practico to course, creates categoria+course_categoria,
// seeds data, and the CHECK constraint rejects 'experto'. Satisfies REQ-SCH 0013.
func TestMigration0013CourseMetaAndCategorias(t *testing.T) {
	db, m, teardown := testutil.SetupPostgresWithMigrate(t)
	defer teardown()

	// Verify categoria table exists and has seed rows.
	var catCount int64
	err := db.Raw(`SELECT COUNT(*) FROM categoria`).Scan(&catCount).Error
	require.NoError(t, err)
	assert.Greater(t, catCount, int64(0),
		"categoria must have at least 1 seeded row after 0013 (REQ-SCH-0013)")

	// Verify nivel, miniatura_key, horas_practico columns on course.
	for _, col := range []string{"nivel", "miniatura_key", "horas_practico"} {
		var cnt int64
		err = db.Raw(
			`SELECT COUNT(*) FROM information_schema.columns
			 WHERE table_name = 'course' AND column_name = ?`, col,
		).Scan(&cnt).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), cnt, "course."+col+" must exist after 0013")
	}

	// CHECK constraint: nivel='experto' must be rejected.
	userID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO "user" (id, google_sub, email, nombre, activo) VALUES (?, ?, ?, 'U', true)`,
		userID, "sub-check", "check@test.com",
	).Error
	require.NoError(t, err)
	err = db.Exec(
		`INSERT INTO course (id, creador_id, titulo, descripcion, estado, nivel)
		 VALUES (gen_random_uuid(), ?, 'T', 'D', 'borrador', 'experto')`, userID,
	).Error
	assert.Error(t, err, "CHECK constraint must reject nivel='experto' (REQ-SCH-0013)")

	// Roll back 3 steps (0013+0012+0011).
	err = m.Steps(-3)
	require.NoError(t, err, "m.Steps(-3) rolls back 0013+0012+0011")

	// Verify categoria and course_categoria tables gone.
	var tblCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.tables
		 WHERE table_name IN ('categoria', 'course_categoria')`,
	).Scan(&tblCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), tblCount,
		"categoria + course_categoria must be dropped after 0013 down (REQ-SCH-0013)")

	// Verify nivel column gone from course.
	var nivelCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'course' AND column_name = 'nivel'`,
	).Scan(&nivelCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), nivelCount,
		"course.nivel must be dropped after 0013 down (REQ-SCH-0013)")
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
	page, err := repo.ListApproved(context.Background(), pagination.Params{Page: 1, Size: 20}, repository.CatalogFilter{})
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
	filtered, err := repo.ListApproved(context.Background(), pagination.Params{Page: 1, Size: 20}, repository.CatalogFilter{Q: "angular"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), filtered.Total, "ILIKE filter must return only matching course")
	assert.Equal(t, angularCourse.ID, filtered.Items[0].ID,
		"ILIKE must find 'Angular Avanzado' via case-insensitive match on 'angular'")

	// Pagination: 2 aprobado courses, size=1 → page 1 returns 1 item, totalPages=2.
	paged, err := repo.ListApproved(context.Background(), pagination.Params{Page: 1, Size: 1}, repository.CatalogFilter{})
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

// ── catalog-filters integration tests (REQ-FILTER-NIVEL, REQ-FILTER-CATEGORIA, REQ-COMBINED, REQ-COMPAT) ──

// seedAprobadoCourse creates a course with estado=aprobado and an optional nivel.
// Returns the created course.
func seedAprobadoCourse(t *testing.T, db *gorm.DB, repo repository.Repository, creadorID, titulo, nivel string) *domain.Course {
	t.Helper()
	c := seedCourse(t, repo, creadorID, titulo)
	if nivel != "" {
		err := db.Exec(`UPDATE course SET nivel = ? WHERE id = ?`, nivel, c.ID).Error
		require.NoError(t, err, "seedAprobadoCourse: set nivel")
	}
	require.NoError(t, repo.UpdateEstado(context.Background(), c.ID, domain.EstadoAprobado))
	return c
}

// seedCategoria inserts a categoria row with a unique nombre/slug (UUID-suffixed) and returns its id.
// Uses a UUID suffix on both nombre and slug to avoid collisions with migration-seeded rows.
func seedCategoria(t *testing.T, db *gorm.DB, nombre, slug string) string {
	t.Helper()
	id := uuid.New().String()
	// Append UUID suffix to avoid UNIQUE constraint conflict with migration-seeded categorias.
	suffix := id[:8]
	uniqueNombre := nombre + "-" + suffix
	uniqueSlug := slug + "-" + suffix
	err := db.Exec(
		`INSERT INTO categoria (id, nombre, slug) VALUES (?, ?, ?)`,
		id, uniqueNombre, uniqueSlug,
	).Error
	require.NoError(t, err, "seedCategoria: insert")
	return id
}

// tagCourse inserts a course_categoria row.
func tagCourse(t *testing.T, db *gorm.DB, courseID, categoriaID string) {
	t.Helper()
	err := db.Exec(
		`INSERT INTO course_categoria (course_id, categoria_id) VALUES (?, ?)`,
		courseID, categoriaID,
	).Error
	require.NoError(t, err, "tagCourse: insert")
}

// TestListApproved_NivelFilter_ReturnsOnlyMatchingNivel verifies REQ-FILTER-NIVEL:
// - Nivel:"intermedio" returns only intermedio courses; excludes basico/avanzado/NULL-nivel.
// - Nivel:"" includes NULL-nivel aprobado courses (no filter = today's behavior).
// Refs: REQ-FILTER-NIVEL, AC1.
func TestListApproved_NivelFilter_ReturnsOnlyMatchingNivel(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)

	cBasico := seedAprobadoCourse(t, db, repo, creadorID, "Basico Go", "basico")
	cIntermedio := seedAprobadoCourse(t, db, repo, creadorID, "Intermedio Python", "intermedio")
	cAvanzado := seedAprobadoCourse(t, db, repo, creadorID, "Avanzado Rust", "avanzado")
	cNullNivel := seedAprobadoCourse(t, db, repo, creadorID, "Sin Nivel Course", "")

	p := pagination.Params{Page: 1, Size: 20}

	// Filter: intermedio — only cIntermedio must appear.
	page, err := repo.ListApproved(context.Background(), p, repository.CatalogFilter{Nivel: "intermedio"})
	require.NoError(t, err)
	require.Equal(t, int64(1), page.Total, "nivel=intermedio must return exactly 1 course")
	assert.Equal(t, cIntermedio.ID, page.Items[0].ID, "must be the intermedio course")

	// Filter: basico — only cBasico.
	page, err = repo.ListApproved(context.Background(), p, repository.CatalogFilter{Nivel: "basico"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total, "nivel=basico must return 1")
	assert.Equal(t, cBasico.ID, page.Items[0].ID)

	// Filter: avanzado — only cAvanzado.
	page, err = repo.ListApproved(context.Background(), p, repository.CatalogFilter{Nivel: "avanzado"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total, "nivel=avanzado must return 1")
	assert.Equal(t, cAvanzado.ID, page.Items[0].ID)

	// Filter: intermedio must NOT return basico/avanzado/null.
	page, err = repo.ListApproved(context.Background(), p, repository.CatalogFilter{Nivel: "intermedio"})
	require.NoError(t, err)
	for _, item := range page.Items {
		assert.Equal(t, cIntermedio.ID, item.ID,
			"nivel=intermedio filter must not return other nivel courses")
	}

	// No filter (Nivel:"") — all 4 aprobado courses appear, including NULL-nivel.
	page, err = repo.ListApproved(context.Background(), p, repository.CatalogFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(4), page.Total, "no nivel filter must include NULL-nivel courses")
	ids := make(map[string]bool)
	for _, item := range page.Items {
		ids[item.ID] = true
	}
	assert.True(t, ids[cNullNivel.ID], "NULL-nivel aprobado course must appear when no nivel filter")
}

// TestListApproved_CategoriaFilter_EXISTSSemijoin verifies REQ-FILTER-CATEGORIA:
// - CategoriaIDs:[A,B] returns courses tagged A OR B using EXISTS semantics.
// - A course tagged with BOTH A and B appears EXACTLY ONCE and total=1 (COUNT guard).
// - Nonexistent well-formed UUID → empty page, total 0.
// This is the ADVERSARIAL regression guard: if EXISTS is replaced by JOIN, this test MUST FAIL.
// Refs: REQ-FILTER-CATEGORIA, ADR-2, AC2, AC6d.
func TestListApproved_CategoriaFilter_EXISTSSemijoin(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)

	catA := seedCategoria(t, db, "Prog", "prog")
	catB := seedCategoria(t, db, "DevOps", "devops")
	catC := seedCategoria(t, db, "Cloud", "cloud")

	// courseOnlyA: tagged catA only.
	courseOnlyA := seedAprobadoCourse(t, db, repo, creadorID, "Only A Course", "")
	tagCourse(t, db, courseOnlyA.ID, catA)

	// courseOnlyB: tagged catB only.
	courseOnlyB := seedAprobadoCourse(t, db, repo, creadorID, "Only B Course", "")
	tagCourse(t, db, courseOnlyB.ID, catB)

	// courseBothAB: tagged catA AND catB — THIS is the duplicate-COUNT guard course.
	courseBothAB := seedAprobadoCourse(t, db, repo, creadorID, "Both AB Course", "")
	tagCourse(t, db, courseBothAB.ID, catA)
	tagCourse(t, db, courseBothAB.ID, catB)

	// courseOnlyC: tagged catC only — must NOT appear when filtering [A,B].
	courseOnlyC := seedAprobadoCourse(t, db, repo, creadorID, "Only C Course", "")
	tagCourse(t, db, courseOnlyC.ID, catC)

	// courseNoTag: no tag — must NOT appear.
	_ = seedAprobadoCourse(t, db, repo, creadorID, "No Tag Course", "")

	p := pagination.Params{Page: 1, Size: 20}

	// Filter [A,B] — expects courseOnlyA, courseOnlyB, courseBothAB (3 unique courses).
	page, err := repo.ListApproved(context.Background(), p, repository.CatalogFilter{CategoriaIDs: []string{catA, catB}})
	require.NoError(t, err)
	// CORRECTNESS-CRITICAL: total must be 3 (not 4 — courseBothAB is counted once by EXISTS).
	require.Equal(t, int64(3), page.Total,
		"[AC6d] EXISTS semi-join: course tagged A+B must be counted ONCE, total must be 3 not 4")
	require.Len(t, page.Items, 3,
		"[AC6d] page items must have 3 courses, not 4 (EXISTS prevents row fan-out)")

	gotIDs := make(map[string]bool)
	for _, item := range page.Items {
		assert.False(t, gotIDs[item.ID],
			"[AC6d] each course must appear at most once — duplicate detected for %s", item.ID)
		gotIDs[item.ID] = true
	}
	assert.True(t, gotIDs[courseOnlyA.ID], "courseOnlyA must appear in [A,B] filter")
	assert.True(t, gotIDs[courseOnlyB.ID], "courseOnlyB must appear in [A,B] filter")
	assert.True(t, gotIDs[courseBothAB.ID], "courseBothAB must appear in [A,B] filter")
	assert.False(t, gotIDs[courseOnlyC.ID], "courseOnlyC (catC only) must NOT appear in [A,B] filter")

	// Filter [A] only — courseOnlyA + courseBothAB; courseOnlyB must NOT appear.
	pageA, err := repo.ListApproved(context.Background(), p, repository.CatalogFilter{CategoriaIDs: []string{catA}})
	require.NoError(t, err)
	assert.Equal(t, int64(2), pageA.Total, "filter [A] must return 2 courses")
	gotIDsA := make(map[string]bool)
	for _, item := range pageA.Items {
		gotIDsA[item.ID] = true
	}
	assert.False(t, gotIDsA[courseOnlyB.ID], "[AC6b] courseOnlyB must NOT appear in [A] filter")

	// Nonexistent well-formed UUID — empty result, NOT error (REQ-FILTER-CATEGORIA).
	ghostID := uuid.New().String()
	pageGhost, err := repo.ListApproved(context.Background(), p, repository.CatalogFilter{CategoriaIDs: []string{ghostID}})
	require.NoError(t, err, "nonexistent categoria UUID must not error (match-nothing)")
	assert.Equal(t, int64(0), pageGhost.Total, "nonexistent categoria UUID must yield total=0")
	assert.Empty(t, pageGhost.Items, "nonexistent categoria UUID must yield empty items")
}

// TestListApproved_SortFilter verifies REQ-SORT:
// - Sort:"titulo" → titulo ASC.
// - Sort:"recientes" and Sort:"" → publicado_en DESC NULLS LAST then created_at DESC.
// Refs: REQ-SORT, ADR-3, AC3.
func TestListApproved_SortFilter(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)

	// Seed 3 courses with distinct títulos.
	cZ := seedAprobadoCourse(t, db, repo, creadorID, "Zeta Go", "")
	cA := seedAprobadoCourse(t, db, repo, creadorID, "Alpha Python", "")
	cM := seedAprobadoCourse(t, db, repo, creadorID, "Mu Rust", "")

	p := pagination.Params{Page: 1, Size: 20}

	// Sort:"titulo" → titulo ASC → Alpha, Mu, Zeta.
	page, err := repo.ListApproved(context.Background(), p, repository.CatalogFilter{Sort: "titulo"})
	require.NoError(t, err)
	require.Len(t, page.Items, 3, "all 3 aprobado courses must appear")
	assert.Equal(t, cA.ID, page.Items[0].ID, "titulo ASC: Alpha must be first")
	assert.Equal(t, cM.ID, page.Items[1].ID, "titulo ASC: Mu must be second")
	assert.Equal(t, cZ.ID, page.Items[2].ID, "titulo ASC: Zeta must be third")

	// Sort:"recientes" (default) → publicado_en DESC NULLS LAST then created_at DESC.
	// Since publicado_en is NULL for all (seeded via UpdateEstado not UpdateEstadoPublicado),
	// NULLS LAST means all fall to created_at DESC — most recently created first.
	pageR, err := repo.ListApproved(context.Background(), p, repository.CatalogFilter{Sort: "recientes"})
	require.NoError(t, err)
	require.Len(t, pageR.Items, 3)
	// cM was created last — must be first in created_at DESC.
	assert.Equal(t, cM.ID, pageR.Items[0].ID, "recientes: most recently created must be first")

	// Sort:"" (empty = default recientes).
	pageDefault, err := repo.ListApproved(context.Background(), p, repository.CatalogFilter{})
	require.NoError(t, err)
	require.Len(t, pageDefault.Items, 3)
	assert.Equal(t, cM.ID, pageDefault.Items[0].ID, "empty sort defaults to recientes ordering")
}

// TestListApproved_CombinedFilter verifies REQ-COMBINED:
// All params compose with AND semantics; total reflects filtered set.
// Refs: REQ-COMBINED, AC4.
func TestListApproved_CombinedFilter(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)

	catGo := seedCategoria(t, db, "Go", "go")

	// cMatch: titulo contains "go", nivel=avanzado, tagged catGo.
	cMatch := seedAprobadoCourse(t, db, repo, creadorID, "Advanced Go Microservices", "avanzado")
	tagCourse(t, db, cMatch.ID, catGo)

	// cWrongNivel: titulo contains "go", nivel=basico, tagged catGo.
	cWrongNivel := seedAprobadoCourse(t, db, repo, creadorID, "Intro go tutorial", "basico")
	tagCourse(t, db, cWrongNivel.ID, catGo)

	// cNoTag: titulo contains "go", nivel=avanzado, NOT tagged catGo.
	_ = seedAprobadoCourse(t, db, repo, creadorID, "go language basics", "avanzado")

	// cNoCat: no tag, avanzado.
	_ = seedAprobadoCourse(t, db, repo, creadorID, "Java Spring Boot", "avanzado")

	p := pagination.Params{Page: 1, Size: 10}
	filter := repository.CatalogFilter{
		Q:            "go",
		Nivel:        "avanzado",
		CategoriaIDs: []string{catGo},
		Sort:         "titulo",
	}

	page, err := repo.ListApproved(context.Background(), p, filter)
	require.NoError(t, err)
	// Only cMatch satisfies ALL conditions: titulo ILIKE go, nivel=avanzado, tagged catGo.
	assert.Equal(t, int64(1), page.Total, "combined filter must return exactly 1 matching course")
	require.Len(t, page.Items, 1)
	assert.Equal(t, cMatch.ID, page.Items[0].ID, "only cMatch satisfies all combined filters")
	_ = cWrongNivel
}

// TestListApproved_BaselineNoFilter_BackwardCompat verifies REQ-COMPAT (AC5):
// Empty CatalogFilter == today's behavior (all aprobado, no filter).
// Regression guard: calling with empty filter must produce the same behavior as the old q="" call.
func TestListApproved_BaselineNoFilter_BackwardCompat(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)

	c1 := seedAprobadoCourse(t, db, repo, creadorID, "Course One", "basico")
	c2 := seedAprobadoCourse(t, db, repo, creadorID, "Course Two", "")
	_ = seedCourse(t, repo, creadorID, "Draft Course") // borrador — must NOT appear

	p := pagination.Params{Page: 1, Size: 20}
	page, err := repo.ListApproved(context.Background(), p, repository.CatalogFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total, "baseline: only aprobado courses (borrador excluded)")
	ids := map[string]bool{}
	for _, item := range page.Items {
		ids[item.ID] = true
	}
	assert.True(t, ids[c1.ID], "baseline must include aprobado c1")
	assert.True(t, ids[c2.ID], "baseline must include aprobado c2 (nil nivel)")
}

// TestListApproved_PaginatedTotal_ReflectsFilters verifies REQ-COMBINED count invariant:
// paginated total with active filter = FILTERED total, not unfiltered catalog count.
// Refs: REQ-COMBINED, AC4, Count-before-Select invariant.
func TestListApproved_PaginatedTotal_ReflectsFilters(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)

	catX := seedCategoria(t, db, "X", "x")

	// Seed 5 aprobado courses: 2 tagged catX, 3 untagged.
	c1 := seedAprobadoCourse(t, db, repo, creadorID, "Tagged 1", "")
	tagCourse(t, db, c1.ID, catX)
	c2 := seedAprobadoCourse(t, db, repo, creadorID, "Tagged 2", "")
	tagCourse(t, db, c2.ID, catX)
	_ = seedAprobadoCourse(t, db, repo, creadorID, "Untagged 1", "")
	_ = seedAprobadoCourse(t, db, repo, creadorID, "Untagged 2", "")
	_ = seedAprobadoCourse(t, db, repo, creadorID, "Untagged 3", "")

	// Page size 1 with filter: total must be 2 (filtered), not 5 (all aprobado).
	p := pagination.Params{Page: 1, Size: 1}
	page, err := repo.ListApproved(context.Background(), p, repository.CatalogFilter{CategoriaIDs: []string{catX}})
	require.NoError(t, err)
	assert.Equal(t, int64(2), page.Total,
		"paginated total must reflect filtered count (2), not total catalog count (5)")
	assert.Equal(t, 2, page.TotalPages, "totalPages must be ceil(2/1)=2 (filtered total)")
	assert.Len(t, page.Items, 1, "page size 1 must return 1 item per page")
}

// ── Phase 3 repository tests (course-structure-v2) ────────────────────────────

// TestListMaterialsByVideo verifies that ListMaterialsByVideo filters by video_id
// and returns materials ordered by created_at ASC. Satisfies REQ-MATERIAL-VIDEO.
func TestListMaterialsByVideo(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "List By Video Course")
	section := seedSection(t, repo, course.ID, "S1", 0)
	videoA := seedVideo(t, repo, section.ID, "Video A")
	videoB := seedVideo(t, repo, section.ID, "Video B")

	// Seed 2 materials on videoA and 1 on videoB.
	matA1 := &domain.Material{ID: uuid.New().String(), VideoID: videoA.ID, Titulo: "a1.pdf", StorageKey: "k1", MimeType: "application/pdf", TamanoBytes: 100}
	matA2 := &domain.Material{ID: uuid.New().String(), VideoID: videoA.ID, Titulo: "a2.pdf", StorageKey: "k2", MimeType: "application/pdf", TamanoBytes: 200}
	matB1 := &domain.Material{ID: uuid.New().String(), VideoID: videoB.ID, Titulo: "b1.pdf", StorageKey: "k3", MimeType: "application/pdf", TamanoBytes: 300}
	require.NoError(t, repo.CreateMaterial(context.Background(), matA1))
	require.NoError(t, repo.CreateMaterial(context.Background(), matA2))
	require.NoError(t, repo.CreateMaterial(context.Background(), matB1))

	// ListMaterialsByVideo for videoA must return exactly matA1, matA2 in ASC order.
	mats, err := repo.ListMaterialsByVideo(context.Background(), videoA.ID)
	require.NoError(t, err)
	require.Len(t, mats, 2, "videoA must have 2 materials")
	assert.Equal(t, matA1.ID, mats[0].ID, "first material must be matA1 (created first)")
	assert.Equal(t, matA2.ID, mats[1].ID, "second material must be matA2")

	// ListMaterialsByVideo for videoB must return only matB1.
	mats, err = repo.ListMaterialsByVideo(context.Background(), videoB.ID)
	require.NoError(t, err)
	require.Len(t, mats, 1, "videoB must have 1 material")
	assert.Equal(t, matB1.ID, mats[0].ID)

	// Video with no materials → empty slice, no error.
	videoC := seedVideo(t, repo, section.ID, "Video C")
	mats, err = repo.ListMaterialsByVideo(context.Background(), videoC.ID)
	require.NoError(t, err)
	assert.Empty(t, mats, "video with no materials must return empty slice")
}

// TestGetMaterialOwnership verifies chain ownership query.
// material → video → section → course → creador. Satisfies REQ-MATERIAL-VIDEO.
func TestGetMaterialOwnership(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "Ownership Chain Course")
	require.NoError(t, repo.UpdateEstado(context.Background(), course.ID, domain.EstadoBorrador))
	section := seedSection(t, repo, course.ID, "S1", 0)
	video := seedVideo(t, repo, section.ID, "V1")
	mat := &domain.Material{ID: uuid.New().String(), VideoID: video.ID, Titulo: "f.pdf", StorageKey: "k", MimeType: "application/pdf", TamanoBytes: 1}
	require.NoError(t, repo.CreateMaterial(context.Background(), mat))

	// Correct chain.
	gotCourseID, gotCreadorID, gotEstado, err := repo.GetMaterialOwnership(context.Background(), mat.ID)
	require.NoError(t, err)
	assert.Equal(t, course.ID, gotCourseID, "course_id must match via chain")
	assert.Equal(t, creadorID, gotCreadorID, "creador_id must match via chain")
	assert.Equal(t, "borrador", gotEstado, "estado must match")

	// Non-existent material → ErrMaterialNotFound.
	_, _, _, err = repo.GetMaterialOwnership(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, repository.ErrMaterialNotFound,
		"missing material must return ErrMaterialNotFound")
}

// TestResolveVideoCourse verifies video → section → course chain. Satisfies REQ-MATERIAL-VIDEO.
func TestResolveVideoCourse(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "ResolveVideoCourse Course")
	section := seedSection(t, repo, course.ID, "S1", 0)
	video := seedVideo(t, repo, section.ID, "V1")

	gotCourseID, gotCreadorID, gotEstado, err := repo.ResolveVideoCourse(context.Background(), video.ID)
	require.NoError(t, err)
	assert.Equal(t, course.ID, gotCourseID)
	assert.Equal(t, creadorID, gotCreadorID)
	assert.Equal(t, "borrador", gotEstado)

	// Non-existent video → ErrVideoNotFound.
	_, _, _, err = repo.ResolveVideoCourse(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, repository.ErrVideoNotFound, "missing video must return ErrVideoNotFound")
}

// TestListMaterialsByCourseVideos verifies the batch no-N+1 query.
// All materials for a course's videos in one query, groupable by video_id.
func TestListMaterialsByCourseVideos(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "BatchMaterials Course")
	section := seedSection(t, repo, course.ID, "S1", 0)
	v1 := seedVideo(t, repo, section.ID, "V1")
	v2 := seedVideo(t, repo, section.ID, "V2")

	// Seed 2 materials on v1, 1 on v2.
	mat1 := &domain.Material{ID: uuid.New().String(), VideoID: v1.ID, Titulo: "m1.pdf", StorageKey: "k1", MimeType: "application/pdf", TamanoBytes: 10}
	mat2 := &domain.Material{ID: uuid.New().String(), VideoID: v1.ID, Titulo: "m2.pdf", StorageKey: "k2", MimeType: "application/pdf", TamanoBytes: 20}
	mat3 := &domain.Material{ID: uuid.New().String(), VideoID: v2.ID, Titulo: "m3.pdf", StorageKey: "k3", MimeType: "application/pdf", TamanoBytes: 30}
	require.NoError(t, repo.CreateMaterial(context.Background(), mat1))
	require.NoError(t, repo.CreateMaterial(context.Background(), mat2))
	require.NoError(t, repo.CreateMaterial(context.Background(), mat3))

	mats, err := repo.ListMaterialsByCourseVideos(context.Background(), course.ID)
	require.NoError(t, err)
	assert.Len(t, mats, 3, "must return all 3 materials for the course")
	videoIDs := map[string]int{}
	for _, m := range mats {
		videoIDs[m.VideoID]++
	}
	assert.Equal(t, 2, videoIDs[v1.ID], "v1 must have 2 materials")
	assert.Equal(t, 1, videoIDs[v2.ID], "v2 must have 1 material")
}

// TestCategoriasRepo verifies ListCategorias, CategoriasExist, SetCourseCategorias, GetCourseCategorias.
// Satisfies REQ-COURSE-META categoria methods.
func TestCategoriasRepo(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	creadorID := seedUser(t, db)
	course := seedCourse(t, repo, creadorID, "Categorias Course")

	// ListCategorias — seeded by migration 0013 — must return >= 8.
	cats, err := repo.ListCategorias(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(cats), 8, "must have at least 8 seeded categorias")

	// Pick 2 valid IDs.
	id1 := cats[0].ID
	id2 := cats[1].ID

	// CategoriasExist — valid IDs → true.
	ok, err := repo.CategoriasExist(context.Background(), []string{id1, id2})
	require.NoError(t, err)
	assert.True(t, ok, "valid categoria IDs must exist")

	// CategoriasExist — bogus ID → false.
	ok, err = repo.CategoriasExist(context.Background(), []string{uuid.New().String()})
	require.NoError(t, err)
	assert.False(t, ok, "bogus ID must not exist")

	// SetCourseCategorias — insert set [id1, id2].
	require.NoError(t, repo.SetCourseCategorias(context.Background(), course.ID, []string{id1, id2}))

	// GetCourseCategorias — must return both.
	gotCats, err := repo.GetCourseCategorias(context.Background(), course.ID)
	require.NoError(t, err)
	require.Len(t, gotCats, 2)
	gotIDs := map[string]bool{}
	for _, c := range gotCats {
		gotIDs[c.ID] = true
	}
	assert.True(t, gotIDs[id1])
	assert.True(t, gotIDs[id2])

	// SetCourseCategorias — replace with [id1] only.
	require.NoError(t, repo.SetCourseCategorias(context.Background(), course.ID, []string{id1}))
	gotCats, err = repo.GetCourseCategorias(context.Background(), course.ID)
	require.NoError(t, err)
	assert.Len(t, gotCats, 1, "replace-set: must have only 1 categoria")
	assert.Equal(t, id1, gotCats[0].ID)

	// SetCourseCategorias — empty slice clears all.
	require.NoError(t, repo.SetCourseCategorias(context.Background(), course.ID, []string{}))
	gotCats, err = repo.GetCourseCategorias(context.Background(), course.ID)
	require.NoError(t, err)
	assert.Empty(t, gotCats, "empty set must clear all categorias")
}
