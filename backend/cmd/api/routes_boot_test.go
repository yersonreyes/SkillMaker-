// Package main — boot-time route registration tests.
// No build tag: runs with standard `make backend-test`.
//
// Validates that all route groups (courses, evaluations, approvals) can be registered
// on the same gin.Engine without panicking due to Gin wildcard-param conflicts.
//
// CRITICAL: Gin panics at ROUTE REGISTRATION TIME when two routes in the same method
// tree use different parameter names at the same position (e.g. :id vs :courseId).
// This test is the safety-net that catches those conflicts before they reach production.
//
// Design §0: no boot test existed prior to C4.1. This file creates it (new file).
package main

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses"
	coursesService "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations"
	evalService "github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── nil-safe mock services ─────────────────────────────────────────────────────

// nilCourseSvc is a nil-safe stub for courses.Service that satisfies the interface
// for route registration (never called during this test).
type nilCourseSvc struct{}

func (n *nilCourseSvc) Create(_ context.Context, _ string, _ coursesService.CreateRequest) (*coursesService.CourseModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) GetByID(_ context.Context, _, _ string) (*coursesService.CourseModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) UpdateByID(_ context.Context, _, _ string, _ coursesService.UpdateRequest) (*coursesService.CourseModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) ListByCreator(_ context.Context, _ string, _ pagination.Params) (pagination.Page[coursesService.CourseModel], error) {
	return pagination.Page[coursesService.CourseModel]{}, nil
}
func (n *nilCourseSvc) CreateSection(_ context.Context, _ string, _ coursesService.SectionCreateRequest) (*coursesService.SectionModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) GetSectionByID(_ context.Context, _ string) (*coursesService.SectionModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) UpdateSection(_ context.Context, _, _ string, _ coursesService.SectionUpdateRequest) (*coursesService.SectionModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) DeleteSection(_ context.Context, _, _ string) error { return nil }
func (n *nilCourseSvc) ListSections(_ context.Context, _ string) ([]coursesService.SectionModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) ListContent(_ context.Context, _, _ string) ([]coursesService.SectionWithVideosModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) ReorderSections(_ context.Context, _, _ string, _ []string) error { return nil }
func (n *nilCourseSvc) CreateVideo(_ context.Context, _ string, _ coursesService.VideoCreateRequest) (*coursesService.VideoModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) UpdateVideo(_ context.Context, _, _ string, _ coursesService.VideoUpdateRequest) (*coursesService.VideoModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) DeleteVideo(_ context.Context, _, _ string) error { return nil }
func (n *nilCourseSvc) ListVideos(_ context.Context, _ string) ([]coursesService.VideoModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) HasContent(_ context.Context, _, _ string) (bool, error) { return false, nil }
func (n *nilCourseSvc) PresignUpload(_ context.Context, _, _ string, _ coursesService.PresignInput) (coursesService.PresignResult, error) {
	return coursesService.PresignResult{}, nil
}
func (n *nilCourseSvc) ConfirmUpload(_ context.Context, _, _ string, _ coursesService.ConfirmInput) (*coursesService.MaterialModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) ListMaterials(_ context.Context, _, _ string) ([]coursesService.MaterialModel, error) {
	return nil, nil
}
func (n *nilCourseSvc) PresignDownload(_ context.Context, _, _, _ string) (coursesService.DownloadResult, error) {
	return coursesService.DownloadResult{}, nil
}
func (n *nilCourseSvc) DeleteMaterial(_ context.Context, _, _ string) error { return nil }
func (n *nilCourseSvc) GetCourseOwnership(_ context.Context, _ string) (creadorID, estado string, err error) {
	return "", "", nil
}
func (n *nilCourseSvc) SetEstado(_ context.Context, _, _ string) error { return nil }
func (n *nilCourseSvc) ListByEstado(_ context.Context, _ string) ([]coursesService.CourseSummary, error) {
	return nil, nil
}

// nilEvalSvc is a nil-safe stub for evaluations.Service.
type nilEvalSvc struct{}

