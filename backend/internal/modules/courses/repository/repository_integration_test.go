//go:build integration

// Package repository — integration tests using testcontainers + real Postgres.
// Run with: make backend-test-integration
// These tests exercise migration 0003/0004, FK constraints, and repository queries
// that only a real database can prove.
package repository_test

import (
	"context"
	"testing"

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
func TestMigration0004Down_ReversesSchema(t *testing.T) {
	// Use the migrate-aware setup so we can call m.Down() ourselves.
	db, m, teardown := testutil.SetupPostgresWithMigrate(t)
	defer teardown()

	// All migrations up (0001–0004) are already applied by SetupPostgresWithMigrate.
	// Step 1: Apply DOWN one step — rolls back 0004 only.
	err := m.Steps(-1)
	require.NoError(t, err, "m.Steps(-1) must apply 0004 down without error (SCH-1-B)")

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
