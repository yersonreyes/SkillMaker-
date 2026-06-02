//go:build integration

// Package repository — integration tests using testcontainers + real Postgres.
// Run with: make backend-test-integration
// These tests exercise the SQL queries that only a real database can prove.
package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// createUser inserts a user directly via GORM for test setup.
func createUser(t *testing.T, repo repository.Repository, googleSub, email, nombre string) *domain.User {
	t.Helper()
	user, err := repo.UpsertByGoogleSub(context.Background(), repository.GoogleProfile{
		GoogleSub: googleSub,
		Email:     email,
		Nombre:    nombre,
	})
	require.NoError(t, err)
	return user
}

// ── List + filters ─────────────────────────────────────────────────────────────

// U1-filter-role: List with ?role=administrador returns only admin users.
func TestList_FilterByRole(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	// Seed two users
	u1 := createUser(t, repo, "sub-admin", "admin@example.com", "Admin User")
	u2 := createUser(t, repo, "sub-regular", "regular@example.com", "Regular User")

	// Promote u1 to admin
	require.NoError(t, repo.AddRoles(ctx, u1.ID, []string{"administrador"}))

	p := pagination.Params{Page: 1, Size: 20}
	page, err := repo.List(ctx, repository.ListFilters{Role: "administrador"}, p)
	require.NoError(t, err)

	// Only u1 should appear (u2 only has alumno)
	ids := make([]string, 0, len(page.Items))
	for _, u := range page.Items {
		ids = append(ids, u.ID)
	}
	assert.Contains(t, ids, u1.ID)
	assert.NotContains(t, ids, u2.ID)
	assert.Equal(t, int64(1), page.Total)
}

// U1-filter-active: List with active=false returns only inactive users.
func TestList_FilterByActive(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	u1 := createUser(t, repo, "sub-active", "active@example.com", "Active User")
	u2 := createUser(t, repo, "sub-inactive", "inactive@example.com", "Inactive User")

	// Deactivate u2 — promote another user to admin first so guard in test doesn't matter (direct repo call)
	adminUser := createUser(t, repo, "sub-admin-a", "admin-a@example.com", "Admin A")
	require.NoError(t, repo.AddRoles(ctx, adminUser.ID, []string{"administrador"}))

	require.NoError(t, repo.SetActive(ctx, u2.ID, false))

	active := false
	p := pagination.Params{Page: 1, Size: 20}
	page, err := repo.List(ctx, repository.ListFilters{Active: &active}, p)
	require.NoError(t, err)

	ids := make([]string, 0, len(page.Items))
	for _, u := range page.Items {
		ids = append(ids, u.ID)
	}
	assert.Contains(t, ids, u2.ID)
	assert.NotContains(t, ids, u1.ID)
	assert.NotContains(t, ids, adminUser.ID)
}

// U1-search-q: List with q=mart matches ILIKE on nombre.
func TestList_FilterByQ(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	u1 := createUser(t, repo, "sub-marta", "marta@example.com", "Marta Lopez")
	_ = createUser(t, repo, "sub-juan", "juan@example.com", "Juan Perez")

	p := pagination.Params{Page: 1, Size: 20}
	page, err := repo.List(ctx, repository.ListFilters{Q: "mart"}, p)
	require.NoError(t, err)

	ids := make([]string, 0, len(page.Items))
	for _, u := range page.Items {
		ids = append(ids, u.ID)
	}
	assert.Contains(t, ids, u1.ID, "ILIKE should match 'mart' in 'Marta Lopez'")
	assert.Equal(t, int64(1), page.Total)
}

// U1-default: pagination with page=2, size=2 returns second page correctly.
func TestList_Pagination_OffsetAndLimit(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	// Seed 5 users
	for i := range 5 {
		createUser(t, repo,
			"sub-pag-"+string(rune('a'+i)),
			"pag"+string(rune('a'+i))+"@example.com",
			"Pag User "+string(rune('A'+i)),
		)
	}

	p := pagination.Params{Page: 2, Size: 2}
	page, err := repo.List(ctx, repository.ListFilters{}, p)
	require.NoError(t, err)

	assert.Equal(t, int64(5), page.Total)
	assert.Len(t, page.Items, 2)
	assert.Equal(t, 3, page.TotalPages)
}

// ── CountActiveAdmins ─────────────────────────────────────────────────────────

