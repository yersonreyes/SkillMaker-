// Package courses is the public API of the courses module.
// All cross-module imports MUST go through this package — never into
// courses/repository, courses/service, or courses/handler directly.
//
// Mirrors the users.go facade pattern exactly.
package courses

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
)

// Re-export the public types so callers only need to import "courses".
type (
	Service       = service.Service
	Repository    = repository.Repository
	CourseModel   = service.CourseModel
	CreateRequest = service.CreateRequest
	UpdateRequest = service.UpdateRequest
)

// NewRepository constructs a GORM-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return repository.New(db)
}

// NewService constructs a Service that delegates to the given Repository.
func NewService(r Repository) Service {
	return service.New(r)
}

// RegisterRoutes mounts the courses routes onto the given Gin route group.
// The group must already carry JWT + RequireRole("creador") middleware.
func RegisterRoutes(creatorGrp *gin.RouterGroup, svc Service) {
	handler.Register(creatorGrp, svc)
}
