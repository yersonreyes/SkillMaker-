// Package service — white-box unit tests for the courses service.
// No build tag: runs with standard `make backend-test`.
//
// Strategy: inject a MockCoursesRepository (testify/mock) so no real DB is needed.
// These tests focus on the service invariants — especially the three load-bearing cases:
//   - LOAD-BEARING-A: Create forces estado=borrador regardless of any client-supplied value.
//   - LOAD-BEARING-B: UpdateByID checks ownership BEFORE the estado transition guard.
//   - Estado transition guard: only borrador and rechazado allow edits.
//
// C2.2 additions (LB-1/LB-2):
//   - LB-1: validateVideoURL cross-validates URL host against proveedor.
//   - LB-2: loadOwnedSection/Video traversal returns ErrNotOwner for non-owner.
//
// C2.3 additions (material methods):
//   - PresignUpload: size/MIME validation, owner gate, estado gate, key prefix.
//   - ConfirmUpload: re-validation (dual validation), key prefix guard.
//   - ListMaterials: owner gate.
//   - PresignDownload: owner gate, material not found.
//   - DeleteMaterial: best-effort delete (load-bearing), owner gate, repo delete before store.Delete.
package service

import (
	"context"
	"io"
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

func (m *MockCoursesRepository) CreateSection(ctx context.Context, s *domain.Section) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockCoursesRepository) GetSectionByID(ctx context.Context, id string) (*domain.Section, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Section), args.Error(1)
}

func (m *MockCoursesRepository) ListSectionsByCourse(ctx context.Context, courseID string) ([]domain.Section, error) {
	args := m.Called(ctx, courseID)
	return args.Get(0).([]domain.Section), args.Error(1)
}

func (m *MockCoursesRepository) UpdateSection(ctx context.Context, id string, fields map[string]any) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *MockCoursesRepository) DeleteSection(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCoursesRepository) ReorderSections(ctx context.Context, courseID string, ids []string) error {
	args := m.Called(ctx, courseID, ids)
	return args.Error(0)
}

func (m *MockCoursesRepository) CreateVideo(ctx context.Context, v *domain.Video) error {
	args := m.Called(ctx, v)
	return args.Error(0)
}

func (m *MockCoursesRepository) GetVideoByID(ctx context.Context, id string) (*domain.Video, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Video), args.Error(1)
}

func (m *MockCoursesRepository) ListVideosBySection(ctx context.Context, sectionID string) ([]domain.Video, error) {
	args := m.Called(ctx, sectionID)
	return args.Get(0).([]domain.Video), args.Error(1)
}

func (m *MockCoursesRepository) UpdateVideo(ctx context.Context, id string, fields map[string]any) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *MockCoursesRepository) DeleteVideo(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCoursesRepository) HasContent(ctx context.Context, courseID string) (bool, error) {
	args := m.Called(ctx, courseID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCoursesRepository) CreateMaterial(ctx context.Context, mat *domain.Material) error {
	args := m.Called(ctx, mat)
	return args.Error(0)
}

func (m *MockCoursesRepository) GetMaterialByID(ctx context.Context, id string) (*domain.Material, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Material), args.Error(1)
}

func (m *MockCoursesRepository) ListMaterialsByVideo(ctx context.Context, videoID string) ([]domain.Material, error) {
	args := m.Called(ctx, videoID)
	return args.Get(0).([]domain.Material), args.Error(1)
}

func (m *MockCoursesRepository) ListMaterialsByCourseVideos(ctx context.Context, courseID string) ([]domain.Material, error) {
	args := m.Called(ctx, courseID)
	return args.Get(0).([]domain.Material), args.Error(1)
}

func (m *MockCoursesRepository) GetMaterialOwnership(ctx context.Context, materialID string) (courseID, creadorID, estado string, err error) {
	args := m.Called(ctx, materialID)
	return args.String(0), args.String(1), args.String(2), args.Error(3)
}

func (m *MockCoursesRepository) ResolveVideoCourse(ctx context.Context, videoID string) (courseID, creadorID, estado string, err error) {
	args := m.Called(ctx, videoID)
	return args.String(0), args.String(1), args.String(2), args.Error(3)
}

func (m *MockCoursesRepository) DeleteMaterial(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockCoursesRepository) ListCategorias(ctx context.Context) ([]domain.Categoria, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Categoria), args.Error(1)
}

func (m *MockCoursesRepository) GetCourseCategorias(ctx context.Context, courseID string) ([]domain.Categoria, error) {
	args := m.Called(ctx, courseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Categoria), args.Error(1)
}

func (m *MockCoursesRepository) SetCourseCategorias(ctx context.Context, courseID string, ids []string) error {
	args := m.Called(ctx, courseID, ids)
	return args.Error(0)
}

func (m *MockCoursesRepository) CategoriasExist(ctx context.Context, ids []string) (bool, error) {
	args := m.Called(ctx, ids)
	return args.Bool(0), args.Error(1)
}

func (m *MockCoursesRepository) ListCategoriasForCourses(ctx context.Context, courseIDs []string) (map[string][]domain.Categoria, error) {
	args := m.Called(ctx, courseIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]domain.Categoria), args.Error(1)
}

// ── T-1.6 additions: UpdateEstadoPublicado + ListByEstado ─────────────────────

func (m *MockCoursesRepository) UpdateEstadoPublicado(ctx context.Context, id string, estado domain.Estado, publicadoEn time.Time) error {
	args := m.Called(ctx, id, estado, publicadoEn)
	return args.Error(0)
}

func (m *MockCoursesRepository) ListByEstado(ctx context.Context, estado domain.Estado) ([]domain.Course, error) {
	args := m.Called(ctx, estado)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Course), args.Error(1)
}

// ── C2.4 additions: catalog + enrollment methods ─────────────────────────────

func (m *MockCoursesRepository) ListApproved(ctx context.Context, p pagination.Params, f repository.CatalogFilter) (pagination.Page[repository.CatalogCourseModel], error) {
	args := m.Called(ctx, p, f)
	return args.Get(0).(pagination.Page[repository.CatalogCourseModel]), args.Error(1)
}

func (m *MockCoursesRepository) GetApprovedDetail(ctx context.Context, courseID string) (*repository.CatalogCourseModel, error) {
	args := m.Called(ctx, courseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.CatalogCourseModel), args.Error(1)
}

func (m *MockCoursesRepository) CreateEnrollment(ctx context.Context, userID, courseID string) error {
	args := m.Called(ctx, userID, courseID)
	return args.Error(0)
}

func (m *MockCoursesRepository) IsEnrolled(ctx context.Context, userID, courseID string) (bool, error) {
	args := m.Called(ctx, userID, courseID)
	return args.Bool(0), args.Error(1)
}

func (m *MockCoursesRepository) ListEnrollmentsByUser(ctx context.Context, userID string) ([]repository.EnrollmentWithCourseModel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.EnrollmentWithCourseModel), args.Error(1)
}

func (m *MockCoursesRepository) MarkCompleted(ctx context.Context, userID, courseID string) error {
	args := m.Called(ctx, userID, courseID)
	return args.Error(0)
}

// ── course-player-progress additions (migration 0014) ─────────────────────────

func (m *MockCoursesRepository) UpsertVideoProgress(ctx context.Context, userID, videoID string, completado bool, lastPositionS int) error {
	args := m.Called(ctx, userID, videoID, completado, lastPositionS)
	return args.Error(0)
}

func (m *MockCoursesRepository) ListVideoProgressByUserAndCourse(ctx context.Context, userID, courseID string) ([]repository.VideoProgressRow, error) {
	args := m.Called(ctx, userID, courseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.VideoProgressRow), args.Error(1)
}

// ── MockStorageClient ─────────────────────────────────────────────────────────

// mockStorageClient is a minimal mock for storage.Client using func fields.
// This avoids testify/mock overhead for the storage interface (5 methods).
type mockStorageClient struct {
	PresignPutURLFn func(ctx context.Context, key string, ttl time.Duration) (string, error)
	PresignGetURLFn func(ctx context.Context, key string, ttl time.Duration) (string, error)
	DeleteFn        func(ctx context.Context, key string) error
	PingFn          func(ctx context.Context) error
	PutObjectFn     func(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error
	// Track calls for assertion.
	deleteCalls    []string
	putCalls       []string
	getCalls       []string
	putObjectCalls []string
}

func (m *mockStorageClient) PresignPutURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	m.putCalls = append(m.putCalls, key)
	if m.PresignPutURLFn != nil {
		return m.PresignPutURLFn(ctx, key, ttl)
	}
	return "https://minio-presigned-put/" + key, nil
}

func (m *mockStorageClient) PresignGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	m.getCalls = append(m.getCalls, key)
	if m.PresignGetURLFn != nil {
		return m.PresignGetURLFn(ctx, key, ttl)
	}
	return "https://minio-presigned-get/" + key, nil
}

func (m *mockStorageClient) Delete(ctx context.Context, key string) error {
	m.deleteCalls = append(m.deleteCalls, key)
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, key)
	}
	return nil
}

func (m *mockStorageClient) Ping(ctx context.Context) error {
	if m.PingFn != nil {
		return m.PingFn(ctx)
	}
	return nil
}

func (m *mockStorageClient) PutObject(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	m.putObjectCalls = append(m.putObjectCalls, objectName)
	if m.PutObjectFn != nil {
		return m.PutObjectFn(ctx, objectName, reader, size, contentType)
	}
	return nil
}

