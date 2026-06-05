// Package handler — HTTP-layer contract tests for the evaluations module.
//
// Strategy: spin up a real gin.Engine per test (gin.New(), no global state),
// mount the routes with middleware chain (injectIdentity → RequireRole("creador")),
// wire a mockEvalSvc, fire requests via net/http/httptest.
//
// CRITICAL [LOAD-BEARING (e)]: the safety-net test registers BOTH courses.Register
// AND evaluations.Register on a single gin.Engine and asserts no panic. This catches
// any Gin wildcard parameter-name conflict (:id vs :courseId) between modules.
// The Gin tree panics at route REGISTRATION time, not at request time.
//
// No build tag: runs with standard `make backend-test`.
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	coursesHandler "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/handler"
	coursesService "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── Mock evaluation service ────────────────────────────────────────────────────

type mockEvalSvc struct {
	mock.Mock
}

func (m *mockEvalSvc) CreateEvaluation(ctx context.Context, courseID, creadorID string, req service.EvaluationCreateRequest) (*service.EvaluationModel, error) {
	args := m.Called(ctx, courseID, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.EvaluationModel), args.Error(1)
}

func (m *mockEvalSvc) GetEvaluation(ctx context.Context, courseID, creadorID string) (*service.EvaluationDetailModel, error) {
	args := m.Called(ctx, courseID, creadorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.EvaluationDetailModel), args.Error(1)
}

func (m *mockEvalSvc) UpdateEvaluation(ctx context.Context, evalID, creadorID string, req service.EvaluationUpdateRequest) (*service.EvaluationModel, error) {
	args := m.Called(ctx, evalID, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.EvaluationModel), args.Error(1)
}

func (m *mockEvalSvc) CreateQuestion(ctx context.Context, evalID, creadorID string, req service.QuestionCreateRequest) (*service.QuestionWithOptionsModel, error) {
	args := m.Called(ctx, evalID, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.QuestionWithOptionsModel), args.Error(1)
}

func (m *mockEvalSvc) UpdateQuestion(ctx context.Context, questionID, creadorID string, req service.QuestionUpdateRequest) (*service.QuestionModel, error) {
	args := m.Called(ctx, questionID, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.QuestionModel), args.Error(1)
}

func (m *mockEvalSvc) DeleteQuestion(ctx context.Context, questionID, creadorID string) error {
	args := m.Called(ctx, questionID, creadorID)
	return args.Error(0)
}

func (m *mockEvalSvc) CreateOption(ctx context.Context, questionID, creadorID string, req service.OptionCreateRequest) (*service.OptionModel, error) {
	args := m.Called(ctx, questionID, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.OptionModel), args.Error(1)
}

func (m *mockEvalSvc) UpdateOption(ctx context.Context, optionID, creadorID string, req service.OptionUpdateRequest) (*service.OptionModel, error) {
	args := m.Called(ctx, optionID, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.OptionModel), args.Error(1)
}

func (m *mockEvalSvc) DeleteOption(ctx context.Context, optionID, creadorID string) error {
	args := m.Called(ctx, optionID, creadorID)
	return args.Error(0)
}

func (m *mockEvalSvc) ValidateEvaluationComplete(ctx context.Context, evalID string) error {
	args := m.Called(ctx, evalID)
	return args.Error(0)
}

func (m *mockEvalSvc) StartAttempt(ctx context.Context, evaluationID, userID string) (*service.AttemptModel, error) {
	args := m.Called(ctx, evaluationID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AttemptModel), args.Error(1)
}

func (m *mockEvalSvc) GetAttempt(ctx context.Context, attemptID, userID string) (*service.AttemptStateModel, error) {
	args := m.Called(ctx, attemptID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AttemptStateModel), args.Error(1)
}

func (m *mockEvalSvc) SaveAnswer(ctx context.Context, attemptID, userID, questionID, optionID string) error {
	args := m.Called(ctx, attemptID, userID, questionID, optionID)
	return args.Error(0)
}

func (m *mockEvalSvc) SubmitAttempt(ctx context.Context, attemptID, userID string) (*service.AttemptResultModel, error) {
	args := m.Called(ctx, attemptID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.AttemptResultModel), args.Error(1)
}

// ValidateSubmitReady satisfies the C4.1 addition to evaluations.Service.
func (m *mockEvalSvc) ValidateSubmitReady(ctx context.Context, courseID string) error {
	args := m.Called(ctx, courseID)
	return args.Error(0)
}

