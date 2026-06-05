// Package reporting is the public API of the reporting module (C6.1).
// All cross-module imports MUST go through this package — never into
// reporting/repository, reporting/service, or reporting/handler directly.
//
// This module is a pure-SQL, read-only cross-module exception (REQ-MODULE).
// It holds a *gorm.DB and queries other modules' tables directly via Raw SQL.
// NO migration added. NO domain package (read-only, no entities).
//
// Mirrors the certificates.go facade pattern exactly.
package reporting

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting/service"
)

// Re-export the public interfaces so callers only need to import "reporting".
type (
	// Service is the public interface of the reporting domain.
	Service = service.Service

	// Repository is the data-access contract for the reporting module.
	Repository = repository.Repository
)

// NewRepository constructs a GORM-backed Repository.
// db must be the application *gorm.DB (all migrations applied).
func NewRepository(db *gorm.DB) Repository {
	return repository.New(db)
}

// NewService constructs a Service backed by the given Repository.
func NewService(repo Repository) Service {
	return service.New(repo)
}

// RegisterAdminRoutes mounts the admin-gated reporting routes.
// adminGrp must carry JWT + RequireRole("administrador") middleware.
// Routes: GET /reports/global, GET /reports/courses.
func RegisterAdminRoutes(adminGrp *gin.RouterGroup, svc Service) {
	handler.RegisterAdminRoutes(adminGrp, svc)
}

// RegisterSupervisorRoutes mounts the supervisor-gated reporting routes.
// supervisorGrp must carry JWT + RequireRole("supervisor") middleware.
// Routes: GET /reports/team.
func RegisterSupervisorRoutes(supervisorGrp *gin.RouterGroup, svc Service) {
	handler.RegisterSupervisorRoutes(supervisorGrp, svc)
}

// RegisterSelfRoutes mounts the JWT-only route with in-handler admin-or-self authz.
// protected must carry JWT middleware only (no RequireRole).
// Routes: GET /reports/users/:id/progress.
func RegisterSelfRoutes(protected *gin.RouterGroup, svc Service) {
	handler.RegisterSelfRoutes(protected, svc)
}
