// Package service — white-box unit tests for the courses service.
// No build tag: runs with standard `make backend-test`.
//
// Strategy: inject a MockCoursesRepository (testify/mock) so no real DB is needed.
// These tests focus on the service invariants — especially the three load-bearing cases:
//   - LOAD-BEARING-A: Create forces estado=borrador regardless of any client-supplied value.
//   - LOAD-BEARING-B: UpdateByID checks ownership BEFORE the estado transition guard.
//   - Estado transition guard: only borrador and rechazado allow edits.
package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// ── MockCoursesRepository ─────────────────────────────────────────────────────

// MockCoursesRepository is a testify/mock implementation of repository.Repository.
// Defined here so it stays test-only; courses is a new domain with no importers.
type MockCoursesRepository struct {
	mock.Mock
}

func (m *MockCoursesRepository) Create(ctx context.Context, c *domain.Course) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *MockCoursesRepository) GetByID(ctx context.Context, id string) (*domain.Course, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Course), args.Error(1)
}

func (m *MockCoursesRepository) UpdateByID(ctx context.Context, id string, fields map[string]any) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *MockCoursesRepository) ListByCreator(ctx context.Context, creadorID string, p pagination.Params) (pagination.Page[domain.Course], error) {
	args := m.Called(ctx, creadorID, p)
	return args.Get(0).(pagination.Page[domain.Course]), args.Error(1)
}

func (m *MockCoursesRepository) UpdateEstado(ctx context.Context, id string, estado domain.Estado) error {
	args := m.Called(ctx, id, estado)
	return args.Error(0)
}

// ── Fixtures ────────────────────────────────────────────────────────────────────

func courseWith(creadorID string, estado domain.Estado) *domain.Course {
	return &domain.Course{
		ID:          uuid.New().String(),
		CreadorID:   creadorID,
		Titulo:      "Test Course",
		Descripcion: "Test Desc",
		Estado:      estado,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func newSvc(repo repository.Repository) *serviceImpl {
	return New(repo).(*serviceImpl)
}

// ── Create tests ───────────────────────────────────────────────────────────────

// LOAD-BEARING-A: Create must FORCE estado=borrador, even when caller provides a
// different value (the DTO doesn't expose estado, but the service must set it).
// Satisfies: REQ-CREATE "Client-sent estado silently overridden", AC1.
func TestCreate_ForcesEstadoBorrador(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()

	// Capture what repo.Create receives to verify the Estado field.
	var captured *domain.Course
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Course")).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*domain.Course)
		}).
		Return(nil)

	req := CreateRequest{Titulo: "Go avanzado", Descripcion: "Curso de Go"}
	result, err := svc.Create(context.Background(), creadorID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// The returned model MUST have estado=borrador.
	assert.Equal(t, domain.EstadoBorrador, result.Estado,
		"[LOAD-BEARING-A] returned model.Estado must be borrador")
	// The domain object sent to repo.Create MUST also have estado=borrador.
	require.NotNil(t, captured)
	assert.Equal(t, domain.EstadoBorrador, captured.Estado,
		"[LOAD-BEARING-A] repo.Create received Estado must be borrador")
	assert.Equal(t, creadorID, captured.CreadorID,
		"creadorID must come from the service argument, not from the request")
	repo.AssertExpectations(t)
}

func TestCreate_SetsCreadorID(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Course")).Return(nil)

	req := CreateRequest{Titulo: "Mi curso", Descripcion: "Desc"}
	result, err := svc.Create(context.Background(), creadorID, req)

	assert.NoError(t, err)
	assert.Equal(t, creadorID, result.CreadorID)
	repo.AssertExpectations(t)
}

// ── GetByID tests ──────────────────────────────────────────────────────────────

func TestGetByID_Owner_ReturnsModel(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	c := courseWith(creadorID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, c.ID).Return(c, nil)

	result, err := svc.GetByID(context.Background(), c.ID, creadorID)

	assert.NoError(t, err)
	assert.Equal(t, c.ID, result.ID)
	repo.AssertExpectations(t)
}

func TestGetByID_ForeignCreador_ReturnsErrNotOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	requesterID := uuid.New().String() // different creador
	c := courseWith(ownerID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, c.ID).Return(c, nil)

	_, err := svc.GetByID(context.Background(), c.ID, requesterID)

	assert.ErrorIs(t, err, ErrNotOwner,
		"non-owner GET must return ErrNotOwner (handler maps this to 404)")
	repo.AssertExpectations(t)
}

func TestGetByID_NotFound_ReturnsErrCourseNotFound(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	courseID := uuid.New().String()
	creadorID := uuid.New().String()
	repo.On("GetByID", mock.Anything, courseID).Return(nil, repository.ErrCourseNotFound)

	_, err := svc.GetByID(context.Background(), courseID, creadorID)

	assert.ErrorIs(t, err, ErrCourseNotFound)
	repo.AssertExpectations(t)
}

// ── UpdateByID tests ───────────────────────────────────────────────────────────

// LOAD-BEARING-B: Ownership MUST be checked BEFORE the transition guard.
// A non-owner editing an aprobado course must get ErrNotOwner, NOT ErrInvalidTransition.
// Satisfies: REQ-DIVERGENCE, design §4 "ownership BEFORE transition".
func TestUpdateByID_OwnershipBeforeState_NonOwnerAprobado(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	requesterID := uuid.New().String() // NOT the owner
	c := courseWith(ownerID, domain.EstadoAprobado)
	repo.On("GetByID", mock.Anything, c.ID).Return(c, nil)

	titulo := "Nuevo titulo"
	req := UpdateRequest{Titulo: &titulo}
	_, err := svc.UpdateByID(context.Background(), c.ID, requesterID, req)

	assert.ErrorIs(t, err, ErrNotOwner,
		"[LOAD-BEARING-B] non-owner of aprobado course must get ErrNotOwner, not ErrInvalidTransition")
	assert.False(t, isInvalidTransition(err),
		"[LOAD-BEARING-B] must NOT be ErrInvalidTransition — ownership outranks state")
	repo.AssertExpectations(t)
}

