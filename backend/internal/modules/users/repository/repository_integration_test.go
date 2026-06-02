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