// GetEvaluationSummaryForStudent satisfies the student-eval-discovery addition.
func (m *mockEvalSvc) GetEvaluationSummaryForStudent(ctx context.Context, courseID string) (*service.EvaluationSummaryModel, error) {
	args := m.Called(ctx, courseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.EvaluationSummaryModel), args.Error(1)
}

// ── Mock courses service (minimal — satisfies full interface) ──────────────────

type mockCourseSvc struct {
	mock.Mock
}

func (m *mockCourseSvc) Create(ctx context.Context, creadorID string, req coursesService.CreateRequest) (*coursesService.CourseModel, error) {
	args := m.Called(ctx, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*coursesService.CourseModel), args.Error(1)
}
func (m *mockCourseSvc) GetByID(ctx context.Context, id, creadorID string) (*coursesService.CourseModel, error) {
	args := m.Called(ctx, id, creadorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*coursesService.CourseModel), args.Error(1)
}
func (m *mockCourseSvc) UpdateByID(ctx context.Context, id, creadorID string, req coursesService.UpdateRequest) (*coursesService.CourseModel, error) {
	args := m.Called(ctx, id, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*coursesService.CourseModel), args.Error(1)
}
func (m *mockCourseSvc) ListByCreator(ctx context.Context, creadorID string, p pagination.Params) (pagination.Page[coursesService.CourseModel], error) {
	args := m.Called(ctx, creadorID, p)
	return args.Get(0).(pagination.Page[coursesService.CourseModel]), args.Error(1)
}
func (m *mockCourseSvc) CreateSection(ctx context.Context, creadorID string, req coursesService.SectionCreateRequest) (*coursesService.SectionModel, error) {
	args := m.Called(ctx, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*coursesService.SectionModel), args.Error(1)
}
func (m *mockCourseSvc) GetSectionByID(ctx context.Context, id string) (*coursesService.SectionModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*coursesService.SectionModel), args.Error(1)
}
func (m *mockCourseSvc) UpdateSection(ctx context.Context, id, creadorID string, req coursesService.SectionUpdateRequest) (*coursesService.SectionModel, error) {
	args := m.Called(ctx, id, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*coursesService.SectionModel), args.Error(1)
}
func (m *mockCourseSvc) DeleteSection(ctx context.Context, id, creadorID string) error {
	args := m.Called(ctx, id, creadorID)
	return args.Error(0)
}
func (m *mockCourseSvc) ListSections(ctx context.Context, courseID string) ([]coursesService.SectionModel, error) {
	args := m.Called(ctx, courseID)
	return args.Get(0).([]coursesService.SectionModel), args.Error(1)
}
func (m *mockCourseSvc) ListContent(ctx context.Context, courseID, creadorID string) ([]coursesService.SectionWithVideosModel, error) {
	args := m.Called(ctx, courseID, creadorID)
	return args.Get(0).([]coursesService.SectionWithVideosModel), args.Error(1)
}
func (m *mockCourseSvc) ReorderSections(ctx context.Context, courseID, creadorID string, ids []string) error {
	args := m.Called(ctx, courseID, creadorID, ids)
	return args.Error(0)
}
func (m *mockCourseSvc) CreateVideo(ctx context.Context, creadorID string, req coursesService.VideoCreateRequest) (*coursesService.VideoModel, error) {
	args := m.Called(ctx, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*coursesService.VideoModel), args.Error(1)
}
func (m *mockCourseSvc) UpdateVideo(ctx context.Context, id, creadorID string, req coursesService.VideoUpdateRequest) (*coursesService.VideoModel, error) {
	args := m.Called(ctx, id, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*coursesService.VideoModel), args.Error(1)
}
func (m *mockCourseSvc) DeleteVideo(ctx context.Context, id, creadorID string) error {
	args := m.Called(ctx, id, creadorID)
	return args.Error(0)
}
func (m *mockCourseSvc) ListVideos(ctx context.Context, sectionID string) ([]coursesService.VideoModel, error) {
	args := m.Called(ctx, sectionID)
	return args.Get(0).([]coursesService.VideoModel), args.Error(1)
}
func (m *mockCourseSvc) HasContent(ctx context.Context, courseID, creadorID string) (bool, error) {
	args := m.Called(ctx, courseID, creadorID)
	return args.Bool(0), args.Error(1)
}
func (m *mockCourseSvc) PresignUpload(ctx context.Context, courseID, creadorID string, req coursesService.PresignInput) (coursesService.PresignResult, error) {
	args := m.Called(ctx, courseID, creadorID, req)
	return args.Get(0).(coursesService.PresignResult), args.Error(1)
}
func (m *mockCourseSvc) ConfirmUpload(ctx context.Context, courseID, creadorID string, req coursesService.ConfirmInput) (*coursesService.MaterialModel, error) {
	args := m.Called(ctx, courseID, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*coursesService.MaterialModel), args.Error(1)
}
func (m *mockCourseSvc) ListMaterials(ctx context.Context, courseID, creadorID string) ([]coursesService.MaterialModel, error) {
	args := m.Called(ctx, courseID, creadorID)
	return args.Get(0).([]coursesService.MaterialModel), args.Error(1)
}
func (m *mockCourseSvc) PresignDownload(ctx context.Context, courseID, materialID, creadorID string) (coursesService.DownloadResult, error) {
	args := m.Called(ctx, courseID, materialID, creadorID)
	return args.Get(0).(coursesService.DownloadResult), args.Error(1)
}
func (m *mockCourseSvc) DeleteMaterial(ctx context.Context, materialID, creadorID string) error {
	args := m.Called(ctx, materialID, creadorID)
	return args.Error(0)
}
func (m *mockCourseSvc) GetCourseOwnership(ctx context.Context, courseID string) (creadorID, estado string, err error) {
	args := m.Called(ctx, courseID)
	return args.String(0), args.String(1), args.Error(2)
}