// newSvcWithStore creates a serviceImpl with the given storage client.
func newSvcWithStore(repo *MockCoursesRepository, store *mockStorageClient) *serviceImpl {
	return New(repo, store, 15*time.Minute, 52_428_800).(*serviceImpl)
}

// materialWith creates a domain.Material fixture for testing.
// course-structure-v2: keyed by videoID (not courseID).
func materialWith(videoID string) *domain.Material {
	return &domain.Material{
		ID:          uuid.New().String(),
		VideoID:     videoID,
		Titulo:      "test-document.pdf",
		StorageKey:  "courses/cid/videos/" + videoID + "/materials/uuid-test-document.pdf",
		MimeType:    "application/pdf",
		TamanoBytes: 1024,
		CreatedAt:   time.Now(),
	}
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

func sectionWith(courseID string) *domain.Section {
	return &domain.Section{
		ID:        uuid.New().String(),
		CourseID:  courseID,
		Titulo:    "Test Section",
		Orden:     0,
		CreatedAt: time.Now(),
	}
}

func videoWith(sectionID string) *domain.Video {
	return &domain.Video{
		ID:        uuid.New().String(),
		SectionID: sectionID,
		Titulo:    "Test Video",
		URL:       "https://www.youtube.com/watch?v=abc123",
		Proveedor: "youtube",
		DuracionS: 120,
		Orden:     0,
		CreatedAt: time.Now(),
	}
}

// newSvc creates a serviceImpl with a nil storage client for tests that do not
// exercise storage methods. Material-specific tests use newSvcWithStore instead.
func newSvc(repo repository.Repository) *serviceImpl {
	return New(repo, nil, 15*time.Minute, 52_428_800).(*serviceImpl)
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

// ── validateVideoURL tests (LB-1) ─────────────────────────────────────────────

// TestValidateVideoURL_TableDriven is the LOAD-BEARING test for url/proveedor cross-validation.
// Spec: VID-1-C, VID-1-D, VID-1-E.
func TestValidateVideoURL_TableDriven(t *testing.T) {
	cases := []struct {
		name      string
		url       string
		proveedor string
		wantErr   bool
	}{
		// ── Valid cases ──────────────────────────────────────────────────────
		{
			name:      "YouTube watch URL",
			url:       "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			proveedor: "youtube",
			wantErr:   false,
		},
		{
			name:      "YouTube short URL (youtu.be)",
			url:       "https://youtu.be/dQw4w9WgXcQ",
			proveedor: "youtube",
			wantErr:   false,
		},
		{
			name:      "Vimeo URL",
			url:       "https://vimeo.com/123456789",
			proveedor: "vimeo",
			wantErr:   false,
		},
		{
			name:      "YouTube without www",
			url:       "https://youtube.com/watch?v=abc",
			proveedor: "youtube",
			wantErr:   false,
		},
		// ── Invalid cases ─────────────────────────────────────────────────
		{
			name:      "[LB-1] Vimeo URL with proveedor=youtube → mismatch",
			url:       "https://vimeo.com/123",
			proveedor: "youtube",
			wantErr:   true,
		},
		{
			name:      "[LB-1] YouTube URL with proveedor=vimeo → mismatch",
			url:       "https://www.youtube.com/watch?v=abc",
			proveedor: "vimeo",
			wantErr:   true,
		},
		{
			name:      "Dailymotion URL with proveedor=youtube → invalid host",
			url:       "https://dailymotion.com/video/x7",
			proveedor: "youtube",
			wantErr:   true,
		},
		{
			name:      "Dailymotion URL with proveedor=vimeo → invalid host",
			url:       "https://dailymotion.com/video/x7",
			proveedor: "vimeo",
			wantErr:   true,
		},
		{
			name:      "youtu.be URL with proveedor=vimeo → mismatch",
			url:       "https://youtu.be/dQw4w9WgXcQ",
			proveedor: "vimeo",
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateVideoURL(tc.url, tc.proveedor)
			if tc.wantErr {
				assert.ErrorIs(t, err, ErrURLProviderMismatch,
					"expected ErrURLProviderMismatch for url=%s proveedor=%s", tc.url, tc.proveedor)
			} else {
				assert.NoError(t, err,
					"expected no error for url=%s proveedor=%s", tc.url, tc.proveedor)
			}
		})
	}
}

// ── Ownership traversal tests (LB-2) ─────────────────────────────────────────

// TestLoadOwnedSection_NonOwner verifies that a non-owner gets ErrNotOwner.
// LB-2: ownership traversal — section → course → ErrNotOwner.
// Spec: SEC-1-B, VID-2-C.
func TestLoadOwnedSection_NonOwner_ReturnsErrNotOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	sec := sectionWith(course.ID)

	repo.On("GetSectionByID", mock.Anything, sec.ID).Return(sec, nil)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	// foreignID attempts to access ownerID's section.
	_, _, err := svc.loadOwnedSection(context.Background(), sec.ID, foreignID)

	assert.ErrorIs(t, err, ErrNotOwner,
		"[LB-2] non-owner section traversal must return ErrNotOwner")
	repo.AssertExpectations(t)
}

// TestLoadOwnedVideo_NonOwner verifies that a non-owner gets ErrNotOwner via video traversal.
// LB-2: ownership traversal — video → section → course → ErrNotOwner.
// Spec: VID-1-F, VID-2-C, VID-3-B.
func TestLoadOwnedVideo_NonOwner_ReturnsErrNotOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	sec := sectionWith(course.ID)
	v := videoWith(sec.ID)

	repo.On("GetVideoByID", mock.Anything, v.ID).Return(v, nil)
	repo.On("GetSectionByID", mock.Anything, sec.ID).Return(sec, nil)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, _, _, err := svc.loadOwnedVideo(context.Background(), v.ID, foreignID)

	assert.ErrorIs(t, err, ErrNotOwner,
		"[LB-2] non-owner video traversal must return ErrNotOwner")
	repo.AssertExpectations(t)
}

// ── Estado gating tests ───────────────────────────────────────────────────────

// TestCreateSection_EnRevision_ReturnsErrInvalidTransition verifies estado gating.
// Spec: SEC-1-E.
func TestCreateSection_EnRevision_ReturnsErrInvalidTransition(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	course := courseWith(creadorID, domain.EstadoEnRevision)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.CreateSection(context.Background(), creadorID, SectionCreateRequest{
		CourseID: course.ID,
		Titulo:   "Intro",
	})

	assert.ErrorIs(t, err, ErrInvalidTransition, "en_revision course must block section creation")
	repo.AssertExpectations(t)
}

// TestCreateSection_Aprobado_ReturnsErrInvalidTransition verifies that creating
// a section on an aprobado course returns ErrInvalidTransition.
func TestCreateSection_Aprobado_ReturnsErrInvalidTransition(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	course := courseWith(creadorID, domain.EstadoAprobado)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.CreateSection(context.Background(), creadorID, SectionCreateRequest{
		CourseID: course.ID,
		Titulo:   "Intro",
	})

	assert.ErrorIs(t, err, ErrInvalidTransition, "aprobado course must block section creation")
	repo.AssertExpectations(t)
}

