// Package courses is the public API of the courses module.
// All cross-module imports MUST go through this package — never into
// courses/repository, courses/service, or courses/handler directly.
//
// Mirrors the users.go facade pattern exactly.
package courses

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"
)

// Re-export the public types so callers only need to import "courses".
type (
	Service       = service.Service
	Repository    = repository.Repository
	CourseModel   = service.CourseModel
	CreateRequest = service.CreateRequest
	UpdateRequest = service.UpdateRequest

	// CourseSummary is the cross-module read model for the approvals seam (C4.1).
	// Re-exported so approvals can declare its CourseStateManager interface
	// returning []courses.CourseSummary without importing courses internals.
	CourseSummary = service.CourseSummary
)

// ErrCourseNotFound is the sentinel returned by GetCourseOwnership when no course
// with the requested ID exists. Re-exported here so cross-module consumers (e.g.
// evaluations) can use errors.Is instead of fragile string matching.
var ErrCourseNotFound = service.ErrCourseNotFound

// NewRepository constructs a GORM-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return repository.New(db)
}

// NewService constructs a Service that delegates to the given Repository and storage Client.
// presignTTL controls how long presigned URLs are valid; maxUploadBytes is the maximum
// file size allowed for uploads.
func NewService(r Repository, store storage.Client, presignTTL time.Duration, maxUploadBytes int64) Service {
	return service.New(r, store, presignTTL, maxUploadBytes)
}

// RegisterRoutes mounts the courses routes onto the given Gin route group.
// The group must already carry JWT + RequireRole("creador") middleware.
func RegisterRoutes(creatorGrp *gin.RouterGroup, svc Service) {
	handler.Register(creatorGrp, svc)
}