// SetEstado satisfies the C4.1 addition to courses.Service.
func (m *mockCourseSvc) SetEstado(ctx context.Context, courseID, estado string) error {
	args := m.Called(ctx, courseID, estado)
	return args.Error(0)
}

// ListByEstado satisfies the C4.1 addition to courses.Service.
func (m *mockCourseSvc) ListByEstado(ctx context.Context, estado string) ([]coursesService.CourseSummary, error) {
	args := m.Called(ctx, estado)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]coursesService.CourseSummary), args.Error(1)
}

// ── C2.4 catalog + enrollment additions ──────────────────────────────────────

func (m *mockCourseSvc) ListCatalog(ctx context.Context, p pagination.Params, q string) (pagination.Page[coursesService.CatalogCourseModel], error) {
	args := m.Called(ctx, p, q)
	return args.Get(0).(pagination.Page[coursesService.CatalogCourseModel]), args.Error(1)
}

func (m *mockCourseSvc) GetCatalogDetail(ctx context.Context, userID, courseID string) (*coursesService.CatalogDetailModel, error) {
	args := m.Called(ctx, userID, courseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*coursesService.CatalogDetailModel), args.Error(1)
}

func (m *mockCourseSvc) Enroll(ctx context.Context, userID, courseID string) error {
	args := m.Called(ctx, userID, courseID)
	return args.Error(0)
}

func (m *mockCourseSvc) ListMyCourses(ctx context.Context, userID string) ([]coursesService.MyCourseModel, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]coursesService.MyCourseModel), args.Error(1)
}

func (m *mockCourseSvc) MarkEnrollmentCompleted(ctx context.Context, userID, courseID string) error {
	args := m.Called(ctx, userID, courseID)
	return args.Error(0)
}

// GetCourseTitulo satisfies the C5.1 addition — certificates seam (CourseTituloReader).
func (m *mockCourseSvc) GetCourseTitulo(_ context.Context, _ string) (string, error) {
	return "", nil
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func injectIdentity(userID string, roles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("roles", roles)
		c.Next()
	}
}

func setupEvalEngine(svc service.Service, userID string) *gin.Engine {
	r := gin.New()
	identity := injectIdentity(userID, []string{"creador"})
	creatorGrp := r.Group("", identity, middleware.RequireRole("creador"))
	handler.Register(creatorGrp, svc)
	return r
}

