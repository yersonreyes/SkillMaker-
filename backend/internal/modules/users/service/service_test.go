// Package service — white-box unit tests for the users service.
// No build tag: runs with standard `make backend-test`.
//
// Test coverage targets:
//   - Last-admin invariant (U5): ErrLastAdmin from PatchRoles and SetActive
//   - Role-patch validation (U4): ErrInvalidRole, ErrAddRemoveConflict, idempotency
//   - GetDetail (U2/U3): found, not-found
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// ── Mock repository ────────────────────────────────────────────────────────────

// MockUsersRepository is a testify/mock implementation of repository.Repository.
type MockUsersRepository struct {
	mock.Mock
}

func (m *MockUsersRepository) UpsertByGoogleSub(ctx context.Context, p repository.GoogleProfile) (*domain.User, error) {
	args := m.Called(ctx, p)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUsersRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUsersRepository) LoadRoleNames(ctx context.Context, userID string) ([]string, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]string), args.Error(1)
}

func (m *MockUsersRepository) List(ctx context.Context, f repository.ListFilters, p pagination.Params) (pagination.Page[domain.User], error) {
	args := m.Called(ctx, f, p)
	return args.Get(0).(pagination.Page[domain.User]), args.Error(1)
}

func (m *MockUsersRepository) SetActive(ctx context.Context, id string, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}

func (m *MockUsersRepository) AddRoles(ctx context.Context, userID string, roleNames []string) error {
	args := m.Called(ctx, userID, roleNames)
	return args.Error(0)
}

func (m *MockUsersRepository) RemoveRoles(ctx context.Context, userID string, roleNames []string) error {
	args := m.Called(ctx, userID, roleNames)
	return args.Error(0)
}

func (m *MockUsersRepository) CountActiveAdmins(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUsersRepository) ExistsUser(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(bool), args.Error(1)
}

func (m *MockUsersRepository) CreateSupervision(ctx context.Context, supervisorID, empleadoID string) (*domain.Supervision, error) {
	args := m.Called(ctx, supervisorID, empleadoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.Supervision), args.Error(1)
}

func (m *MockUsersRepository) ListSupervisions(ctx context.Context) ([]domain.Supervision, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]domain.Supervision), args.Error(1)
}

func (m *MockUsersRepository) DeleteSupervision(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUsersRepository) ListEmployeesBySupervisor(ctx context.Context, supervisorID string) ([]domain.User, error) {
	args := m.Called(ctx, supervisorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]domain.User), args.Error(1)
}

// ── Fixtures ───────────────────────────────────────────────────────────────────

func adminUser(id string) *domain.User {
	return &domain.User{
		ID:        id,
		Email:     id + "@example.com",
		Nombre:    "Admin " + id,
		Activo:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Roles:     []domain.Role{{ID: 4, Nombre: "administrador"}},
	}
}

func regularUser(id string) *domain.User {
	return &domain.User{
		ID:        id,
		Email:     id + "@example.com",
		Nombre:    "User " + id,
		Activo:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Roles:     []domain.Role{{ID: 1, Nombre: "alumno"}},
	}
}

// newSvc is a helper that returns a concrete *serviceImpl for white-box testing.
func newSvc(repo repository.Repository) *serviceImpl {
	return New(repo).(*serviceImpl)
}

// ── U5: Last-admin invariant — PatchRoles path ─────────────────────────────────

// U5-last-admin-role: removing "administrador" from the last active admin → ErrLastAdmin
func TestPatchRoles_LastAdmin_BlocksRemoveWhenOnlyOne(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	userID := "admin-1"
	user := adminUser(userID)
	// GetByID called twice: once for GetDetail (ensureNotLastAdmin), once for the refresh at the end (blocked before this)
	repo.On("GetByID", mock.Anything, userID).Return(user, nil)
	repo.On("LoadRoleNames", mock.Anything, userID).Return([]string{"administrador"}, nil)
	repo.On("CountActiveAdmins", mock.Anything).Return(int64(1), nil)

	_, err := svc.PatchRoles(context.Background(), userID, []string{}, []string{"administrador"})
	assert.ErrorIs(t, err, ErrLastAdmin, "should block removing admin role from last active admin")
}