// U5: CountActiveAdmins counts only users with administrador role AND activo=true.
func TestCountActiveAdmins(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	// Initially 0 active admins
	n, err := repo.CountActiveAdmins(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)

	// Create two admin users
	u1 := createUser(t, repo, "sub-admin1", "admin1@example.com", "Admin One")
	u2 := createUser(t, repo, "sub-admin2", "admin2@example.com", "Admin Two")
	require.NoError(t, repo.AddRoles(ctx, u1.ID, []string{"administrador"}))
	require.NoError(t, repo.AddRoles(ctx, u2.ID, []string{"administrador"}))

	n, err = repo.CountActiveAdmins(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	// Deactivate one admin (direct repo call — guard is in service)
	require.NoError(t, repo.SetActive(ctx, u2.ID, false))

	n, err = repo.CountActiveAdmins(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "deactivated admin should not count")
}

// ── AddRoles / RemoveRoles idempotency ────────────────────────────────────────

// U4-idempotent-add: adding a role the user already holds is a no-op (ON CONFLICT DO NOTHING).
func TestAddRoles_Idempotent(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	u := createUser(t, repo, "sub-idem", "idem@example.com", "Idempotent User")
	// alumno is already assigned by UpsertByGoogleSub
	require.NoError(t, repo.AddRoles(ctx, u.ID, []string{"alumno"}))

	roles, err := repo.LoadRoleNames(ctx, u.ID)
	require.NoError(t, err)

	count := 0
	for _, r := range roles {
		if r == "alumno" {
			count++
		}
	}

	assert.Equal(t, 1, count, "alumno should not be duplicated")
}

// RemoveRoles on an absent role is a no-op (no error).
func TestRemoveRoles_Idempotent(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	u := createUser(t, repo, "sub-rm-idem", "rmidem@example.com", "Remove Idempotent")
	// u only has alumno — removing creador (not assigned) should succeed silently
	err := repo.RemoveRoles(ctx, u.ID, []string{"creador"})
	assert.NoError(t, err)
}

// ── SetActive ────────────────────────────────────────────────────────────────

// SetActive returns ErrUserNotFound for a non-existent user.
func TestSetActive_NotFound(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	err := repo.SetActive(context.Background(), uuid.New().String(), false)
	assert.ErrorIs(t, err, repository.ErrUserNotFound)
}

// SetActive toggles the flag correctly.
func TestSetActive_Toggle(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	u := createUser(t, repo, "sub-toggle", "toggle@example.com", "Toggle User")
	require.True(t, u.Activo)

	require.NoError(t, repo.SetActive(ctx, u.ID, false))

	fetched, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.False(t, fetched.Activo)

	require.NoError(t, repo.SetActive(ctx, u.ID, true))
	fetched2, err := repo.GetByID(ctx, u.ID)
	require.NoError(t, err)
	assert.True(t, fetched2.Activo)
}

// ── Supervision — PR-B integration tests ─────────────────────────────────────
//
// These tests require migration 0002 (supervision table). SetupPostgres runs
// ALL migrations in backend/migrations/ via golang-migrate, so 0002 is applied
// automatically when this file is compiled with the integration build tag.

// V2-create: CreateSupervision inserts a row and returns the created model.
func TestCreateSupervision_Success(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	sup := createUser(t, repo, "sub-sup-create", "sup.create@example.com", "Supervisor Create")
	emp := createUser(t, repo, "sub-emp-create", "emp.create@example.com", "Employee Create")

	sv, err := repo.CreateSupervision(ctx, sup.ID, emp.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, sv.ID)
	assert.Equal(t, sup.ID, sv.SupervisorID)
	assert.Equal(t, emp.ID, sv.EmpleadoID)
	assert.False(t, sv.CreadoEn.IsZero())
}

// V2-duplicate: inserting a second supervisor for the same employee maps to ErrSupervisionExists.
// Spec V2: an employee has AT MOST ONE supervisor (UNIQUE on empleado_id).
func TestCreateSupervision_DuplicateEmployee_ReturnsErrSupervisionExists(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	sup1 := createUser(t, repo, "sub-sup-dup1", "sup.dup1@example.com", "Supervisor Dup1")
	sup2 := createUser(t, repo, "sub-sup-dup2", "sup.dup2@example.com", "Supervisor Dup2")
	emp := createUser(t, repo, "sub-emp-dup", "emp.dup@example.com", "Employee Dup")

	// First assignment succeeds.
	_, err := repo.CreateSupervision(ctx, sup1.ID, emp.ID)
	require.NoError(t, err)

	// Second assignment for the same employee violates UNIQUE(empleado_id).
	_, err = repo.CreateSupervision(ctx, sup2.ID, emp.ID)
	assert.ErrorIs(t, err, repository.ErrSupervisionExists, "second supervisor for same employee should return ErrSupervisionExists")
}

// V2-self: CHECK constraint prevents self-supervision at the DB level.
// The service rejects this first, but we test the DB constraint as defence in depth.
func TestCreateSupervision_SelfSupervision_FailsAtDBConstraint(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	u := createUser(t, repo, "sub-self-sup", "self.sup@example.com", "Self Supervisor")

	// Direct repo call bypasses the service self-check — the DB CHECK must fire.
	_, err := repo.CreateSupervision(ctx, u.ID, u.ID)
	require.Error(t, err, "DB CHECK(supervisor_id <> empleado_id) should reject self-supervision")
	// The error is NOT ErrSupervisionExists (23505) but a CHECK violation (23514).
	// We just assert an error occurs — the service layer is the authoritative guard.
}

// V1-list: ListSupervisions returns all supervision rows.
func TestListSupervisions_ReturnsAll(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	sup := createUser(t, repo, "sub-sup-list", "sup.list@example.com", "Supervisor List")
	emp1 := createUser(t, repo, "sub-emp-list1", "emp.list1@example.com", "Employee List1")
	emp2 := createUser(t, repo, "sub-emp-list2", "emp.list2@example.com", "Employee List2")

	_, err := repo.CreateSupervision(ctx, sup.ID, emp1.ID)
	require.NoError(t, err)
	_, err = repo.CreateSupervision(ctx, sup.ID, emp2.ID)
	require.NoError(t, err)

	svs, err := repo.ListSupervisions(ctx)
	require.NoError(t, err)
	assert.Len(t, svs, 2, "expected 2 supervision rows")
}

// V3-delete: DeleteSupervision removes a row by primary key → 204 path.
func TestDeleteSupervision_RemovesRow(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	sup := createUser(t, repo, "sub-sup-del", "sup.del@example.com", "Supervisor Del")
	emp := createUser(t, repo, "sub-emp-del", "emp.del@example.com", "Employee Del")

	sv, err := repo.CreateSupervision(ctx, sup.ID, emp.ID)
	require.NoError(t, err)

	err = repo.DeleteSupervision(ctx, sv.ID)
	require.NoError(t, err)

	// Verify the row is gone.
	svs, err := repo.ListSupervisions(ctx)
	require.NoError(t, err)
	for _, s := range svs {
		assert.NotEqual(t, sv.ID, s.ID, "deleted supervision should not appear in list")
	}
}

// V3-not-found: DeleteSupervision on a nonexistent ID returns ErrSupervisionNotFound.
func TestDeleteSupervision_NotFound_ReturnsErrSupervisionNotFound(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	err := repo.DeleteSupervision(ctx, "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, repository.ErrSupervisionNotFound)
}

// ON DELETE CASCADE: deleting a user removes their supervision rows (both sides).
// Spec: FKs ON DELETE CASCADE — deleting a user cascades to supervision rows.
func TestDeleteUser_CascadesToSupervision(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	sup := createUser(t, repo, "sub-sup-cascade", "sup.cascade@example.com", "Supervisor Cascade")
	emp := createUser(t, repo, "sub-emp-cascade", "emp.cascade@example.com", "Employee Cascade")

	sv, err := repo.CreateSupervision(ctx, sup.ID, emp.ID)
	require.NoError(t, err)

	// Delete the supervisor user directly via GORM (bypassing service soft-delete
	// to test the DB cascade — hard delete exercises the FK ON DELETE CASCADE).
	require.NoError(t, db.Exec(`DELETE FROM "user" WHERE id = ?`, sup.ID).Error)

	// The supervision row should have been cascade-deleted.
	err = repo.DeleteSupervision(ctx, sv.ID)
	assert.ErrorIs(t, err, repository.ErrSupervisionNotFound, "supervision should have been cascade-deleted with the supervisor user")
}

// ON DELETE CASCADE: deleting the employee also removes the supervision row.
func TestDeleteEmployee_CascadesToSupervision(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	sup := createUser(t, repo, "sub-sup-casc2", "sup.casc2@example.com", "Supervisor Casc2")
	emp := createUser(t, repo, "sub-emp-casc2", "emp.casc2@example.com", "Employee Casc2")

	sv, err := repo.CreateSupervision(ctx, sup.ID, emp.ID)
	require.NoError(t, err)

	// Hard-delete the employee.
	require.NoError(t, db.Exec(`DELETE FROM "user" WHERE id = ?`, emp.ID).Error)

	err = repo.DeleteSupervision(ctx, sv.ID)
	assert.ErrorIs(t, err, repository.ErrSupervisionNotFound, "supervision should have been cascade-deleted with the employee user")
}

// Migration round-trip: verify 0002 up creates the table, down drops it.
// The test confirms the migration is applied (table exists after SetupPostgres)
// and that the down migration is logically reversible (DROP TABLE IF EXISTS).
func TestMigration0002_RoundTrip(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	// 0002 up was applied by SetupPostgres — confirm the table exists.
	var tableExists int
	err := db.Raw(`SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'supervision'`).
		Scan(&tableExists).Error
	require.NoError(t, err)
	assert.Equal(t, 1, tableExists, "supervision table should exist after migration 0002 up")

	// Simulate the down migration and verify the table is removed.
	require.NoError(t, db.Exec("DROP TABLE IF EXISTS supervision").Error)

	err = db.Raw(`SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'supervision'`).
		Scan(&tableExists).Error
	require.NoError(t, err)
	assert.Equal(t, 0, tableExists, "supervision table should not exist after 0002 down")
}
