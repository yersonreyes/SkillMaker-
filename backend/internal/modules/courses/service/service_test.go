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

func (m *MockCoursesRepository) ListMaterialsByCourse(ctx context.Context, courseID string) ([]domain.Material, error) {
	args := m.Called(ctx, courseID)
	return args.Get(0).([]domain.Material), args.Error(1)
}

func (m *MockCoursesRepository) DeleteMaterial(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ── MockStorageClient ─────────────────────────────────────────────────────────

// mockStorageClient is a minimal mock for storage.Client using func fields.
// This avoids testify/mock overhead for the storage interface (4 methods).
type mockStorageClient struct {
	PresignPutURLFn func(ctx context.Context, key string, ttl time.Duration) (string, error)
	PresignGetURLFn func(ctx context.Context, key string, ttl time.Duration) (string, error)
	DeleteFn        func(ctx context.Context, key string) error
	PingFn          func(ctx context.Context) error
	// Track calls for assertion.
	deleteCalls []string
	putCalls    []string
	getCalls    []string
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

// newSvcWithStore creates a serviceImpl with the given storage client.
func newSvcWithStore(repo *MockCoursesRepository, store *mockStorageClient) *serviceImpl {
	return New(repo, store, 15*time.Minute, 52_428_800).(*serviceImpl)
}

// materialWith creates a domain.Material fixture for testing.
func materialWith(courseID string) *domain.Material {
	return &domain.Material{
		ID:          uuid.New().String(),
		CourseID:    courseID,
		Titulo:      "test-document.pdf",
		StorageKey:  "courses/" + courseID + "/materials/uuid-test-document.pdf",
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

// ── PresignUpload tests (C2.3) ────────────────────────────────────────────────

// TestPresignUpload_HappyPath verifies successful presign returns key with correct prefix.
// Spec: REQ-PRESIGN happy path.
func TestPresignUpload_HappyPath(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	result, err := svc.PresignUpload(context.Background(), course.ID, ownerID, PresignInput{
		Nombre:      "documento.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 1024,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.UploadURL)
	assert.Contains(t, result.Key, "courses/"+course.ID+"/materials/",
		"key must start with courses/{id}/materials/")
	assert.False(t, result.ExpiresAt.IsZero())
	repo.AssertExpectations(t)
}

// TestPresignUpload_FileTooLarge verifies ErrFileTooLarge when size exceeds max.
// Spec: REQ-PRESIGN "File too large" scenario, AC4.
func TestPresignUpload_FileTooLarge(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.PresignUpload(context.Background(), course.ID, ownerID, PresignInput{
		Nombre:      "big.zip",
		ContentType: "application/zip",
		TamanoBytes: 52_428_801, // 1 byte over 50 MB
	})

	assert.ErrorIs(t, err, ErrFileTooLarge,
		"file size exceeding max must return ErrFileTooLarge")
	repo.AssertExpectations(t)
}

// TestPresignUpload_MIMENotAllowed verifies ErrMIMENotAllowed for non-whitelisted type.
// Spec: REQ-PRESIGN "MIME not in whitelist" scenario, AC5.
func TestPresignUpload_MIMENotAllowed(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.PresignUpload(context.Background(), course.ID, ownerID, PresignInput{
		Nombre:      "malware.exe",
		ContentType: "application/x-msdownload",
		TamanoBytes: 1024,
	})

	assert.ErrorIs(t, err, ErrMIMENotAllowed,
		"non-whitelisted MIME must return ErrMIMENotAllowed")
	repo.AssertExpectations(t)
}

// TestPresignUpload_NonOwner verifies ErrNotOwner for non-owner caller.
// Spec: REQ-PRESIGN "Non-owner creador" scenario, AC2.
func TestPresignUpload_NonOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.PresignUpload(context.Background(), course.ID, foreignID, PresignInput{
		Nombre:      "doc.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 1024,
	})

	assert.ErrorIs(t, err, ErrNotOwner,
		"non-owner must get ErrNotOwner from PresignUpload")
	repo.AssertExpectations(t)
}

// TestPresignUpload_EstadoGate verifies ErrInvalidTransition for non-editable course.
// Spec: REQ-PRESIGN "Course estado gate" scenario.
func TestPresignUpload_EstadoGate(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoEnRevision) // not editable
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.PresignUpload(context.Background(), course.ID, ownerID, PresignInput{
		Nombre:      "doc.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 1024,
	})

	assert.ErrorIs(t, err, ErrInvalidTransition,
		"non-editable course estado must return ErrInvalidTransition")
	repo.AssertExpectations(t)
}

// ── ConfirmUpload tests (C2.3) ────────────────────────────────────────────────

// TestConfirmUpload_HappyPath verifies that a valid confirm persists the material.
// Spec: REQ-CONFIRM happy path.
func TestConfirmUpload_HappyPath(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	validKey := "courses/" + course.ID + "/materials/uuid-documento.pdf"

	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
	repo.On("CreateMaterial", mock.Anything, mock.AnythingOfType("*domain.Material")).Return(nil)

	model, err := svc.ConfirmUpload(context.Background(), course.ID, ownerID, ConfirmInput{
		Key:         validKey,
		Nombre:      "documento.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 1024,
	})

	require.NoError(t, err)
	require.NotNil(t, model)
	assert.Equal(t, "documento.pdf", model.Titulo,
		"Titulo must be set from req.Nombre (D1: nombre is the wire label)")
	repo.AssertExpectations(t)
}

// TestConfirmUpload_BadKeyPrefix verifies ErrInvalidMaterialKey for wrong key prefix.
// Spec: REQ-CONFIRM key validation.
func TestConfirmUpload_BadKeyPrefix(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.ConfirmUpload(context.Background(), course.ID, ownerID, ConfirmInput{
		Key:         "courses/other-course/materials/uuid-file.pdf", // wrong courseID in prefix
		Nombre:      "file.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 1024,
	})

	assert.ErrorIs(t, err, ErrInvalidMaterialKey,
		"key with wrong prefix must return ErrInvalidMaterialKey")
	repo.AssertExpectations(t)
}

// TestConfirmUpload_ReValidationSize verifies that a tampered size at confirm is rejected.
// LOAD-BEARING: dual validation — even if presign would have passed, tampered confirm is rejected.
// Spec: REQ-CONFIRM "Re-validation — file too large at confirm" scenario.
func TestConfirmUpload_ReValidationSize(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	validKey := "courses/" + course.ID + "/materials/uuid-big.pdf"
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.ConfirmUpload(context.Background(), course.ID, ownerID, ConfirmInput{
		Key:         validKey,
		Nombre:      "big.pdf",
		ContentType: "application/pdf",
		TamanoBytes: 52_428_801, // tampered: exceeds max
	})

	assert.ErrorIs(t, err, ErrFileTooLarge,
		"[LOAD-BEARING] tampered size at confirm must return ErrFileTooLarge (dual validation)")
	repo.AssertExpectations(t)
}

// TestConfirmUpload_ReValidationMIME verifies that a tampered MIME at confirm is rejected.
// LOAD-BEARING: dual validation — even if presign would have passed, tampered confirm is rejected.
// Spec: REQ-CONFIRM "Re-validation — MIME not allowed at confirm" scenario.
func TestConfirmUpload_ReValidationMIME(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	validKey := "courses/" + course.ID + "/materials/uuid-file.exe"
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.ConfirmUpload(context.Background(), course.ID, ownerID, ConfirmInput{
		Key:         validKey,
		Nombre:      "file.exe",
		ContentType: "application/x-msdownload", // tampered: not in whitelist
		TamanoBytes: 1024,
	})

	assert.ErrorIs(t, err, ErrMIMENotAllowed,
		"[LOAD-BEARING] tampered MIME at confirm must return ErrMIMENotAllowed (dual validation)")
	repo.AssertExpectations(t)
}

// ── ListMaterials tests (C2.3) ────────────────────────────────────────────────

// TestListMaterials_NonOwner verifies ErrNotOwner for non-owner caller.
// Spec: REQ-LIST "Non-owner list" scenario.
func TestListMaterials_NonOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.ListMaterials(context.Background(), course.ID, foreignID)

	assert.ErrorIs(t, err, ErrNotOwner,
		"non-owner must get ErrNotOwner from ListMaterials (→ 404 via read helper)")
	repo.AssertExpectations(t)
}

// ── PresignDownload tests (C2.3) ──────────────────────────────────────────────

// TestPresignDownload_HappyPath verifies that PresignGetURL is called with the stored key.
// Spec: REQ-DOWNLOAD happy path.
func TestPresignDownload_HappyPath(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	mat := materialWith(course.ID)

	repo.On("GetMaterialByID", mock.Anything, mat.ID).Return(mat, nil)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	result, err := svc.PresignDownload(context.Background(), course.ID, mat.ID, ownerID)

	require.NoError(t, err)
	assert.Contains(t, result.URL, mat.StorageKey,
		"PresignGetURL must be called with the stored key")
	assert.Equal(t, []string{mat.StorageKey}, store.getCalls,
		"PresignGetURL must be called exactly once with the stored key")
	repo.AssertExpectations(t)
}

// TestPresignDownload_NonOwner verifies ErrNotOwner for non-owner caller.
// Spec: REQ-DOWNLOAD "Non-owner download" scenario.
func TestPresignDownload_NonOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	mat := materialWith(course.ID)

	repo.On("GetMaterialByID", mock.Anything, mat.ID).Return(mat, nil)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	_, err := svc.PresignDownload(context.Background(), course.ID, mat.ID, foreignID)

	assert.ErrorIs(t, err, ErrNotOwner,
		"non-owner must get ErrNotOwner from PresignDownload (→ 404 via read helper)")
	repo.AssertExpectations(t)
}

// TestPresignDownload_MaterialNotFound verifies ErrMaterialNotFound when ID does not exist.
// Spec: REQ-DOWNLOAD "Material not found" scenario.
func TestPresignDownload_MaterialNotFound(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	materialID := uuid.New().String()
	repo.On("GetMaterialByID", mock.Anything, materialID).Return(nil, repository.ErrMaterialNotFound)

	_, err := svc.PresignDownload(context.Background(), "any-course", materialID, ownerID)

	assert.ErrorIs(t, err, ErrMaterialNotFound,
		"missing material must return ErrMaterialNotFound")
	repo.AssertExpectations(t)
}

// ── DeleteMaterial tests (C2.3) ───────────────────────────────────────────────

// TestDeleteMaterial_HappyPath verifies that DeleteMaterial calls repo.DeleteMaterial
// BEFORE store.Delete, and returns nil on success.
// LOAD-BEARING: repo.DeleteMaterial must precede store.Delete per D5 / spec REQ-DELETE.
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
	course := courseWith(ownerID, domain.EstadoBorrador)
	mat := materialWith(course.ID)

	repo.On("GetMaterialByID", mock.Anything, mat.ID).Return(mat, nil)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
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
	assert.Equal(t, "store.Delete", callOrder[1],
		"store.Delete must be called after repo.DeleteMaterial")
	repo.AssertExpectations(t)
}

// TestDeleteMaterial_StoreDeleteError verifies best-effort delete: store.Delete error
// MUST NOT fail the request. The method must still return nil.
// LOAD-BEARING: D5 — never fail the DELETE endpoint on object-delete error.
// Spec: REQ-DELETE "Object already gone — still succeeds" scenario, AC6.
func TestDeleteMaterial_StoreDeleteError_StillReturnsNil(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{
		DeleteFn: func(_ context.Context, _ string) error {
			return assert.AnError // simulates a transient storage error
		},
	}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	mat := materialWith(course.ID)

	repo.On("GetMaterialByID", mock.Anything, mat.ID).Return(mat, nil)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)
	repo.On("DeleteMaterial", mock.Anything, mat.ID).Return(nil)

	err := svc.DeleteMaterial(context.Background(), mat.ID, ownerID)

	assert.NoError(t, err,
		"[LOAD-BEARING] store.Delete error must be swallowed — DeleteMaterial must return nil (D5)")
	repo.AssertExpectations(t)
}