// U5-multi-admin-safe: two active admins exist → removing admin role from one succeeds
func TestPatchRoles_LastAdmin_AllowsRemoveWhenMultipleAdmins(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	userID := "admin-1"
	user := adminUser(userID)
	repo.On("GetByID", mock.Anything, userID).Return(user, nil)
	repo.On("LoadRoleNames", mock.Anything, userID).Return([]string{"administrador"}, nil)
	repo.On("CountActiveAdmins", mock.Anything).Return(int64(2), nil)
	repo.On("AddRoles", mock.Anything, userID, []string{}).Return(nil)
	repo.On("RemoveRoles", mock.Anything, userID, []string{"administrador"}).Return(nil)

	// Second GetByID call for refreshed detail after mutation
	refreshedUser := &domain.User{
		ID:        userID,
		Email:     "admin-1@example.com",
		Nombre:    "Admin admin-1",
		Activo:    true,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Roles:     []domain.Role{{ID: 1, Nombre: "alumno"}},
	}
	repo.On("GetByID", mock.Anything, userID).Return(refreshedUser, nil).Maybe()

	_, err := svc.PatchRoles(context.Background(), userID, []string{}, []string{"administrador"})
	assert.NoError(t, err, "should allow removing admin role when other active admins exist")
}

// U5-last-admin-active: deactivating the last active admin → ErrLastAdmin
func TestSetActive_LastAdmin_BlocksDeactivation(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	userID := "admin-1"
	user := adminUser(userID)
	repo.On("GetByID", mock.Anything, userID).Return(user, nil)
	repo.On("LoadRoleNames", mock.Anything, userID).Return([]string{"administrador"}, nil)
	repo.On("CountActiveAdmins", mock.Anything).Return(int64(1), nil)

	_, err := svc.SetActive(context.Background(), userID, false)
	assert.ErrorIs(t, err, ErrLastAdmin, "should block deactivating the last active admin")
}

// U5-multi-admin-safe via SetActive path: two active admins → deactivation allowed
func TestSetActive_LastAdmin_AllowsWhenMultipleAdmins(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	userID := "admin-1"
	user := adminUser(userID)
	repo.On("GetByID", mock.Anything, userID).Return(user, nil)
	repo.On("LoadRoleNames", mock.Anything, userID).Return([]string{"administrador"}, nil)
	repo.On("CountActiveAdmins", mock.Anything).Return(int64(2), nil)
	repo.On("SetActive", mock.Anything, userID, false).Return(nil)

	// Refreshed user after deactivation
	deactivated := adminUser(userID)
	deactivated.Activo = false
	repo.On("GetByID", mock.Anything, userID).Return(deactivated, nil).Maybe()

	_, err := svc.SetActive(context.Background(), userID, false)
	assert.NoError(t, err)
}

// SetActive on a non-admin user always passes the guard (no admin concern).
func TestSetActive_NonAdmin_NeverTriggersGuard(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	userID := "user-1"
	user := regularUser(userID)
	repo.On("GetByID", mock.Anything, userID).Return(user, nil)
	repo.On("LoadRoleNames", mock.Anything, userID).Return([]string{"alumno"}, nil)
	repo.On("SetActive", mock.Anything, userID, false).Return(nil)

	deactivated := regularUser(userID)
	deactivated.Activo = false
	repo.On("GetByID", mock.Anything, userID).Return(deactivated, nil).Maybe()

	_, err := svc.SetActive(context.Background(), userID, false)
	assert.NoError(t, err)
	// CountActiveAdmins must NOT have been called (non-admin target)
	repo.AssertNotCalled(t, "CountActiveAdmins", mock.Anything)
}

// ── U4: Role-patch validation ──────────────────────────────────────────────────

// U4-invalid-role: unknown role in add → ErrInvalidRole
func TestPatchRoles_InvalidRole_InAdd(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	_, err := svc.PatchRoles(context.Background(), "user-1", []string{"superusuario"}, []string{})
	assert.ErrorIs(t, err, ErrInvalidRole)
}