// TestCreateSection_Borrador_OK verifies that creating a section on a borrador course succeeds.
func TestCreateSection_Borrador_OK(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	course := courseWith(creadorID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
	repo.On("CreateSection", mock.Anything, mock.AnythingOfType("*domain.Section")).Return(nil)

	result, err := svc.CreateSection(context.Background(), creadorID, SectionCreateRequest{
		CourseID: course.ID,
		Titulo:   "Intro",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, course.ID, result.CourseID)
	repo.AssertExpectations(t)
}

// ── ReorderSections validation tests ─────────────────────────────────────────

// TestReorderSections_ForeignID_ReturnsErrInvalidReorderSet verifies that a foreign section
// ID returns ErrInvalidReorderSet (→ 400), not ErrInvalidTransition (→ 409).
// Spec: ROR-1-B.
func TestReorderSections_ForeignID_ReturnsErrInvalidReorderSet(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	course := courseWith(creadorID, domain.EstadoBorrador)
	sec1 := sectionWith(course.ID)
	sec2 := sectionWith(course.ID)
	foreignID := uuid.New().String()

	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
	repo.On("ListSectionsByCourse", mock.Anything, course.ID).Return([]domain.Section{*sec1, *sec2}, nil)

	err := svc.ReorderSections(context.Background(), course.ID, creadorID, []string{sec1.ID, sec2.ID, foreignID})
	assert.ErrorIs(t, err, ErrInvalidReorderSet, "foreign section ID must return ErrInvalidReorderSet (ROR-1-B)")
	repo.AssertExpectations(t)
}

// TestReorderSections_IncompleteSet_ReturnsErrInvalidReorderSet verifies that an incomplete
// section set returns ErrInvalidReorderSet (→ 400), not ErrInvalidTransition (→ 409).
// Spec: ROR-1-C.
func TestReorderSections_IncompleteSet_ReturnsErrInvalidReorderSet(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	creadorID := uuid.New().String()
	course := courseWith(creadorID, domain.EstadoBorrador)
	sec1 := sectionWith(course.ID)
	sec2 := sectionWith(course.ID)
	sec3 := sectionWith(course.ID)

	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
	repo.On("ListSectionsByCourse", mock.Anything, course.ID).Return([]domain.Section{*sec1, *sec2, *sec3}, nil)

	// Only passing 2 of 3 section IDs — incomplete.
	err := svc.ReorderSections(context.Background(), course.ID, creadorID, []string{sec1.ID, sec2.ID})
	assert.ErrorIs(t, err, ErrInvalidReorderSet, "incomplete section set must return ErrInvalidReorderSet (ROR-1-C)")
	repo.AssertExpectations(t)
}

// ── UpdateVideo URL re-validation tests ──────────────────────────────────────

// TestUpdateVideo_UrlProviderMismatch_ReturnsErrURLProviderMismatch tests spec VID-2-B.
func TestUpdateVideo_UrlProviderMismatch_ReturnsError(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	sec := sectionWith(course.ID)
	v := videoWith(sec.ID) // originally youtube

	repo.On("GetVideoByID", mock.Anything, v.ID).Return(v, nil)
	repo.On("GetSectionByID", mock.Anything, sec.ID).Return(sec, nil)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	vimeoURL := "https://vimeo.com/999"
	// Updating URL to vimeo but keeping proveedor=youtube → mismatch.
	_, err := svc.UpdateVideo(context.Background(), v.ID, ownerID, VideoUpdateRequest{
		URL:       &vimeoURL,
		Proveedor: nil, // keeps original youtube
	})

	assert.ErrorIs(t, err, ErrURLProviderMismatch, "vimeo URL with proveedor=youtube must return ErrURLProviderMismatch")
	repo.AssertExpectations(t)
}

// ── DeleteSection tests ───────────────────────────────────────────────────────

// TestDeleteSection_Owner_OK verifies that an owner can delete their section.
func TestDeleteSection_Owner_OK(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	sec := sectionWith(course.ID)

	repo.On("GetSectionByID", mock.Anything, sec.ID).Return(sec, nil)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
	repo.On("DeleteSection", mock.Anything, sec.ID).Return(nil)

	err := svc.DeleteSection(context.Background(), sec.ID, ownerID)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// TestDeleteSection_NonOwner_ReturnsErrNotOwner verifies that a non-owner deleting a section gets ErrNotOwner.
func TestDeleteSection_NonOwner_ReturnsErrNotOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	sec := sectionWith(course.ID)

	repo.On("GetSectionByID", mock.Anything, sec.ID).Return(sec, nil)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	err := svc.DeleteSection(context.Background(), sec.ID, foreignID)
	assert.ErrorIs(t, err, ErrNotOwner)
	repo.AssertExpectations(t)
}

// ── ListSections tests ────────────────────────────────────────────────────────

// TestListSections_DelegatesAndMaps verifies ListSections maps domain slices.
func TestListSections_DelegatesAndMaps(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	courseID := uuid.New().String()
	s1 := sectionWith(courseID)
	s2 := sectionWith(courseID)

	repo.On("ListSectionsByCourse", mock.Anything, courseID).Return([]domain.Section{*s1, *s2}, nil)

	result, err := svc.ListSections(context.Background(), courseID)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

// ── GetSectionByID tests ──────────────────────────────────────────────────────

// TestGetSectionByID_NotFound_ReturnsErrSectionNotFound verifies sentinel wrapping.
func TestGetSectionByID_NotFound_ReturnsErrSectionNotFound(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	sectionID := uuid.New().String()
	repo.On("GetSectionByID", mock.Anything, sectionID).Return(nil, repository.ErrSectionNotFound)

	_, err := svc.GetSectionByID(context.Background(), sectionID)
	assert.ErrorIs(t, err, ErrSectionNotFound)
	repo.AssertExpectations(t)
}

// ── DeleteVideo tests ─────────────────────────────────────────────────────────

// TestDeleteVideo_Owner_OK verifies owner can delete their video.
func TestDeleteVideo_Owner_OK(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	sec := sectionWith(course.ID)
	v := videoWith(sec.ID)

	repo.On("GetVideoByID", mock.Anything, v.ID).Return(v, nil)
	repo.On("GetSectionByID", mock.Anything, sec.ID).Return(sec, nil)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
	repo.On("DeleteVideo", mock.Anything, v.ID).Return(nil)

	err := svc.DeleteVideo(context.Background(), v.ID, ownerID)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// ── ListVideos tests ──────────────────────────────────────────────────────────

// TestListVideos_DelegatesAndMaps verifies ListVideos maps domain slices.
func TestListVideos_DelegatesAndMaps(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	sectionID := uuid.New().String()
	v1 := videoWith(sectionID)
	v2 := videoWith(sectionID)

	repo.On("ListVideosBySection", mock.Anything, sectionID).Return([]domain.Video{*v1, *v2}, nil)

	result, err := svc.ListVideos(context.Background(), sectionID)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

// ── HasContent tests ──────────────────────────────────────────────────────────

// TestHasContent_Owner_ReturnsTrue verifies HasContent is true when course has videos.
// Spec: HC-1-A.
func TestHasContent_Owner_ReturnsTrue(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)

	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
	repo.On("HasContent", mock.Anything, course.ID).Return(true, nil)

	result, err := svc.HasContent(context.Background(), course.ID, ownerID)
	assert.NoError(t, err)
	assert.True(t, result)
	repo.AssertExpectations(t)
}

// TestHasContent_NonOwner_ReturnsErrNotOwner verifies owner gate.
func TestHasContent_NonOwner_ReturnsErrNotOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)

	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.HasContent(context.Background(), course.ID, foreignID)
	assert.ErrorIs(t, err, ErrNotOwner)
	repo.AssertExpectations(t)
}

// ── CreateVideo happy-path test ───────────────────────────────────────────────

// TestCreateVideo_ValidYouTube_OK verifies CreateVideo succeeds with valid YouTube URL.
// Spec: VID-1-A.
func TestCreateVideo_ValidYouTube_OK(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	sec := sectionWith(course.ID)

	repo.On("GetSectionByID", mock.Anything, sec.ID).Return(sec, nil)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
	repo.On("ListVideosBySection", mock.Anything, sec.ID).Return([]domain.Video{}, nil)
	repo.On("CreateVideo", mock.Anything, mock.AnythingOfType("*domain.Video")).Return(nil)

	result, err := svc.CreateVideo(context.Background(), ownerID, VideoCreateRequest{
		SectionID: sec.ID,
		Titulo:    "Test Video",
		URL:       "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		Proveedor: "youtube",
		DuracionS: 120,
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "youtube", result.Proveedor)
	repo.AssertExpectations(t)
}

// ── UpdateSection happy-path test ─────────────────────────────────────────────

// TestUpdateSection_Owner_OK verifies that an owner can update a section titulo.
func TestUpdateSection_Owner_OK(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	sec := sectionWith(course.ID)

	repo.On("GetSectionByID", mock.Anything, sec.ID).Return(sec, nil).Once()
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
	repo.On("UpdateSection", mock.Anything, sec.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)
	// Re-read after update.
	repo.On("GetSectionByID", mock.Anything, sec.ID).Return(sec, nil).Once()

	newTitle := "Updated Section"
	result, err := svc.UpdateSection(context.Background(), sec.ID, ownerID, SectionUpdateRequest{
		Titulo: &newTitle,
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	repo.AssertExpectations(t)
}

// ── ReorderSections happy-path test ───────────────────────────────────────────

// TestReorderSections_ValidSet_OK verifies that a valid reorder succeeds.
// Spec: ROR-1-A.
func TestReorderSections_ValidSet_OK(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	sec1 := sectionWith(course.ID)
	sec2 := sectionWith(course.ID)
	sec3 := sectionWith(course.ID)

	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
	repo.On("ListSectionsByCourse", mock.Anything, course.ID).Return([]domain.Section{*sec1, *sec2, *sec3}, nil)
	repo.On("ReorderSections", mock.Anything, course.ID, []string{sec3.ID, sec1.ID, sec2.ID}).Return(nil)

	err := svc.ReorderSections(context.Background(), course.ID, ownerID, []string{sec3.ID, sec1.ID, sec2.ID})
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// ── PresignUpload tests (course-structure-v2: videoID-based) ─────────────────

// TestPresignUpload_HappyPath verifies successful presign returns key with correct video prefix.
// course-structure-v2: key now includes videoID in path.
func TestPresignUpload_HappyPath(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	// Mock ResolveVideoCourse: video → course chain.
	repo.On("ResolveVideoCourse", mock.Anything, videoID).Return(courseID, ownerID, "borrador", nil)

	result, err := svc.PresignUpload(context.Background(), videoID, ownerID, PresignInput{
		Nombre:      "documento.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 1024,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.UploadURL)
	assert.Contains(t, result.Key, "courses/"+courseID+"/videos/"+videoID+"/materials/",
		"key must contain courses/{courseID}/videos/{videoID}/materials/")
	assert.False(t, result.ExpiresAt.IsZero())
	repo.AssertExpectations(t)
}

// TestPresignUpload_FileTooLarge verifies ErrFileTooLarge when size exceeds max.
func TestPresignUpload_FileTooLarge(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	repo.On("ResolveVideoCourse", mock.Anything, videoID).Return(courseID, ownerID, "borrador", nil)

	_, err := svc.PresignUpload(context.Background(), videoID, ownerID, PresignInput{
		Nombre:      "big.zip",
		ContentType: "application/zip",
		TamanoBytes: 52_428_801,
	})

	assert.ErrorIs(t, err, ErrFileTooLarge)
	repo.AssertExpectations(t)
}

// TestPresignUpload_MIMENotAllowed verifies ErrMIMENotAllowed for non-whitelisted type.
func TestPresignUpload_MIMENotAllowed(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	repo.On("ResolveVideoCourse", mock.Anything, videoID).Return(courseID, ownerID, "borrador", nil)

	_, err := svc.PresignUpload(context.Background(), videoID, ownerID, PresignInput{
		Nombre:      "malware.exe",
		ContentType: "application/x-msdownload",
		TamanoBytes: 1024,
	})

	assert.ErrorIs(t, err, ErrMIMENotAllowed)
	repo.AssertExpectations(t)
}

// TestPresignUpload_NonOwner verifies ErrNotOwner for non-owner caller.
// REQ-SEC chain ownership cross-creator → ErrNotOwner.
func TestPresignUpload_NonOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	repo.On("ResolveVideoCourse", mock.Anything, videoID).Return(courseID, ownerID, "borrador", nil)

	_, err := svc.PresignUpload(context.Background(), videoID, foreignID, PresignInput{
		Nombre:      "doc.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 1024,
	})

	assert.ErrorIs(t, err, ErrNotOwner,
		"cross-creator material presign must return ErrNotOwner (REQ-SEC chain ownership)")
	repo.AssertExpectations(t)
}

// TestPresignUpload_EstadoGate verifies ErrInvalidTransition for aprobado course.
func TestPresignUpload_EstadoGate(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	repo.On("ResolveVideoCourse", mock.Anything, videoID).Return(courseID, ownerID, "aprobado", nil)

	_, err := svc.PresignUpload(context.Background(), videoID, ownerID, PresignInput{
		Nombre:      "doc.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 1024,
	})

	assert.ErrorIs(t, err, ErrInvalidTransition,
		"aprobado course must block material presign (assertCourseEditable gate)")
	repo.AssertExpectations(t)
}

// ── ConfirmUpload tests (course-structure-v2: videoID-based) ──────────────────

// TestConfirmUpload_HappyPath verifies that a valid confirm persists the material.
func TestConfirmUpload_HappyPath(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	validKey := "courses/" + courseID + "/videos/" + videoID + "/materials/uuid-documento.pdf"

	repo.On("ResolveVideoCourse", mock.Anything, videoID).Return(courseID, ownerID, "borrador", nil)
	repo.On("CreateMaterial", mock.Anything, mock.AnythingOfType("*domain.Material")).Return(nil)

	model, err := svc.ConfirmUpload(context.Background(), videoID, ownerID, ConfirmInput{
		Key:         validKey,
		Nombre:      "documento.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 1024,
	})

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Equal(t, "documento.pdf", model.Titulo)
	assert.Equal(t, videoID, model.VideoID, "VideoID must be set on the persisted material")
	repo.AssertExpectations(t)
}

// TestConfirmUpload_BadKeyPrefix verifies ErrInvalidMaterialKey for wrong key prefix.
func TestConfirmUpload_BadKeyPrefix(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	repo.On("ResolveVideoCourse", mock.Anything, videoID).Return(courseID, ownerID, "borrador", nil)

	_, err := svc.ConfirmUpload(context.Background(), videoID, ownerID, ConfirmInput{
		Key:         "courses/other-course/videos/other-video/materials/uuid-file.pdf", // wrong IDs
		Nombre:      "file.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 1024,
	})

	assert.ErrorIs(t, err, ErrInvalidMaterialKey)
	repo.AssertExpectations(t)
}

// TestConfirmUpload_ReValidationSize verifies tampered size at confirm is rejected.
// LOAD-BEARING: dual validation.
func TestConfirmUpload_ReValidationSize(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	validKey := "courses/" + courseID + "/videos/" + videoID + "/materials/uuid-big.pdf"
	repo.On("ResolveVideoCourse", mock.Anything, videoID).Return(courseID, ownerID, "borrador", nil)

	_, err := svc.ConfirmUpload(context.Background(), videoID, ownerID, ConfirmInput{
		Key:         validKey,
		Nombre:      "big.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 52_428_801, // tampered: exceeds max
	})

	assert.ErrorIs(t, err, ErrFileTooLarge, "[LOAD-BEARING] dual validation: tampered size rejected")
	repo.AssertExpectations(t)
}

// TestConfirmUpload_ReValidationMIME verifies tampered MIME at confirm is rejected.
// LOAD-BEARING: dual validation.
func TestConfirmUpload_ReValidationMIME(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	validKey := "courses/" + courseID + "/videos/" + videoID + "/materials/uuid-file.exe"
	repo.On("ResolveVideoCourse", mock.Anything, videoID).Return(courseID, ownerID, "borrador", nil)

	_, err := svc.ConfirmUpload(context.Background(), videoID, ownerID, ConfirmInput{
		Key:         validKey,
		Nombre:      "file.exe",
		ContentType: "application/x-msdownload", // tampered
		TamanoBytes: 1024,
	})

	assert.ErrorIs(t, err, ErrMIMENotAllowed, "[LOAD-BEARING] dual validation: tampered MIME rejected")
	repo.AssertExpectations(t)
}

// ── ListMaterialsByVideo tests (course-structure-v2) ─────────────────────────

// TestListMaterialsByVideo_NonOwner verifies ErrNotOwner for non-owner.
func TestListMaterialsByVideo_NonOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	repo.On("ResolveVideoCourse", mock.Anything, videoID).Return(courseID, ownerID, "borrador", nil)

	_, err := svc.ListMaterialsByVideo(context.Background(), videoID, foreignID)

	assert.ErrorIs(t, err, ErrNotOwner)
	repo.AssertExpectations(t)
}

// ── PresignDownload tests (course-structure-v2: chain ownership) ──────────────

// TestPresignDownload_HappyPath verifies PresignGetURL is called with the stored key.
// course-structure-v2: uses GetMaterialOwnership for chain check (no courseID arg).
func TestPresignDownload_HappyPath(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	videoID := uuid.New().String()
	mat := materialWith(videoID)

	// GetMaterialOwnership resolves chain.
	repo.On("GetMaterialOwnership", mock.Anything, mat.ID).Return("course-id", ownerID, "borrador", nil)
	repo.On("GetMaterialByID", mock.Anything, mat.ID).Return(mat, nil)

	result, err := svc.PresignDownload(context.Background(), mat.ID, ownerID)

	require.NoError(t, err)
	assert.Contains(t, result.URL, mat.StorageKey)
	assert.Equal(t, []string{mat.StorageKey}, store.getCalls)
	repo.AssertExpectations(t)
}

// TestPresignDownload_NonOwner verifies ErrNotOwner for a non-owner who is NOT enrolled.
// OQ3 update: the service now checks enrollment when callerID != ownerID.
// A non-enrolled non-owner still gets ErrNotOwner.
func TestPresignDownload_NonOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	courseID := "course-id"
	videoID := uuid.New().String()
	mat := materialWith(videoID)

	repo.On("GetMaterialOwnership", mock.Anything, mat.ID).Return(courseID, ownerID, "borrador", nil)
	// Non-owner and NOT enrolled — IsEnrolled returns false.
	repo.On("IsEnrolled", mock.Anything, foreignID, courseID).Return(false, nil)

	_, err := svc.PresignDownload(context.Background(), mat.ID, foreignID)

	assert.ErrorIs(t, err, ErrNotOwner,
		"non-owner non-enrolled caller must receive ErrNotOwner (OQ3: owner OR enrolled)")
	repo.AssertExpectations(t)
}

// TestPresignDownload_MaterialNotFound verifies ErrMaterialNotFound.
func TestPresignDownload_MaterialNotFound(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	materialID := uuid.New().String()
	repo.On("GetMaterialOwnership", mock.Anything, materialID).Return("", "", "", repository.ErrMaterialNotFound)

	_, err := svc.PresignDownload(context.Background(), materialID, "any-caller")

	assert.ErrorIs(t, err, ErrMaterialNotFound)
	repo.AssertExpectations(t)
}

// ── DeleteMaterial tests (course-structure-v2: chain ownership) ───────────────

// TestDeleteMaterial_HappyPath verifies repo.DeleteMaterial BEFORE store.Delete.
// LOAD-BEARING: D5 ordering.
func TestDeleteMaterial_HappyPath(t *testing.T) {
	repo := &MockCoursesRepository{}
	var callOrder []string
	store := &mockStorageClient{
		DeleteFn: func(_ context.Context, _ string) error {
			callOrder = append(callOrder, "store.Delete")
			return nil
		},
	}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	videoID := uuid.New().String()
	mat := materialWith(videoID)

	repo.On("GetMaterialOwnership", mock.Anything, mat.ID).Return("course-id", ownerID, "borrador", nil)
	repo.On("GetMaterialByID", mock.Anything, mat.ID).Return(mat, nil)
	repo.On("DeleteMaterial", mock.Anything, mat.ID).
		Run(func(_ mock.Arguments) {
			callOrder = append(callOrder, "repo.DeleteMaterial")
		}).
		Return(nil)

	err := svc.DeleteMaterial(context.Background(), mat.ID, ownerID)

	require.NoError(t, err)
	require.Len(t, callOrder, 2)
	assert.Equal(t, "repo.DeleteMaterial", callOrder[0],
		"[LOAD-BEARING] repo.DeleteMaterial must be called BEFORE store.Delete")
	assert.Equal(t, "store.Delete", callOrder[1])
	repo.AssertExpectations(t)
}

// TestDeleteMaterial_StoreDeleteError verifies best-effort delete: store.Delete error is swallowed.
// LOAD-BEARING: D5.
func TestDeleteMaterial_StoreDeleteError_StillReturnsNil(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{
		DeleteFn: func(_ context.Context, _ string) error {
			return assert.AnError
		},
	}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	videoID := uuid.New().String()
	mat := materialWith(videoID)

	repo.On("GetMaterialOwnership", mock.Anything, mat.ID).Return("course-id", ownerID, "borrador", nil)
	repo.On("GetMaterialByID", mock.Anything, mat.ID).Return(mat, nil)
	repo.On("DeleteMaterial", mock.Anything, mat.ID).Return(nil)

	err := svc.DeleteMaterial(context.Background(), mat.ID, ownerID)

	assert.NoError(t, err, "[LOAD-BEARING] store.Delete error must be swallowed (D5)")
	repo.AssertExpectations(t)
}

// TestDeleteMaterial_NonOwner verifies ErrNotOwner for non-owner via chain.
func TestDeleteMaterial_NonOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	videoID := uuid.New().String()
	mat := materialWith(videoID)

	repo.On("GetMaterialOwnership", mock.Anything, mat.ID).Return("course-id", ownerID, "borrador", nil)

	err := svc.DeleteMaterial(context.Background(), mat.ID, foreignID)

	assert.ErrorIs(t, err, ErrNotOwner)
	repo.AssertExpectations(t)
}

// TestDeleteMaterial_MaterialNotFound verifies ErrMaterialNotFound.
func TestDeleteMaterial_MaterialNotFound(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	materialID := uuid.New().String()
	repo.On("GetMaterialOwnership", mock.Anything, materialID).Return("", "", "", repository.ErrMaterialNotFound)

	err := svc.DeleteMaterial(context.Background(), materialID, "any-caller")

	assert.ErrorIs(t, err, ErrMaterialNotFound)
	repo.AssertExpectations(t)
}

// ── FIX-1: PresignDownload owner-OR-enrolled tests (course-structure-v2 OQ3 resolution) ─────

// TestPresignDownload_EnrolledNonOwner_Returns200 verifies that an enrolled non-owner
// can download a material (OQ3 resolution: material is course content for learners).
// STRICT TDD: RED → GREEN.
func TestPresignDownload_EnrolledNonOwner_Returns200(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	enrolledUserID := uuid.New().String() // non-owner but enrolled
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	mat := materialWith(videoID)

	// Chain resolves to ownerID, but caller is enrolledUserID.
	repo.On("GetMaterialOwnership", mock.Anything, mat.ID).Return(courseID, ownerID, "aprobado", nil)
	// IsEnrolled returns true — enrolled non-owner should be allowed.
	repo.On("IsEnrolled", mock.Anything, enrolledUserID, courseID).Return(true, nil)
	repo.On("GetMaterialByID", mock.Anything, mat.ID).Return(mat, nil)

	result, err := svc.PresignDownload(context.Background(), mat.ID, enrolledUserID)

	require.NoError(t, err, "enrolled non-owner must be allowed to download (OQ3 resolution)")
	assert.NotEmpty(t, result.URL)
	repo.AssertExpectations(t)
}

// TestPresignDownload_OwnerCanDownload verifies owner can always download regardless of enrollment.
// OQ3 resolution: owner OR enrolled.
func TestPresignDownload_OwnerCanDownload(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	mat := materialWith(videoID)

	repo.On("GetMaterialOwnership", mock.Anything, mat.ID).Return(courseID, ownerID, "borrador", nil)
	// Owner: no need to check IsEnrolled, but ensure no unexpected call happens.
	repo.On("GetMaterialByID", mock.Anything, mat.ID).Return(mat, nil)

	result, err := svc.PresignDownload(context.Background(), mat.ID, ownerID)

	require.NoError(t, err, "owner must always be allowed to download")
	assert.NotEmpty(t, result.URL)
	repo.AssertExpectations(t)
}

// TestPresignDownload_NonEnrolledNonOwner_Returns403 verifies that a non-enrolled non-owner
// receives ErrNotOwner (→ 403). OQ3 resolution: neither owner nor enrolled.
func TestPresignDownload_NonEnrolledNonOwner_Returns403(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	strangerID := uuid.New().String() // not owner, not enrolled
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	mat := materialWith(videoID)

	repo.On("GetMaterialOwnership", mock.Anything, mat.ID).Return(courseID, ownerID, "aprobado", nil)
	repo.On("IsEnrolled", mock.Anything, strangerID, courseID).Return(false, nil)

	_, err := svc.PresignDownload(context.Background(), mat.ID, strangerID)

	assert.ErrorIs(t, err, ErrNotOwner,
		"non-enrolled non-owner must receive ErrNotOwner (OQ3 resolution)")
	repo.AssertExpectations(t)
}

// TestPresignDownload_NonExistentMaterial_Returns404 verifies ErrMaterialNotFound for missing material.
func TestPresignDownload_NonExistentMaterial_Returns404(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	materialID := uuid.New().String()
	repo.On("GetMaterialOwnership", mock.Anything, materialID).Return("", "", "", repository.ErrMaterialNotFound)

	_, err := svc.PresignDownload(context.Background(), materialID, "any-caller")

	assert.ErrorIs(t, err, ErrMaterialNotFound,
		"non-existent material must return ErrMaterialNotFound")
	repo.AssertExpectations(t)
}

// ── FIX-3 service: ConfirmUpload and ConfirmThumbnail estado-gate tests ─────────

// TestConfirmUpload_NonEditableCourse_Blocked verifies ConfirmUpload blocks on aprobado course.
// FIX-3 SUGGESTION-1: service gate test was missing.
func TestConfirmUpload_NonEditableCourse_Blocked(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	videoID := uuid.New().String()
	validKey := "courses/" + courseID + "/videos/" + videoID + "/materials/uuid-file.pdf"

	// Course is aprobado — ConfirmUpload must be blocked.
	repo.On("ResolveVideoCourse", mock.Anything, videoID).Return(courseID, ownerID, "aprobado", nil)

	_, err := svc.ConfirmUpload(context.Background(), videoID, ownerID, ConfirmInput{
		Key:         validKey,
		Nombre:      "file.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 1024,
	})

	assert.ErrorIs(t, err, ErrInvalidTransition,
		"aprobado course must block ConfirmUpload (assertCourseEditable gate)")
	repo.AssertExpectations(t)
}

// TestConfirmThumbnail_NonEditableCourse_Blocked verifies ConfirmThumbnail blocks on en_revision course.
// FIX-3 SUGGESTION-1: service gate test was missing.
func TestConfirmThumbnail_NonEditableCourse_Blocked(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoEnRevision)
	validKey := "courses/" + course.ID + "/thumbnail/uuid-cover.jpg"
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	err := svc.ConfirmThumbnail(context.Background(), course.ID, ownerID, validKey)

	assert.ErrorIs(t, err, ErrInvalidTransition,
		"en_revision course must block ConfirmThumbnail (assertCourseEditable gate)")
	repo.AssertExpectations(t)
}

// ── sanitizeFilename tests ─────────────────────────────────────────────────────

// TestSanitizeFilename verifies the filename sanitization function.
func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"hello world.pdf", "hello_world.pdf"},
		{"../../../etc/passwd", "passwd"},
		{"   ", "file"}, // all spaces → underscores → trimmed → empty → "file"
		// "! @#" becomes "_" (space→_, then !@# dropped, leaving _) → underscore before .pdf
		{"file with spaces and special chars! @#.pdf", "file_with_spaces_and_special_chars_.pdf"},
		{"normal.pdf", "normal.pdf"},
		{"", "file"}, // empty input → filepath.Base returns "." → fallback to "file"
	}
	for _, tc := range cases {
		got := sanitizeFilename(tc.input)
		assert.Equal(t, tc.want, got, "sanitizeFilename(%q) = %q, want %q", tc.input, got, tc.want)
	}
}

