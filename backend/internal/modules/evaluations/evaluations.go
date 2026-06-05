// Package evaluations is the public API of the evaluations module.
// All cross-module imports MUST go through this package — never into
// evaluations/repository, evaluations/service, or evaluations/handler directly.
//
// Mirrors the courses.go facade pattern exactly.
package evaluations

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/service"
)

// Re-export the public types so callers only need to import "evaluations".
type (
	// Service is the public interface of the evaluations domain.
	Service = service.Service
	// Repository is the data-access contract for the evaluations module.
	Repository = repository.Repository
	// CoursesChecker is the narrow cross-module seam into courses (ADR-1).
	// Re-exported so main.go can reference it if needed; coursesSvc satisfies it structurally.
	CoursesChecker = service.CoursesChecker

	// EvaluationModel is the flat service read model.
	EvaluationModel = service.EvaluationModel
	// EvaluationDetailModel is the nested service read model.
	EvaluationDetailModel = service.EvaluationDetailModel

	// EvaluationCreateRequest is the service-level create request.
	EvaluationCreateRequest = service.EvaluationCreateRequest
	// EvaluationUpdateRequest is the service-level update request.
	EvaluationUpdateRequest = service.EvaluationUpdateRequest
)

// NewRepository constructs a GORM-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return repository.New(db)
}

// NewService constructs a Service that delegates to the given Repository and CoursesChecker.
// courses.Service satisfies CoursesChecker structurally (ADR-1) — pass coursesSvc directly.
func NewService(r Repository, courses CoursesChecker) Service {
	return service.New(r, courses)
}

// RegisterRoutes mounts the evaluations routes onto the given Gin route group.
// The group must already carry JWT + RequireRole("creador") middleware.
func RegisterRoutes(creatorGrp *gin.RouterGroup, svc Service) {
	handler.Register(creatorGrp, svc)
}