// TestDeleteMaterial_NonOwner verifies ErrNotOwner for non-owner caller.
// Spec: REQ-DELETE "Non-owner delete" scenario.
func TestDeleteMaterial_NonOwner(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	foreignID := uuid.New().String()
	course := courseWith(ownerID, domain.EstadoBorrador)
	mat := materialWith(course.ID)

	repo.On("GetMaterialByID", mock.Anything, mat.ID).Return(mat, nil)
	repo.On("GetByID", mock.Anything, course.ID).Return(course, nil)

	err := svc.DeleteMaterial(context.Background(), mat.ID, foreignID)

	assert.ErrorIs(t, err, ErrNotOwner,
		"non-owner must get ErrNotOwner from DeleteMaterial (→ 403)")
	repo.AssertExpectations(t)
}

// TestDeleteMaterial_MaterialNotFound verifies ErrMaterialNotFound when ID does not exist.
// Spec: REQ-DELETE "Material not found" scenario.
func TestDeleteMaterial_MaterialNotFound(t *testing.T) {
	repo := &MockCoursesRepository{}
	store := &mockStorageClient{}
	svc := newSvcWithStore(repo, store)

	ownerID := uuid.New().String()
	materialID := uuid.New().String()
	repo.On("GetMaterialByID", mock.Anything, materialID).Return(nil, repository.ErrMaterialNotFound)

	err := svc.DeleteMaterial(context.Background(), materialID, ownerID)

	assert.ErrorIs(t, err, ErrMaterialNotFound,
		"missing material must return ErrMaterialNotFound")
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