// ── GetCourseOwnership tests (C3.1 cross-module seam) ────────────────────────

// TestGetCourseOwnership_HappyPath verifies the seam returns creadorID and estado as string.
// Spec: CMO-1-A structural satisfaction.
func TestGetCourseOwnership_HappyPath(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	c := courseWith(ownerID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, c.ID).Return(c, nil)

	creadorID, estado, err := svc.GetCourseOwnership(context.Background(), c.ID)

	assert.NoError(t, err)
	assert.Equal(t, ownerID, creadorID, "GetCourseOwnership must return the correct creadorID")
	assert.Equal(t, "borrador", estado, "GetCourseOwnership must return estado as plain string")
	repo.AssertExpectations(t)
}

// TestGetCourseOwnership_NotFound_ReturnsErrCourseNotFound verifies sentinel wrapping.
// Spec: CMO-1-A, ADR-1.
func TestGetCourseOwnership_NotFound_ReturnsErrCourseNotFound(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	courseID := uuid.New().String()
	repo.On("GetByID", mock.Anything, courseID).Return(nil, repository.ErrCourseNotFound)

	_, _, err := svc.GetCourseOwnership(context.Background(), courseID)

	assert.ErrorIs(t, err, ErrCourseNotFound,
		"GetCourseOwnership must wrap repository.ErrCourseNotFound → service.ErrCourseNotFound")
	repo.AssertExpectations(t)
}