// U4-invalid-role: unknown role in remove → ErrInvalidRole
func TestPatchRoles_InvalidRole_InRemove(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	_, err := svc.PatchRoles(context.Background(), "user-1", []string{}, []string{"superusuario"})
	assert.ErrorIs(t, err, ErrInvalidRole)
}

// U4-conflict: same role in both add and remove → ErrAddRemoveConflict
func TestPatchRoles_Conflict_SameRoleInAddAndRemove(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	_, err := svc.PatchRoles(context.Background(), "user-1", []string{"creador"}, []string{"creador"})
	assert.ErrorIs(t, err, ErrAddRemoveConflict)
}

// U4-idempotent-add: adding a role the user already holds calls AddRoles (repo handles idempotency).
func TestPatchRoles_Idempotent_Add(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	userID := "user-1"
	user := regularUser(userID) // already has "alumno"
	repo.On("GetByID", mock.Anything, userID).Return(user, nil)
	repo.On("LoadRoleNames", mock.Anything, userID).Return([]string{"alumno"}, nil)
	repo.On("AddRoles", mock.Anything, userID, []string{"alumno"}).Return(nil)
	repo.On("RemoveRoles", mock.Anything, userID, []string{}).Return(nil)

	_, err := svc.PatchRoles(context.Background(), userID, []string{"alumno"}, []string{})
	assert.NoError(t, err)
	repo.AssertCalled(t, "AddRoles", mock.Anything, userID, []string{"alumno"})
}

// U4: empty add and remove → no-op (no repo calls beyond the final GetDetail).
func TestPatchRoles_EmptyLists_NoOp(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	userID := "user-1"
	user := regularUser(userID)
	repo.On("GetByID", mock.Anything, userID).Return(user, nil)
	repo.On("LoadRoleNames", mock.Anything, userID).Return([]string{"alumno"}, nil)
	repo.On("AddRoles", mock.Anything, userID, []string{}).Return(nil)
	repo.On("RemoveRoles", mock.Anything, userID, []string{}).Return(nil)

	_, err := svc.PatchRoles(context.Background(), userID, []string{}, []string{})
	assert.NoError(t, err)
}

// ── U2/U3: GetDetail ──────────────────────────────────────────────────────────

// U2-found: user exists → returns populated UserDetailModel
func TestGetDetail_Found(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	userID := "user-42"
	user := regularUser(userID)
	repo.On("GetByID", mock.Anything, userID).Return(user, nil)
	repo.On("LoadRoleNames", mock.Anything, userID).Return([]string{"alumno"}, nil)

	detail, err := svc.GetDetail(context.Background(), userID)
	assert.NoError(t, err)
	assert.Equal(t, userID, detail.ID)
	assert.Equal(t, []string{"alumno"}, detail.Roles)
}

