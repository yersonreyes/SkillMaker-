// Package users is the public API of the users module.
// All cross-module imports MUST go through this package — never into
// users/repository or users/service directly.
package users

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/service"
)

// Re-export the public types so callers only need to import "users".
type (
	Service          = service.Service
	GoogleProfile    = service.GoogleProfile
	UserSummary      = service.UserSummary
	UserDetailModel  = service.UserDetailModel
	SupervisionModel = service.SupervisionModel
	ListFilters      = service.ListFilters
	Repository       = repository.Repository
	Handler          = handler.Handler
)

// NewRepository constructs a GORM-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return repository.New(db)
}

// NewService constructs a Service that delegates to the given Repository.
func NewService(r Repository) Service {
	return service.New(r)
}

// RegisterRoutes mounts the users routes onto two pre-built Gin route groups.
//
//   - admin: must already carry JWT + RequireRole("administrador") middleware
//   - me:    must already carry JWT middleware
//
// PR-B will extend this function to also register supervision routes.
func RegisterRoutes(admin, me *gin.RouterGroup, svc Service) {
	handler.Register(admin, me, svc)
}