// ── T-1.6: SetEstado + ListByEstado service tests ─────────────────────────────

// TestSetEstado_Aprobado_CallsUpdateEstadoPublicado verifies that
// SetEstado("aprobado") routes to UpdateEstadoPublicado (not UpdateEstado).
// Spec: REQ-XMOD XMOD-3; Design §2 D2 service.
func TestSetEstado_Aprobado_CallsUpdateEstadoPublicado(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	courseID := uuid.New().String()
	// SetEstado("aprobado") must call UpdateEstadoPublicado.
	repo.On("UpdateEstadoPublicado", mock.Anything, courseID, domain.EstadoAprobado, mock.AnythingOfType("time.Time")).
		Return(nil)

	err := svc.SetEstado(context.Background(), courseID, "aprobado")
	assert.NoError(t, err, "SetEstado(aprobado) must succeed")
	repo.AssertExpectations(t)
	// UpdateEstado must NOT be called for aprobado.
	repo.AssertNotCalled(t, "UpdateEstado")
}

// TestSetEstado_Rechazado_CallsUpdateEstado verifies that
// SetEstado("rechazado") routes to UpdateEstado (NOT UpdateEstadoPublicado).
// Spec: REQ-XMOD XMOD-3; Design §2 D2 service.
func TestSetEstado_Rechazado_CallsUpdateEstado(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	courseID := uuid.New().String()
	// SetEstado("rechazado") must call UpdateEstado (not UpdateEstadoPublicado).
	repo.On("UpdateEstado", mock.Anything, courseID, domain.EstadoRechazado).Return(nil)

	err := svc.SetEstado(context.Background(), courseID, "rechazado")
	assert.NoError(t, err, "SetEstado(rechazado) must succeed")
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "UpdateEstadoPublicado")
}