func do(engine *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf.Write(b)
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// ── [LOAD-BEARING (e)] Gin route safety-net test ─────────────────────────────

// TestGinRouteBoot_CoursesAndEvaluations_NoPanic registers BOTH courses and evaluations
// routes on a single gin.Engine and asserts no panic.
// This catches :id vs :courseId wildcard conflicts at registration time.
// The gin route tree panics at registration time on param-name conflicts —
// this is the safety-net that guards the CRITICAL Gin param convention.
// Spec: ADR-7 R5.
func TestGinRouteBoot_CoursesAndEvaluations_NoPanic(t *testing.T) {
	evalSvc := &mockEvalSvc{}
	courseSvc := &mockCourseSvc{}

	assert.NotPanics(t, func() {
		r := gin.New()
		identity := injectIdentity("user-1", []string{"creador"})
		creatorGrp := r.Group("", identity, middleware.RequireRole("creador"))

		// Register BOTH modules on the SAME creatorGrp.
		coursesHandler.Register(creatorGrp, courseSvc)
		handler.Register(creatorGrp, evalSvc)
	}, "[LOAD-BEARING (e)] courses + evaluations registered on single gin.Engine must NOT panic")
}

// TestGinRouteBoot_AllEvaluationsRoutesResolve verifies all 9 evaluation routes are reachable.
// Spec: ADR-7.
func TestGinRouteBoot_AllEvaluationsRoutesResolve(t *testing.T) {
	evalSvc := &mockEvalSvc{}
	courseSvc := &mockCourseSvc{}

	var routes gin.RoutesInfo
	assert.NotPanics(t, func() {
		r := gin.New()
		identity := injectIdentity("user-1", []string{"creador"})
		creatorGrp := r.Group("", identity, middleware.RequireRole("creador"))
		coursesHandler.Register(creatorGrp, courseSvc)
		handler.Register(creatorGrp, evalSvc)
		routes = r.Routes()
	})

	// Count evaluations-specific routes.
	evalRoutes := map[string]bool{
		"POST:/courses/:courseId/evaluation": false,
		"GET:/courses/:id/evaluation":        false,
		"PATCH:/evaluations/:id":             false,
		"POST:/evaluations/:id/questions":    false,
		"PATCH:/questions/:id":               false,
		"DELETE:/questions/:id":              false,
		"POST:/questions/:id/options":        false,
		"PATCH:/options/:id":                 false,
		"DELETE:/options/:id":                false,
	}

	for _, route := range routes {
		key := route.Method + ":" + route.Path
		if _, ok := evalRoutes[key]; ok {
			evalRoutes[key] = true
		}
	}

	for route, found := range evalRoutes {
		assert.True(t, found, "evaluation route %q must be registered", route)
	}
}

// ── Error mapping tests ────────────────────────────────────────────────────────

// TestCreateEvaluation_NonOwner_Returns403 verifies ErrNotOwner → 403 on write route.
// Spec: EVL-1-C, ERR-1-A.
func TestCreateEvaluation_NonOwner_Returns403(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := "requester-1"
	engine := setupEvalEngine(svc, userID)

	courseID := "course-1"
	svc.On("CreateEvaluation", mock.Anything, courseID, userID, mock.Anything).
		Return(nil, service.ErrNotOwner)

	w := do(engine, http.MethodPost, "/courses/"+courseID+"/evaluation",
		map[string]any{"notaMinima": 70, "intentosMax": 3})
	assert.Equal(t, http.StatusForbidden, w.Code, "non-owner create evaluation must return 403")
}

// TestCreateEvaluation_NotEditable_Returns409 verifies ErrCourseNotEditable → 409.
// Spec: EVL-1-E.
func TestCreateEvaluation_NotEditable_Returns409(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := "owner-1"
	engine := setupEvalEngine(svc, userID)

	courseID := "course-1"
	svc.On("CreateEvaluation", mock.Anything, courseID, userID, mock.Anything).
		Return(nil, service.ErrCourseNotEditable)

	w := do(engine, http.MethodPost, "/courses/"+courseID+"/evaluation",
		map[string]any{"notaMinima": 70})
	assert.Equal(t, http.StatusConflict, w.Code, "non-editable course must return 409")
}

// TestCreateEvaluation_EvalExists_Returns409 verifies ErrEvaluationExists → 409.
// Spec: EVL-1-B.
func TestCreateEvaluation_EvalExists_Returns409(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := "owner-1"
	engine := setupEvalEngine(svc, userID)

	courseID := "course-1"
	svc.On("CreateEvaluation", mock.Anything, courseID, userID, mock.Anything).
		Return(nil, service.ErrEvaluationExists)

	w := do(engine, http.MethodPost, "/courses/"+courseID+"/evaluation",
		map[string]any{"notaMinima": 70})
	assert.Equal(t, http.StatusConflict, w.Code, "duplicate evaluation must return 409")
}

// TestCreateEvaluation_HappyPath_Returns201 verifies happy-path POST returns 201.
func TestCreateEvaluation_HappyPath_Returns201(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := "owner-1"
	engine := setupEvalEngine(svc, userID)

	courseID := "course-1"
	model := &service.EvaluationModel{
		ID:          "eval-1",
		CourseID:    courseID,
		NotaMinima:  70,
		IntentosMax: 3,
	}
	svc.On("CreateEvaluation", mock.Anything, courseID, userID, mock.Anything).
		Return(model, nil)

	w := do(engine, http.MethodPost, "/courses/"+courseID+"/evaluation",
		map[string]any{"notaMinima": 70, "intentosMax": 3})
	assert.Equal(t, http.StatusCreated, w.Code)
}

// TestGetEvaluation_NonOwner_Returns404 verifies ErrNotOwner → 404 on GET route.
// Spec: EVL-2-C, ERR-1-A.
func TestGetEvaluation_NonOwner_Returns404(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := "requester-1"
	engine := setupEvalEngine(svc, userID)

	courseID := "course-1"
	svc.On("GetEvaluation", mock.Anything, courseID, userID).
		Return(nil, service.ErrNotOwner)

	w := do(engine, http.MethodGet, "/courses/"+courseID+"/evaluation", nil)
	assert.Equal(t, http.StatusNotFound, w.Code, "non-owner GET evaluation must return 404 (read convention)")
}

// TestGetEvaluation_HappyPath_Returns200 verifies nested tree response on GET.
func TestGetEvaluation_HappyPath_Returns200(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := "owner-1"
	engine := setupEvalEngine(svc, userID)

	courseID := "course-1"
	detail := &service.EvaluationDetailModel{
		EvaluationModel: service.EvaluationModel{
			ID: "eval-1", CourseID: courseID, NotaMinima: 70, IntentosMax: 0,
		},
		Questions: []service.QuestionWithOptionsModel{},
	}
	svc.On("GetEvaluation", mock.Anything, courseID, userID).Return(detail, nil)

	w := do(engine, http.MethodGet, "/courses/"+courseID+"/evaluation", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "eval-1", resp["id"])
}

// TestCreateQuestion_InvalidTipo_Returns400 verifies ErrInvalidQuestionType → 400.
// Spec: QST-1-C.
func TestCreateQuestion_InvalidTipo_Returns400(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := "owner-1"
	engine := setupEvalEngine(svc, userID)

	evalID := "eval-1"
	svc.On("CreateQuestion", mock.Anything, evalID, userID, mock.Anything).
		Return(nil, service.ErrInvalidQuestionType)

	w := do(engine, http.MethodPost, "/evaluations/"+evalID+"/questions",
		map[string]any{"enunciado": "Q?", "tipo": "abierta", "puntaje": 5})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestDeleteQuestion_NonOwner_Returns403 verifies non-owner DELETE returns 403.
func TestDeleteQuestion_NonOwner_Returns403(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := "requester-1"
	engine := setupEvalEngine(svc, userID)

	questionID := "q-1"
	svc.On("DeleteQuestion", mock.Anything, questionID, userID).Return(service.ErrNotOwner)

	w := do(engine, http.MethodDelete, "/questions/"+questionID, nil)
	assert.Equal(t, http.StatusForbidden, w.Code, "non-owner DELETE question must return 403")
}

// TestDeleteQuestion_Owner_Returns204 verifies successful DELETE returns 204.
func TestDeleteQuestion_Owner_Returns204(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := "owner-1"
	engine := setupEvalEngine(svc, userID)

	questionID := "q-1"
	svc.On("DeleteQuestion", mock.Anything, questionID, userID).Return(nil)

	w := do(engine, http.MethodDelete, "/questions/"+questionID, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ── Fixture helpers ────────────────────────────────────────────────────────────

func evalModel(id, courseID string) *service.EvaluationModel {
	return &service.EvaluationModel{
		ID:          id,
		CourseID:    courseID,
		NotaMinima:  70,
		IntentosMax: 0,
	}
}

func questionModel(id, evalID string) *service.QuestionModel {
	return &service.QuestionModel{
		ID:           id,
		EvaluationID: evalID,
		Enunciado:    "Test question",
		Tipo:         "opcion_multiple",
		Puntaje:      5,
		Orden:        0,
	}
}

func optionModel(id, questionID string) *service.OptionModel {
	return &service.OptionModel{
		ID:         id,
		QuestionID: questionID,
		Texto:      "Option A",
		Correcta:   true,
		Orden:      0,
	}
}

// suppress unused warning for fixture helpers
var (
	_ = evalModel
	_ = questionModel
	_ = optionModel
	_ = time.Now
)
