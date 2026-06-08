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
	// VerifyResult is the public read model returned by VerifyCertificate.
	VerifyResult = service.VerifyResult

	// UserNameReader is the narrow read seam into the users module.
	UserNameReader = service.UserNameReader
	// CourseTituloReader is the narrow read seam into the courses module.
	CourseTituloReader = service.CourseTituloReader

	// Notifier is the narrow outbound seam for notifications (notifications-inapp).
	// Re-exported so main.go can pass notifications.Service without importing internals.
	Notifier = service.Notifier

	// Option is a functional option for the certificates service constructor.
	Option = service.Option
)

// WithNotifier wires a Notifier into the certificates service.
// notifications.Service satisfies this structurally (duck typing).
var WithNotifier = service.WithNotifier

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
// Variadic opts: pass certificates.WithNotifier(notificationsSvc) to wire the Notifier.
func NewService(
	repo Repository,
	store storage.Client,
	userNames UserNameReader,
	courseTitulos CourseTituloReader,
	presignTTL time.Duration,
	opts ...Option,
) Service {
	return service.New(repo, store, userNames, courseTitulos, presignTTL, opts...)
}

// RegisterRoutes mounts the certificates and badges routes onto the given JWT-protected route group.
func RegisterRoutes(protected *gin.RouterGroup, svc Service) {
	handler.Register(protected, svc)
}

// RegisterPublicRoutes mounts the PUBLIC (no-JWT) certificate verification route
// onto the given public route group (the unauthenticated /api group).
func RegisterPublicRoutes(public *gin.RouterGroup, svc Service) {
	handler.RegisterPublic(public, svc)
}