// TestSetEstado_EnRevision_CallsUpdateEstado verifies SetEstado("en_revision") uses UpdateEstado.
func TestSetEstado_EnRevision_CallsUpdateEstado(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	courseID := uuid.New().String()
	repo.On("UpdateEstado", mock.Anything, courseID, domain.EstadoEnRevision).Return(nil)

	err := svc.SetEstado(context.Background(), courseID, "en_revision")
	assert.NoError(t, err, "SetEstado(en_revision) must succeed")
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "UpdateEstadoPublicado")
}

// TestSetEstado_InvalidEstado_ReturnsErrInvalidTransition verifies defense-in-depth.
func TestSetEstado_InvalidEstado_ReturnsErrInvalidTransition(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	err := svc.SetEstado(context.Background(), uuid.New().String(), "invalid_state")
	assert.ErrorIs(t, err, ErrInvalidTransition,
		"SetEstado with invalid estado must return ErrInvalidTransition")
	repo.AssertNotCalled(t, "UpdateEstado")
	repo.AssertNotCalled(t, "UpdateEstadoPublicado")
}

// TestListByEstado_DelegatesToRepoAndMapsToCourseSummary verifies that
// ListByEstado delegates to repo and maps []domain.Course → []CourseSummary.
// Spec: REQ-XMOD XMOD-3; Design §2 D2 service.
func TestListByEstado_DelegatesToRepoAndMapsToCourseSummary(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	courses := []domain.Course{
		{
			ID:        uuid.New().String(),
			CreadorID: ownerID,
			Titulo:    "Pending Course",
			Estado:    domain.EstadoEnRevision,
			CreatedAt: time.Now(),
		},
	}
	repo.On("ListByEstado", mock.Anything, domain.EstadoEnRevision).Return(courses, nil)

	result, err := svc.ListByEstado(context.Background(), "en_revision")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, courses[0].ID, result[0].ID)
	assert.Equal(t, "Pending Course", result[0].Titulo)
	assert.Equal(t, ownerID, result[0].CreadorID)
	repo.AssertExpectations(t)
}

// TestListByEstado_EmptyList verifies empty list is returned without error.
func TestListByEstado_EmptyList(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	repo.On("ListByEstado", mock.Anything, domain.EstadoEnRevision).Return([]domain.Course{}, nil)

	result, err := svc.ListByEstado(context.Background(), "en_revision")
	require.NoError(t, err)
	assert.Empty(t, result, "must return empty slice when repo returns no courses")
	repo.AssertExpectations(t)
}

// ── C2.4 catalog + enrollment service unit tests ──────────────────────────────

