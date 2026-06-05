// Package certificates is the public API of the certificates module.
// All cross-module imports MUST go through this package — never into
// certificates/repository, certificates/service, or certificates/handler directly.
//
// Mirrors the courses.go and evaluations.go facade pattern exactly.
package certificates

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"
)

// Re-export the public types so callers only need to import "certificates".
type (
	// Service is the public interface of the certificates domain.
	Service = service.Service
	// Repository is the data-access contract for the certificates module.
	Repository = repository.Repository

	// CertificateModel is the service-layer read model for a certificate.
	CertificateModel = service.CertificateModel
	// BadgeModel is the service-layer read model for an earned badge.
	BadgeModel = service.BadgeModel
	// RankingModel is the service-layer read model for a ranking entry.
	RankingModel = service.RankingModel
	// DownloadResult is returned by GetDownloadURL.
	DownloadResult = service.DownloadResult

	// UserNameReader is the narrow read seam into the users module.
	UserNameReader = service.UserNameReader
	// CourseTituloReader is the narrow read seam into the courses module.
	CourseTituloReader = service.CourseTituloReader
)

// Re-export sentinels so main.go and tests can use errors.Is without importing internals.
var (
	ErrCertificateNotFound = service.ErrCertificateNotFound
	ErrNotOwner            = service.ErrNotOwner
	ErrNoPDF               = service.ErrNoPDF
)

// NewRepository constructs a GORM-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return repository.New(db)
}

// NewService constructs a Service backed by the given Repository and cross-module seams.
func NewService(
	repo Repository,
	store storage.Client,
	userNames UserNameReader,
	courseTitulos CourseTituloReader,
	presignTTL time.Duration,
) Service {
	return service.New(repo, store, userNames, courseTitulos, presignTTL)
}

// RegisterRoutes mounts the certificates and badges routes onto the given JWT-protected route group.
func RegisterRoutes(protected *gin.RouterGroup, svc Service) {
	handler.Register(protected, svc)
}
