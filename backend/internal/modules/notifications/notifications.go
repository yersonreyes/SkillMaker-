// Package notifications is the public API of the notifications module.
// All cross-module imports MUST go through this package — never into
// notifications/repository, notifications/service, or notifications/handler directly.
//
// This is a LEAF module: it imports NOBODY from other domain modules.
// consumers (approvals, certificates) declare their OWN narrow Notifier interface;
// notifications.Service satisfies those structurally (duck typing — no import cycle).
package notifications

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/service"
)

// Re-export the public types so callers only need to import "notifications".
type (
	// Service is the public interface of the notifications domain.
	Service = service.Service
	// Repository is the data-access contract for the notifications module.
	Repository = repository.Repository
	// NotificationModel is the service-layer read model for a notification.
	NotificationModel = service.NotificationModel
)

// Re-export sentinel so main.go and tests can use errors.Is without importing internals.
var ErrNotFound = service.ErrNotFound

// NewRepository constructs a GORM-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return repository.New(db)
}

// NewService constructs a Service backed by the given Repository.
func NewService(repo Repository) Service {
	return service.New(repo)
}

// RegisterRoutes mounts the notifications routes onto the given JWT-protected route group.
func RegisterRoutes(protected *gin.RouterGroup, svc Service) {
	handler.Register(protected, svc)
}