// TestListCatalog_DelegatesAndMapsPage verifies ListCatalog passes CatalogFilter verbatim to repo.
// Phase 2 (catalog-filters): updated to use CatalogFilter struct; ListCategoriasForCourses still called.
// Refs: REQ-COMPAT; ADR-1.
func TestListCatalog_DelegatesAndMapsPage(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	p := pagination.Params{Page: 1, Size: 20}
	filter := repository.CatalogFilter{Q: "go"}
	repoPage := pagination.Page[repository.CatalogCourseModel]{
		Items: []repository.CatalogCourseModel{
			{ID: "c1", Titulo: "Go", Descripcion: "Desc", CreadorNombre: "Alice", CreatedAt: time.Now()},
		},
		Page: 1, Size: 20, Total: 1, TotalPages: 1,
	}
	repo.On("ListApproved", mock.Anything, p, filter).Return(repoPage, nil)
	repo.On("ListCategoriasForCourses", mock.Anything, []string{"c1"}).
		Return(map[string][]domain.Categoria{}, nil)

	page, err := svc.ListCatalog(context.Background(), p, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	assert.Len(t, page.Items, 1)
	assert.Equal(t, "c1", page.Items[0].ID)
	assert.Equal(t, "Alice", page.Items[0].CreadorNombre)
	repo.AssertExpectations(t)
}

// TestListCatalog_PassesFullFilter_Verbatim verifies ListCatalog passes a full CatalogFilter
// verbatim to repo.ListApproved and that ListCategoriasForCourses is still called on IDs.
// Refs: REQ-COMBINED; ADR-1 (filter passthrough, no re-validation in service).
func TestListCatalog_PassesFullFilter_Verbatim(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	p := pagination.Params{Page: 1, Size: 10}
	filter := repository.CatalogFilter{
		Q:            "docker",
		Nivel:        "avanzado",
		CategoriaIDs: []string{"cat-uuid-1", "cat-uuid-2"},
		Sort:         "titulo",
	}
	repoPage := pagination.Page[repository.CatalogCourseModel]{
		Items: []repository.CatalogCourseModel{
			{ID: "c2", Titulo: "Docker Avanzado", Descripcion: "D", CreadorNombre: "Bob", CreatedAt: time.Now()},
		},
		Page: 1, Size: 10, Total: 1, TotalPages: 1,
	}
	// Assert the filter is passed verbatim — NOT decomposed or re-built.
	repo.On("ListApproved", mock.Anything, p, filter).Return(repoPage, nil)
	repo.On("ListCategoriasForCourses", mock.Anything, []string{"c2"}).
		Return(map[string][]domain.Categoria{}, nil)

	page, err := svc.ListCatalog(context.Background(), p, filter)
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	assert.Equal(t, "c2", page.Items[0].ID)
	repo.AssertExpectations(t)
}

// TestGetCatalogDetail_NotAprobado_Returns404 verifies non-approved → ErrCourseNotFound.
func TestGetCatalogDetail_NotAprobado_Returns404(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	repo.On("GetApprovedDetail", mock.Anything, "missing-id").
		Return(nil, repository.ErrCourseNotFound)

	_, err := svc.GetCatalogDetail(context.Background(), "userA", "missing-id")
	assert.ErrorIs(t, err, ErrCourseNotFound,
		"non-aprobado or missing course must return service.ErrCourseNotFound")
	repo.AssertExpectations(t)
}

// TestGetCatalogDetail_NotEnrolled_ReturnsPreview verifies preview when not enrolled.
// course-structure-v2: metadata always populated; Sections nil on non-enrolled.
func TestGetCatalogDetail_NotEnrolled_ReturnsPreview(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	courseID := uuid.New().String()
	userID := uuid.New().String()
	detail := &repository.CatalogCourseModel{
		ID:            courseID,
		Titulo:        "Go Course",
		Descripcion:   "Desc",
		CreadorNombre: "Alice",
		CreatedAt:     time.Now(),
	}

	repo.On("GetApprovedDetail", mock.Anything, courseID).Return(detail, nil)
	repo.On("IsEnrolled", mock.Anything, userID, courseID).Return(false, nil)
	repo.On("GetCourseCategorias", mock.Anything, courseID).Return([]domain.Categoria{}, nil)

	result, err := svc.GetCatalogDetail(context.Background(), userID, courseID)
	require.NoError(t, err)
	assert.False(t, result.Enrolled, "non-enrolled user must get Enrolled=false")
	assert.Nil(t, result.Sections, "non-enrolled user must get nil Sections (preview)")
	assert.Equal(t, courseID, result.ID)
	repo.AssertExpectations(t)
}

// TestGetCatalogDetail_Enrolled_ReturnsTree verifies full tree when enrolled.
// course-structure-v2: per-video materials; no course-level materiales.
func TestGetCatalogDetail_Enrolled_ReturnsTree(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	courseID := uuid.New().String()
	userID := uuid.New().String()
	sectionID := uuid.New().String()
	videoID := uuid.New().String()
	detail := &repository.CatalogCourseModel{
		ID:            courseID,
		Titulo:        "Go Course",
		Descripcion:   "Desc",
		CreadorNombre: "Alice",
		CreatedAt:     time.Now(),
	}
	sections := []domain.Section{
		{ID: sectionID, CourseID: courseID, Titulo: "Cap 1", Orden: 0},
	}
	videos := []domain.Video{
		{ID: videoID, SectionID: sectionID, Titulo: "Video 1", URL: "https://youtube.com/v", Proveedor: "youtube"},
	}
	// course-structure-v2: materials keyed by video_id, not course_id.
	materials := []domain.Material{
		{ID: uuid.New().String(), VideoID: videoID, Titulo: "doc.pdf", StorageKey: "k", MimeType: "application/pdf", TamanoBytes: 100},
	}

	repo.On("GetApprovedDetail", mock.Anything, courseID).Return(detail, nil)
	repo.On("IsEnrolled", mock.Anything, userID, courseID).Return(true, nil)
	repo.On("GetCourseCategorias", mock.Anything, courseID).Return([]domain.Categoria{}, nil)
	repo.On("ListSectionsByCourse", mock.Anything, courseID).Return(sections, nil)
	repo.On("ListMaterialsByCourseVideos", mock.Anything, courseID).Return(materials, nil)
	// course-player-progress: ListVideoProgressByUserAndCourse is now called by buildContentTree.
	repo.On("ListVideoProgressByUserAndCourse", mock.Anything, userID, courseID).
		Return([]repository.VideoProgressRow{}, nil)
	repo.On("ListVideosBySection", mock.Anything, sectionID).Return(videos, nil)

	result, err := svc.GetCatalogDetail(context.Background(), userID, courseID)
	require.NoError(t, err)
	assert.True(t, result.Enrolled, "enrolled user must get Enrolled=true")
	require.Len(t, result.Sections, 1, "enrolled user must see 1 section")
	assert.Equal(t, sectionID, result.Sections[0].Section.ID)
	require.Len(t, result.Sections[0].Videos, 1)
	// course-structure-v2: materials are now per-video, not course-level.
	require.Len(t, result.Sections[0].Videos[0].Materiales, 1,
		"per-video material must be populated (course-structure-v2)")
	repo.AssertExpectations(t)
}

// TestEnroll_NonAprobado_Returns404 verifies enroll on non-approved → ErrCourseNotFound.
func TestEnroll_NonAprobado_Returns404(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	courseID := uuid.New().String()
	userID := uuid.New().String()

	// GetByID returns a borrador course.
	repo.On("GetByID", mock.Anything, courseID).Return(courseWith(userID, domain.EstadoBorrador), nil)

	err := svc.Enroll(context.Background(), userID, courseID)
	assert.ErrorIs(t, err, ErrCourseNotFound,
		"enroll on borrador must return ErrCourseNotFound (draft-invisibility)")
	repo.AssertExpectations(t)
}

// TestEnroll_Aprobado_CallsCreateEnrollment verifies enroll on aprobado calls repo.
func TestEnroll_Aprobado_CallsCreateEnrollment(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	courseID := uuid.New().String()
	userID := uuid.New().String()
	aprobado := courseWith("creator-id", domain.EstadoAprobado)
	aprobado.ID = courseID

	repo.On("GetByID", mock.Anything, courseID).Return(aprobado, nil)
	repo.On("CreateEnrollment", mock.Anything, userID, courseID).Return(nil)

	err := svc.Enroll(context.Background(), userID, courseID)
	require.NoError(t, err, "enroll on aprobado must succeed")
	repo.AssertExpectations(t)
}

// TestMarkEnrollmentCompleted_DelegatesToRepo verifies seam delegates to MarkCompleted.
func TestMarkEnrollmentCompleted_DelegatesToRepo(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	userID := uuid.New().String()
	courseID := uuid.New().String()

	repo.On("MarkCompleted", mock.Anything, userID, courseID).Return(nil)

	err := svc.MarkEnrollmentCompleted(context.Background(), userID, courseID)
	require.NoError(t, err, "MarkEnrollmentCompleted must delegate to repo.MarkCompleted")
	repo.AssertExpectations(t)
}

// TestMarkEnrollmentCompleted_NoOp verifies nil returned when no enrollment row.
func TestMarkEnrollmentCompleted_NoOp(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	userID := uuid.New().String()
	courseID := uuid.New().String()

	// MarkCompleted with 0 rows affected → nil (no-op, no error).
	repo.On("MarkCompleted", mock.Anything, userID, courseID).Return(nil)

	err := svc.MarkEnrollmentCompleted(context.Background(), userID, courseID)
	assert.NoError(t, err, "MarkEnrollmentCompleted no-op path must return nil")
	repo.AssertExpectations(t)
}

// ── course-structure-v2 coverage additions ────────────────────────────────────

// TestListCategorias_DelegatesAndMaps verifies ListCategorias delegates to repo.
func TestListCategorias_DelegatesAndMaps(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	cats := []domain.Categoria{
		{ID: "c1", Nombre: "Backend", Slug: "backend"},
		{ID: "c2", Nombre: "Frontend", Slug: "frontend"},
	}
	repo.On("ListCategorias", mock.Anything).Return(cats, nil)

	result, err := svc.ListCategorias(context.Background())
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "c1", result[0].ID)
	assert.Equal(t, "Backend", result[0].Nombre)
	repo.AssertExpectations(t)
}

// TestPresignThumbnail_HappyPath verifies PresignThumbnail returns key with thumbnail prefix.
func TestPresignThumbnail_HappyPath(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	result, err := svc.PresignThumbnail(context.Background(), course.ID, ownerID, PresignInput{
		Nombre:      "cover.jpg",
		ContentType: "image/jpeg",
		TamanoBytes: 512_000,
	})

	require.NoError(t, err)
	assert.Contains(t, result.Key, "courses/"+course.ID+"/thumbnail/")
	assert.NotEmpty(t, result.UploadURL)
	repo.AssertExpectations(t)
}

// TestPresignThumbnail_NonOwner verifies ErrNotOwner for non-owner.
func TestPresignThumbnail_NonOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.PresignThumbnail(context.Background(), course.ID, foreignID, PresignInput{
		Nombre:      "cover.jpg",
		ContentType: "image/jpeg",
		TamanoBytes: 512_000,
	})

	assert.ErrorIs(t, err, ErrNotOwner)
	repo.AssertExpectations(t)
}

// TestPresignThumbnail_NonImageMIME verifies ErrMIMENotAllowed for non-image MIME.
func TestPresignThumbnail_NonImageMIME(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.PresignThumbnail(context.Background(), course.ID, ownerID, PresignInput{
		Nombre:      "notes.pdf",
		ContentType: "application/pdf", // not an image MIME
		TamanoBytes: 1024,
	})

	assert.ErrorIs(t, err, ErrMIMENotAllowed, "non-image MIME must be rejected for thumbnail")
	repo.AssertExpectations(t)
}