// U2-not-found: user does not exist → ErrUserNotFound
func TestGetDetail_NotFound(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	repo.On("GetByID", mock.Anything, "ghost-id").Return(nil, repository.ErrUserNotFound)

	_, err := svc.GetDetail(context.Background(), "ghost-id")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestList_DelegatesToRepo(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	filters := ListFilters{Q: "mart"}
	params := pagination.Params{Page: 1, Size: 20}

	user := regularUser("u1")
	repoPage := pagination.NewPage([]domain.User{*user}, 1, params)
	repo.On("List", mock.Anything, repository.ListFilters{Q: "mart"}, params).Return(repoPage, nil)
	repo.On("LoadRoleNames", mock.Anything, "u1").Return([]string{"alumno"}, nil)

	page, err := svc.List(context.Background(), filters, params)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	assert.Len(t, page.Items, 1)
	assert.Equal(t, "u1", page.Items[0].ID)
}

// ── CreateSupervision ─────────────────────────────────────────────────────────

// V2-self: supervisorID == empleadoID → ErrSelfSupervision
func TestCreateSupervision_SelfSupervision(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	_, err := svc.CreateSupervision(context.Background(), "user-1", "user-1")
	assert.ErrorIs(t, err, ErrSelfSupervision)
}

// V2-missing-user: supervisor does not exist → ErrUserNotFound
func TestCreateSupervision_SupervisorNotFound(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	repo.On("ExistsUser", mock.Anything, "missing-sup").Return(false, nil)

	_, err := svc.CreateSupervision(context.Background(), "missing-sup", "emp-1")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

// V2-missing-user: employee does not exist → ErrUserNotFound
func TestCreateSupervision_EmployeeNotFound(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	repo.On("ExistsUser", mock.Anything, "sup-1").Return(true, nil)
	repo.On("ExistsUser", mock.Anything, "missing-emp").Return(false, nil)

	_, err := svc.CreateSupervision(context.Background(), "sup-1", "missing-emp")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

// V2-duplicate: repo returns ErrSupervisionExists → propagated as service ErrSupervisionExists
func TestCreateSupervision_Duplicate_PropagatesConflict(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	repo.On("ExistsUser", mock.Anything, "sup-1").Return(true, nil)
	repo.On("ExistsUser", mock.Anything, "emp-1").Return(true, nil)
	repo.On("CreateSupervision", mock.Anything, "sup-1", "emp-1").Return(nil, repository.ErrSupervisionExists)

	_, err := svc.CreateSupervision(context.Background(), "sup-1", "emp-1")
	assert.ErrorIs(t, err, ErrSupervisionExists)
}

// Happy path: supervision created successfully
func TestCreateSupervision_Success(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	repo.On("ExistsUser", mock.Anything, "sup-1").Return(true, nil)
	repo.On("ExistsUser", mock.Anything, "emp-1").Return(true, nil)
	sv := &domain.Supervision{ID: "sv-id", SupervisorID: "sup-1", EmpleadoID: "emp-1"}
	repo.On("CreateSupervision", mock.Anything, "sup-1", "emp-1").Return(sv, nil)

	result, err := svc.CreateSupervision(context.Background(), "sup-1", "emp-1")
	assert.NoError(t, err)
	assert.Equal(t, "sv-id", result.ID)
}

// ListSupervisions delegates to repo and maps to SupervisionModels.
func TestListSupervisions_DelegatesToRepo(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	svs := []domain.Supervision{
		{ID: "sv-1", SupervisorID: "sup-1", EmpleadoID: "emp-1"},
		{ID: "sv-2", SupervisorID: "sup-1", EmpleadoID: "emp-2"},
	}
	repo.On("ListSupervisions", mock.Anything).Return(svs, nil)

	result, err := svc.ListSupervisions(context.Background())
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "sv-1", result[0].ID)
}

// DeleteSupervision: not found in repo → service ErrSupervisionNotFound (maps to 404).
// Uses ErrorIs to guard against future sentinel-remapping regressions.
// Resolves PR-A deviation #5: placeholder ErrUserNotFound is now ErrSupervisionNotFound.
func TestDeleteSupervision_NotFound(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	repo.On("DeleteSupervision", mock.Anything, "sv-999").Return(repository.ErrSupervisionNotFound)

	err := svc.DeleteSupervision(context.Background(), "sv-999")
	assert.ErrorIs(t, err, ErrSupervisionNotFound, "missing supervision should surface as ErrSupervisionNotFound (404)")
}

// DeleteSupervision: success path
func TestDeleteSupervision_Success(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	repo.On("DeleteSupervision", mock.Anything, "sv-1").Return(nil)

	err := svc.DeleteSupervision(context.Background(), "sv-1")
	assert.NoError(t, err)
}

// ── GetByID (existing method) — regression guard ───────────────────────────────

// GetByID (existing method) still works — regression guard.
func TestGetByID_Delegates(t *testing.T) {
	repo := &MockUsersRepository{}
	svc := newSvc(repo)

	user := regularUser("u1")
	repo.On("GetByID", mock.Anything, "u1").Return(user, nil)
	repo.On("LoadRoleNames", mock.Anything, "u1").Return([]string{"alumno"}, nil)

	summary, err := svc.GetByID(context.Background(), "u1")
	assert.NoError(t, err)
	assert.Equal(t, "u1", summary.ID)

	// Verify that errors.Is works for the re-wrapped sentinel
	_ = errors.Is(err, ErrUserNotFound)
}
