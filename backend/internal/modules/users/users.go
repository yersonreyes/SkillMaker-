// Package users is the public API of the users module.
// All cross-module imports MUST go through this package — never into
// users/repository or users/service directly.
package users

import (
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/service"
)

// Re-export the public types so callers only need to import "users".
type (
	Service       = service.Service
	GoogleProfile = service.GoogleProfile
	UserSummary   = service.UserSummary
	Repository    = repository.Repository
)

// NewRepository constructs a GORM-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return repository.New(db)
}

// NewService constructs a Service that delegates to the given Repository.
func NewService(r Repository) Service {
	return service.New(r)
}