// TestConfirmThumbnail_HappyPath verifies ConfirmThumbnail sets miniatura_key.
func TestConfirmThumbnail_HappyPath(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	validKey := "courses/" + course.ID + "/thumbnail/uuid-cover.jpg"
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
	repo.On("UpdateByID", mock.Anything, course.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	err := svc.ConfirmThumbnail(context.Background(), course.ID, ownerID, validKey)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// TestConfirmThumbnail_BadKeyPrefix verifies ErrInvalidMaterialKey for wrong prefix.
func TestConfirmThumbnail_BadKeyPrefix(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	err := svc.ConfirmThumbnail(context.Background(), course.ID, ownerID,
		"courses/other-course/thumbnail/uuid-img.jpg")

	assert.ErrorIs(t, err, ErrInvalidMaterialKey)
	repo.AssertExpectations(t)
}

// TestCreate_WithCategoriaIDs verifies Create validates and sets categoriaIDs.
func TestCreate_WithCategoriaIDs(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	catID := uuid.New().String()
	repo.On("CategoriasExist", mock.Anything, []string{catID}).Return(true, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Course")).Return(nil)
	repo.On("SetCourseCategorias", mock.Anything, mock.AnythingOfType("string"), []string{catID}).Return(nil)

	_, err := svc.Create(context.Background(), ownerID, CreateRequest{
		Titulo:       "My Course",
		Descripcion:  "Desc",
		CategoriaIDs: []string{catID},
	})
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// TestCreate_WithBogusCategoria verifies ErrInvalidCategoria when category not found.
func TestCreate_WithBogusCategoria(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	ownerID := uuid.New().String()
	bogusID := uuid.New().String()
	repo.On("CategoriasExist", mock.Anything, []string{bogusID}).Return(false, nil)

	_, err := svc.Create(context.Background(), ownerID, CreateRequest{
		Titulo:       "My Course",
		CategoriaIDs: []string{bogusID},
	})
	assert.ErrorIs(t, err, ErrInvalidCategoria, "bogus categoria ID must return ErrInvalidCategoria")
	repo.AssertExpectations(t)
}

// TestRoundHorasVideo_Math verifies the rounding formula matches the spec.
// Spec REQ-COMPUTED: 1 decimal; ROUND(SUM/3600, 1).
func TestRoundHorasVideo_Math(t *testing.T) {
	cases := []struct{ input, want float64 }{
		{3600.0 / 3600, 1.0},     // 3600s → 1.0
		{5400.0 / 3600, 1.5},     // 5400s → 1.5
		{0.0, 0.0},               // 0s → 0.0
		{3700.0 / 3600, 1.0},     // 3700/3600 = 1.0277... → rounds to 1.0
		{2 * 3600.0 / 3600, 2.0}, // 7200s → 2.0
	}
	for _, tc := range cases {
		got := roundHorasVideo(tc.input)
		assert.Equal(t, tc.want, got, "roundHorasVideo(%v) = %v, want %v", tc.input, got, tc.want)
	}
}

// TestListMyCourses_DelegatesAndMaps verifies ListMyCourses maps repo model.
func TestListMyCourses_DelegatesAndMaps(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	userID := uuid.New().String()
	rows := []repository.EnrollmentWithCourseModel{
		{CourseID: "c1", Titulo: "Go", CreadorNombre: "Alice", Completado: false, InscritoEn: time.Now()},
	}
	repo.On("ListEnrollmentsByUser", mock.Anything, userID).Return(rows, nil)

	result, err := svc.ListMyCourses(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "c1", result[0].CourseID)
	assert.Equal(t, "Alice", result[0].CreadorNombre)
	repo.AssertExpectations(t)
}

// ── MarkVideoProgress service unit tests (course-player-progress / REQ-PROGRESS-WRITE) ──

// TestMarkVideoProgress_Enrolled_CallsUpsert verifies that an enrolled caller triggers UpsertVideoProgress.
// Satisfies REQ-PROGRESS-WRITE / Design §3.
func TestMarkVideoProgress_Enrolled_CallsUpsert(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	userID := uuid.New().String()
	videoID := uuid.New().String()
	courseID := uuid.New().String()
	creadorID := uuid.New().String()

	repo.On("ResolveVideoCourse", mock.Anything, videoID).
		Return(courseID, creadorID, "aprobado", nil)
	repo.On("IsEnrolled", mock.Anything, userID, courseID).Return(true, nil)
	repo.On("UpsertVideoProgress", mock.Anything, userID, videoID, true, 0).Return(nil)

	err := svc.MarkVideoProgress(context.Background(), userID, videoID, true, 0)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

// TestMarkVideoProgress_NotEnrolled_ReturnsErrNotEnrolled verifies that a non-enrolled caller
// receives ErrNotEnrolled and UpsertVideoProgress is never called. REQ-SEC / REQ-DECOUPLE.
func TestMarkVideoProgress_NotEnrolled_ReturnsErrNotEnrolled(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	userID := uuid.New().String()
	videoID := uuid.New().String()
	courseID := uuid.New().String()
	creadorID := uuid.New().String()

	repo.On("ResolveVideoCourse", mock.Anything, videoID).
		Return(courseID, creadorID, "aprobado", nil)
	repo.On("IsEnrolled", mock.Anything, userID, courseID).Return(false, nil)
	// UpsertVideoProgress must NOT be called.

	err := svc.MarkVideoProgress(context.Background(), userID, videoID, true, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotEnrolled, "non-enrolled caller must receive ErrNotEnrolled")
	repo.AssertNotCalled(t, "UpsertVideoProgress")
	repo.AssertExpectations(t)
}

// TestMarkVideoProgress_VideoNotFound_ReturnsErrVideoNotFound verifies that a nonexistent video
// returns ErrVideoNotFound and no upsert is attempted. REQ-SEC (404 no-leak).
func TestMarkVideoProgress_VideoNotFound_ReturnsErrVideoNotFound(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	userID := uuid.New().String()
	videoID := uuid.New().String()

	repo.On("ResolveVideoCourse", mock.Anything, videoID).
		Return("", "", "", repository.ErrVideoNotFound)
	// IsEnrolled and UpsertVideoProgress must NOT be called.

	err := svc.MarkVideoProgress(context.Background(), userID, videoID, true, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVideoNotFound, "nonexistent video must return ErrVideoNotFound")
	repo.AssertNotCalled(t, "IsEnrolled")
	repo.AssertNotCalled(t, "UpsertVideoProgress")
	repo.AssertExpectations(t)
}

// TestMarkVideoProgress_NeverCallsMarkEnrollmentCompleted verifies that MarkVideoProgress
// NEVER touches MarkEnrollmentCompleted or MarkCompleted (decouple invariant, REQ-DECOUPLE).
func TestMarkVideoProgress_NeverCallsMarkEnrollmentCompleted(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	userID := uuid.New().String()
	videoID := uuid.New().String()
	courseID := uuid.New().String()
	creadorID := uuid.New().String()

	repo.On("ResolveVideoCourse", mock.Anything, videoID).
		Return(courseID, creadorID, "aprobado", nil)
	repo.On("IsEnrolled", mock.Anything, userID, courseID).Return(true, nil)
	repo.On("UpsertVideoProgress", mock.Anything, userID, videoID, true, 0).Return(nil)

	err := svc.MarkVideoProgress(context.Background(), userID, videoID, true, 0)
	require.NoError(t, err)

	// Assert decoupling: MarkCompleted must NEVER be called (REQ-DECOUPLE / D4).
	repo.AssertNotCalled(t, "MarkCompleted")
	repo.AssertExpectations(t)
}

// TestMarkVideoProgress_InvalidUUID_ReturnsErrVideoNotFound verifies that a malformed
// (non-UUID) videoID returns ErrVideoNotFound immediately, without reaching the repository.
// This prevents Postgres SQLSTATE 22P02 from escaping as a 500 (REQ-PROGRESS-WRITE / D3).
// Fix: uuid.Parse(videoID) check at the top of MarkVideoProgress — invalid → ErrVideoNotFound.
func TestMarkVideoProgress_InvalidUUID_ReturnsErrVideoNotFound(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	userID := uuid.New().String()
	// "not-a-uuid" is NOT a valid UUID — must be caught BEFORE any repo call.

	err := svc.MarkVideoProgress(context.Background(), userID, "not-a-uuid", true, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVideoNotFound,
		"non-UUID videoID must return ErrVideoNotFound (prevents Postgres 22P02 → 500)")

	// The repo must never be reached — no SQL call should occur.
	repo.AssertNotCalled(t, "ResolveVideoCourse")
	repo.AssertNotCalled(t, "IsEnrolled")
	repo.AssertNotCalled(t, "UpsertVideoProgress")
	repo.AssertExpectations(t)
}

// TestMarkVideoProgress_InvalidUUID_Triangulation verifies another malformed form
// (empty string) also returns ErrVideoNotFound, triangulating the UUID guard.
func TestMarkVideoProgress_InvalidUUID_Triangulation(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvc(repo)

	userID := uuid.New().String()

	err := svc.MarkVideoProgress(context.Background(), userID, "", true, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrVideoNotFound,
		"empty string videoID must also return ErrVideoNotFound (UUID validation catches it)")

	repo.AssertNotCalled(t, "ResolveVideoCourse")
	repo.AssertExpectations(t)
}

// ── buildContentTree completado attachment tests (REQ-PROGRESS-READ) ──────────

// TestBuildContentTree_AttachesCompletadoFromOneBatchCall verifies:
//   - ListVideoProgressByUserAndCourse is called EXACTLY ONCE (no N+1, REQ-PROGRESS-READ).
//   - vm.Completado is set correctly from the progress map.
//
// Satisfies REQ-PROGRESS-READ / Design §3.
func TestBuildContentTree_AttachesCompletadoFromOneBatchCall(t *testing.T) {
	repo := &MockCoursesRepository{}
	svc := newSvcWithStore(repo, &mockStorageClient{})

	ctx := context.Background()
	courseID := uuid.New().String()
	userID := uuid.New().String()
	sectionID := uuid.New().String()
	videoID1 := uuid.New().String()
	videoID2 := uuid.New().String()

	sections := []domain.Section{
		{ID: sectionID, CourseID: courseID, Titulo: "S1", Orden: 0},
	}
	videos1 := []domain.Video{
		{ID: videoID1, SectionID: sectionID, Titulo: "V1", URL: "http://x", Proveedor: "youtube"},
		{ID: videoID2, SectionID: sectionID, Titulo: "V2", URL: "http://y", Proveedor: "youtube"},
	}
	progressRows := []repository.VideoProgressRow{
		{VideoID: videoID1, Completado: true},
		// videoID2 absent — should default to false.
	}

	repo.On("ListSectionsByCourse", mock.Anything, courseID).Return(sections, nil)
	repo.On("ListMaterialsByCourseVideos", mock.Anything, courseID).Return([]domain.Material{}, nil)
	// CRITICAL: called EXACTLY ONCE (no N+1).
	repo.On("ListVideoProgressByUserAndCourse", mock.Anything, userID, courseID).
		Return(progressRows, nil).Once()
	repo.On("ListVideosBySection", mock.Anything, sectionID).Return(videos1, nil)

	result, err := svc.buildContentTree(ctx, userID, courseID)
	require.NoError(t, err)
	require.Len(t, result, 1, "must return 1 section")
	require.Len(t, result[0].Videos, 2, "must return 2 videos")

	// videoID1 must be completado=true; videoID2 must be completado=false (default).
	assert.True(t, result[0].Videos[0].Completado, "V1 must have Completado=true from progress row")
	assert.False(t, result[0].Videos[1].Completado, "V2 must have Completado=false (no progress row)")

	// Assert ListVideoProgressByUserAndCourse called exactly once (no N+1).
	repo.AssertNumberOfCalls(t, "ListVideoProgressByUserAndCourse", 1)
	repo.AssertExpectations(t)
}