func (n *nilEvalSvc) CreateEvaluation(_ context.Context, _, _ string, _ evalService.EvaluationCreateRequest) (*evalService.EvaluationModel, error) {
	return nil, nil
}
func (n *nilEvalSvc) GetEvaluation(_ context.Context, _, _ string) (*evalService.EvaluationDetailModel, error) {
	return nil, nil
}
func (n *nilEvalSvc) UpdateEvaluation(_ context.Context, _, _ string, _ evalService.EvaluationUpdateRequest) (*evalService.EvaluationModel, error) {
	return nil, nil
}
func (n *nilEvalSvc) CreateQuestion(_ context.Context, _, _ string, _ evalService.QuestionCreateRequest) (*evalService.QuestionWithOptionsModel, error) {
	return nil, nil
}
func (n *nilEvalSvc) UpdateQuestion(_ context.Context, _, _ string, _ evalService.QuestionUpdateRequest) (*evalService.QuestionModel, error) {
	return nil, nil
}
func (n *nilEvalSvc) DeleteQuestion(_ context.Context, _, _ string) error { return nil }
func (n *nilEvalSvc) CreateOption(_ context.Context, _, _ string, _ evalService.OptionCreateRequest) (*evalService.OptionModel, error) {
	return nil, nil
}
func (n *nilEvalSvc) UpdateOption(_ context.Context, _, _ string, _ evalService.OptionUpdateRequest) (*evalService.OptionModel, error) {
	return nil, nil
}
func (n *nilEvalSvc) DeleteOption(_ context.Context, _, _ string) error            { return nil }
func (n *nilEvalSvc) ValidateEvaluationComplete(_ context.Context, _ string) error { return nil }
func (n *nilEvalSvc) ValidateSubmitReady(_ context.Context, _ string) error        { return nil }
func (n *nilEvalSvc) StartAttempt(_ context.Context, _, _ string) (*evalService.AttemptModel, error) {
	return nil, nil
}
func (n *nilEvalSvc) GetAttempt(_ context.Context, _, _ string) (*evalService.AttemptStateModel, error) {
	return nil, nil
}
func (n *nilEvalSvc) SaveAnswer(_ context.Context, _, _, _, _ string) error { return nil }
func (n *nilEvalSvc) SubmitAttempt(_ context.Context, _, _ string) (*evalService.AttemptResultModel, error) {
	return nil, nil
}

// nilApprovalSvc is a nil-safe stub for approvals.Service.
type nilApprovalSvc struct{}

func (n *nilApprovalSvc) SubmitToReview(_ context.Context, _, _ string) error { return nil }
func (n *nilApprovalSvc) Approve(_ context.Context, _, _, _ string) error     { return nil }
func (n *nilApprovalSvc) Reject(_ context.Context, _, _, _ string) error      { return nil }
func (n *nilApprovalSvc) ListPending(_ context.Context) ([]courses.CourseSummary, error) {
	return nil, nil
}
func (n *nilApprovalSvc) ListHistory(_ context.Context, _, _ string, _ bool) ([]domain.Approval, error) {
	return nil, nil
}

// ── Boot test ─────────────────────────────────────────────────────────────────

// TestRouteBoot_AllModules_NoPanic registers ALL module routes (courses + evaluations + approvals)
// on a single gin.Engine and asserts no panic occurs.
//
// This test catches :courseId vs :id Gin param-tree conflicts at registration time.
// Spec: AC-15; Design §0 grounding delta (no existing boot test).
func TestRouteBoot_AllModules_NoPanic(t *testing.T) {
	courseSvc := &nilCourseSvc{}
	evalSvc := &nilEvalSvc{}
	approvalSvc := &nilApprovalSvc{}

	assert.NotPanics(t, func() {
		r := gin.New()
		api := r.Group("/api")

		protected := api.Group("", middleware.JWT("test-secret"))
		adminGrp := protected.Group("", middleware.RequireRole("administrador"))
		creatorGrp := protected.Group("", middleware.RequireRole("creador"))

		// Register all module routes (same order as main.go).
		courses.RegisterRoutes(creatorGrp, courseSvc)
		evaluations.RegisterRoutes(creatorGrp, evalSvc)
		evaluations.RegisterStudentRoutes(protected, evalSvc)

		// Approvals routes (C4.1).
		approvals.RegisterCreatorRoutes(creatorGrp, approvalSvc)
		approvals.RegisterAdminRoutes(adminGrp, approvalSvc)
		approvals.RegisterHistoryRoutes(protected, approvalSvc)

		// Verify all 5 approvals routes are registered.
		routes := r.Routes()
		routeMap := make(map[string]bool, len(routes))
		for _, ri := range routes {
			routeMap[ri.Method+" "+ri.Path] = true
		}

		assert.True(t, routeMap["POST /api/courses/:courseId/submit"],
			"submit route must be registered")
		assert.True(t, routeMap["GET /api/approvals/pending"],
			"list pending route must be registered")
		assert.True(t, routeMap["POST /api/courses/:courseId/approve"],
			"approve route must be registered")
		assert.True(t, routeMap["POST /api/courses/:courseId/reject"],
			"reject route must be registered")
		assert.True(t, routeMap["GET /api/courses/:id/approvals"],
			"list history route must be registered")
	}, "registering all module routes must not panic (no Gin param-tree conflict)")

	_ = time.Now() // suppress unused import
}
