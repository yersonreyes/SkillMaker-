// Package approvals is the public API of the approvals module.
// All cross-module imports MUST go through this package — never into
// approvals/repository, approvals/service, or approvals/handler directly.
//
// Mirrors the courses/evaluations facade pattern exactly.
package approvals

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/service"
)

// Re-export the public types so callers only need to import "approvals".
type (
	// Service is the public interface of the approvals domain.
	Service = service.Service
	// Repository is the data-access contract for the approvals module.
	Repository = repository.Repository

	// CourseStateManager is the narrow cross-module seam into courses (C4.1).
	// Re-exported so main.go can reference it if needed; coursesSvc satisfies it structurally.
	CourseStateManager = service.CourseStateManager

	// EvaluationValidator is the narrow cross-module seam into evaluations (C4.1).
	// Re-exported so main.go can reference it if needed; evaluationsSvc satisfies it structurally.
	EvaluationValidator = service.EvaluationValidator
)

// NewRepository constructs a GORM-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return repository.New(db)
}

// NewService constructs a Service backed by the given Repository, CourseStateManager,
// and EvaluationValidator. coursesSvc and evaluationsSvc satisfy the seams structurally.
func NewService(r Repository, courses CourseStateManager, evals EvaluationValidator) Service {
	return service.New(r, courses, evals)
}

// RegisterCreatorRoutes mounts the creator-gated approvals routes:
//   - POST /courses/:courseId/submit
//
// creatorGrp must carry JWT + RequireRole("creador") middleware.
func RegisterCreatorRoutes(creatorGrp *gin.RouterGroup, svc Service) {
	handler.RegisterCreator(creatorGrp, svc)
}

// RegisterAdminRoutes mounts the admin-gated approvals routes:
//   - GET  /approvals/pending
//   - POST /courses/:courseId/approve
//   - POST /courses/:courseId/reject
//
// adminGrp must carry JWT + RequireRole("administrador") middleware.
func RegisterAdminRoutes(adminGrp *gin.RouterGroup, svc Service) {
	handler.RegisterAdmin(adminGrp, svc)
}

// RegisterHistoryRoutes mounts the JWT-only history route (in-handler owner-or-admin authz):
//   - GET /courses/:id/approvals
//
// protected must carry JWT middleware only (no RequireRole restriction).
func RegisterHistoryRoutes(protected *gin.RouterGroup, svc Service) {
	handler.RegisterHistory(protected, svc)
}
