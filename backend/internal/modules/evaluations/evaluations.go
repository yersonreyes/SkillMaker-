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

	// EnrollmentCompleter is the forward seam for marking enrollments completed (C2.4).
	EnrollmentCompleter = service.EnrollmentCompleter
	// CertificateIssuer is the forward seam for issuing certificates on pass (C5.1).
	CertificateIssuer = service.CertificateIssuer
	// Option configures optional service dependencies via functional options (ADR-A).
	Option = service.Option
)

// NewRepository constructs a GORM-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return repository.New(db)
}

// NewService constructs a Service that delegates to the given Repository and CoursesChecker.
// courses.Service satisfies CoursesChecker structurally (ADR-1) — pass coursesSvc directly.
// Forward seams (EnrollmentCompleter, CertificateIssuer) are injected via functional options (ADR-A).
// The existing 2-arg call in main.go stays valid — opts is variadic.
func NewService(r Repository, courses CoursesChecker, opts ...Option) Service {
	return service.New(r, courses, opts...)
}

// WithEnrollmentCompleter returns an Option that injects an EnrollmentCompleter seam.
// Wire in C2.4: evaluations.NewService(repo, courses, evaluations.WithEnrollmentCompleter(enrollSvc)).
func WithEnrollmentCompleter(ec EnrollmentCompleter) Option {
	return service.WithEnrollmentCompleter(ec)
}

// WithCertificateIssuer returns an Option that injects a CertificateIssuer seam.
// Wire in C5.1: evaluations.NewService(repo, courses, evaluations.WithCertificateIssuer(certSvc)).
func WithCertificateIssuer(ci CertificateIssuer) Option {
	return service.WithCertificateIssuer(ci)
}

// RegisterRoutes mounts the creator evaluations routes onto the given Gin route group.
// The group must already carry JWT + RequireRole("creador") middleware.
func RegisterRoutes(creatorGrp *gin.RouterGroup, svc Service) {
	handler.Register(creatorGrp, svc)
}

// RegisterStudentRoutes mounts the student attempt routes onto a JWT-only route group.
// Student routes use the protected group (no RequireRole restriction).
func RegisterStudentRoutes(protectedGrp *gin.RouterGroup, svc Service) {
	handler.RegisterStudent(protectedGrp, svc)
}