// isInvalidTransition is a test helper to confirm the error is not ErrInvalidTransition.
func isInvalidTransition(err error) bool {
	return err == ErrInvalidTransition
}

func TestUpdateByID_TransitionGuard_EnRevision(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	c := courseWith(creadorID, domain.EstadoEnRevision)
	repo.On("GetByID", mock.Anything, c.ID).Return(c, nil)

	titulo := "X"
	req := UpdateRequest{Titulo: &titulo}
	_, err := svc.UpdateByID(context.Background(), c.ID, creadorID, req)

	assert.ErrorIs(t, err, ErrInvalidTransition,
		"own course with estado=en_revision must return ErrInvalidTransition")
	repo.AssertExpectations(t)
}

func TestUpdateByID_TransitionGuard_Aprobado(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	c := courseWith(creadorID, domain.EstadoAprobado) // IS the owner
	repo.On("GetByID", mock.Anything, c.ID).Return(c, nil)

	titulo := "X"
	req := UpdateRequest{Titulo: &titulo}
	_, err := svc.UpdateByID(context.Background(), c.ID, creadorID, req)

	assert.ErrorIs(t, err, ErrInvalidTransition,
		"owner editing aprobado course must return ErrInvalidTransition (ownership passes, state blocks)")
	repo.AssertExpectations(t)
}

func TestUpdateByID_Borrador_OK(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	c := courseWith(creadorID, domain.EstadoBorrador)

	// GetByID is called TWICE: first for ownership+state check, second inside re-read GetByID.
	repo.On("GetByID", mock.Anything, c.ID).Return(c, nil)
	repo.On("UpdateByID", mock.Anything, c.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	titulo := "Nuevo titulo"
	req := UpdateRequest{Titulo: &titulo}
	result, err := svc.UpdateByID(context.Background(), c.ID, creadorID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	repo.AssertExpectations(t)
}

func TestUpdateByID_Rechazado_OK(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	c := courseWith(creadorID, domain.EstadoRechazado)

	repo.On("GetByID", mock.Anything, c.ID).Return(c, nil)
	repo.On("UpdateByID", mock.Anything, c.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	desc := "Revisada"
	req := UpdateRequest{Descripcion: &desc}
	result, err := svc.UpdateByID(context.Background(), c.ID, creadorID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	repo.AssertExpectations(t)
}

// TestUpdateByID_PartialUpdate_NilTituloUntouched verifies that a nil Titulo
// in UpdateRequest does NOT appear in the fields map sent to the repository.
// Satisfies: design §4 partial update field-map.
func TestUpdateByID_PartialUpdate_NilTituloUntouched(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	c := courseWith(creadorID, domain.EstadoBorrador)

	var capturedFields map[string]any
	repo.On("GetByID", mock.Anything, c.ID).Return(c, nil)
	repo.On("UpdateByID", mock.Anything, c.ID, mock.AnythingOfType("map[string]interface {}")).
		Run(func(args mock.Arguments) {
			capturedFields = args.Get(2).(map[string]any)
		}).
		Return(nil)

	desc := "Nueva descripcion"
	req := UpdateRequest{Titulo: nil, Descripcion: &desc} // Titulo is nil
	_, err := svc.UpdateByID(context.Background(), c.ID, creadorID, req)

	assert.NoError(t, err)
	require.NotNil(t, capturedFields)
	_, hasTitulo := capturedFields["titulo"]
	assert.False(t, hasTitulo, "nil Titulo must NOT appear in the fields map")
	_, hasDesc := capturedFields["descripcion"]
	assert.True(t, hasDesc, "non-nil Descripcion MUST appear in the fields map")
	repo.AssertExpectations(t)
}

// ── ListByCreator tests ────────────────────────────────────────────────────────

func TestListByCreator_DelegatesAndMaps(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	p := pagination.Params{Page: 1, Size: 20}

	domainCourses := []domain.Course{
		*courseWith(creadorID, domain.EstadoBorrador),
		*courseWith(creadorID, domain.EstadoRechazado),
	}
	domainPage := pagination.Page[domain.Course]{
		Items:      domainCourses,
		Page:       1,
		Size:       20,
		Total:      2,
		TotalPages: 1,
	}
	repo.On("ListByCreator", mock.Anything, creadorID, p).Return(domainPage, nil)

	page, err := svc.ListByCreator(context.Background(), creadorID, p)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)
	assert.Equal(t, 1, page.Page)
	assert.Equal(t, 2, len(page.Items))
	repo.AssertExpectations(t)
}

// TestCreate_ValidEstadoConst verifies domain.Estado.Valid() covers all four states.
// This exercises the domain.Estado.Valid() method (task 1.2 cross-ref).
func TestCreate_ValidEstadoConst(t *testing.T) {
	assert.True(t, domain.EstadoBorrador.Valid())
	assert.True(t, domain.EstadoEnRevision.Valid())
	assert.True(t, domain.EstadoAprobado.Valid())
	assert.True(t, domain.EstadoRechazado.Valid())
	assert.False(t, domain.Estado("invalid").Valid())
}
