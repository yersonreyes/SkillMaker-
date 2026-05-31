// Package auth is the public API of the authentication module.
// All cross-module imports MUST go through this package — never into
// auth/service, auth/repository, or auth/handler directly.
// This enforces encapsulation: the internal packages can be refactored
// freely without breaking callers.
package auth

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users"
)

// Re-export the public types so callers only need to import "auth".
type (
	Service    = service.Service
	Config     = service.Config
	Repository = repository.Repository
)

// NewRepository constructs a GORM-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return repository.New(db)
}

// NewService constructs a Service with the injected config, users service,
// and refresh-token repository.
func NewService(cfg Config, u users.Service, r Repository) Service {
	return service.New(cfg, u, r)
}

// RegisterRoutes mounts POST /auth/google, /auth/refresh, /auth/logout
// under the provided router group (e.g. /api).
func RegisterRoutes(rg *gin.RouterGroup, svc Service) {
	handler.Register(rg, svc)
}
