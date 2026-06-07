// Package service contains the business logic for the certificates module.
// It is HTTP-agnostic: it returns domain sentinels and read-models.
// Handlers are responsible for mapping sentinels → HTTP status codes.
//
// Cross-module seams: UserNameReader (from users), CourseTituloReader (from courses).
// These narrow interfaces are declared HERE — the consumer declares the interface (ADR-1 / evaluations precedent).
// certificates imports ONLY users + courses FACADES — NEVER evaluations (would create a cycle).
package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/service/pdf"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"
)

// ── Notifier seam (NON-FATAL, cross-module, notifications-inapp) ──────────────

// Notifier is the narrow outbound seam for sending notifications.
// notifications.Service satisfies this structurally (duck typing — no import).
// Each consuming module declares its OWN narrow interface; there is NO shared package.
type Notifier interface {
	Notify(ctx context.Context, userID, tipo, titulo, cuerpo, refID string) error
}

// Option is a functional option for the certificates service constructor.
type Option func(*serviceImpl)

// WithNotifier wires a Notifier into the certificates service.
// If not provided, the notifier field is nil and Notify calls are skipped safely.
func WithNotifier(n Notifier) Option {
	return func(s *serviceImpl) { s.notifier = n }
}

// ── Cross-module seams ─────────────────────────────────────────────────────────

// UserNameReader is the narrow read seam into the users module.
// users.Service.GetByID returns *UserSummary (signature mismatch) → thin adapter in main.go.
type UserNameReader interface {
	GetUserNombre(ctx context.Context, userID string) (string, error)
}

// CourseTituloReader is the narrow read seam into the courses module.
// courses.Service.GetCourseTitulo (added in C5.1) satisfies this structurally — pass coursesSvc directly.
type CourseTituloReader interface {
	GetCourseTitulo(ctx context.Context, courseID string) (string, error)
}

// ── Read models ────────────────────────────────────────────────────────────────

// CertificateModel is the service-layer read model returned to handlers and DTOs.
type CertificateModel struct {
	ID           string
	UserID       string
	CourseID     string
	CourseTitulo string
	Codigo       string
	StorageKey   string
	EmitidoEn    time.Time
}

// BadgeModel is the service-layer read model for an earned badge.
type BadgeModel struct {
	ID          string
	Nombre      string
	Descripcion string
	OtorgadoEn  time.Time
}

// RankingModel is the service-layer read model for a ranking entry.
type RankingModel struct {
	UserID string
	Nombre string
	Total  int64
}

// DownloadResult is returned by GetDownloadURL containing the presigned URL and expiry.
type DownloadResult struct {
	URL       string
	ExpiresAt time.Time
}

// ── Service interface ──────────────────────────────────────────────────────────

// Service is the public interface of the certificates domain.
// Other modules (handlers, facade, evaluations via CertificateIssuer seam) depend on this.
type Service interface {
	// IssueOnPass satisfies evaluations.CertificateIssuer (FROZEN signature).
	// Idempotent: if a certificate already exists for (userID, courseID) → returns nil.
	// Non-fatal: evaluations.SubmitAttempt swallows the returned error (non-fatal seam).
	IssueOnPass(ctx context.Context, userID, courseID string) error

	// ListMyCertificates returns all certificates for userID ordered by emitido_en DESC.
	ListMyCertificates(ctx context.Context, userID string) ([]CertificateModel, error)

	// GetCertificate returns the certificate for the given ID, owner-scoped.
	// Returns ErrCertificateNotFound if the cert does not exist OR the caller is not the owner.
	// Anti-enumeration: non-owners get 404 (hides existence), mirrors courses read path.
	GetCertificate(ctx context.Context, certID, userID string) (*CertificateModel, error)

	// GetDownloadURL returns a presigned GET URL for the certificate PDF, owner-scoped.
	// Returns ErrCertificateNotFound for non-owners (anti-enum).
	// Returns ErrNoPDF sentinel if storage_key is empty (issued but PDF failed previously).
	GetDownloadURL(ctx context.Context, certID, userID string) (DownloadResult, error)

	// ListMyBadges returns all earned badges for userID.
	ListMyBadges(ctx context.Context, userID string) ([]BadgeModel, error)

	// Ranking returns the top-n users by certificate count (0-cert users excluded).
	Ranking(ctx context.Context, n int) ([]RankingModel, error)

	// EvaluateBadges counts the user's certificates and awards all qualifying badges idempotently.
	// Called internally by IssueOnPass after persisting the certificate.
	// Errors are swallowed by IssueOnPass (non-fatal).
	EvaluateBadges(ctx context.Context, userID string) error
}

// ── Repository interface (declared here for the service to depend on) ─────────

// Repository is the narrow data-access seam for the certificates service.
// The full implementation lives in certificates/repository.
type Repository interface {
	GetByUserCourse(ctx context.Context, userID, courseID string) (*domain.Certificate, error)
	Create(ctx context.Context, cert *domain.Certificate) error
	GetByID(ctx context.Context, certID string) (*domain.Certificate, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Certificate, error)
	CountByUser(ctx context.Context, userID string) (int64, error)
	ListBadgesByUser(ctx context.Context, userID string) ([]BadgeWithGrant, error)
	ListBadgesUpToThreshold(ctx context.Context, count int64) ([]domain.Badge, error)
	AwardBadge(ctx context.Context, userID, badgeID string) error
	Ranking(ctx context.Context, n int) ([]RankingRow, error)
}

// BadgeWithGrant is a read model joining badge + user_badge.otorgado_en.
type BadgeWithGrant struct {
	ID          string
	Nombre      string
	Descripcion string
	OtorgadoEn  time.Time
}

// RankingRow is the raw aggregate returned by the repository Ranking query.
type RankingRow struct {
	UserID string
	Nombre string
	Total  int64
}

// ── concrete implementation ────────────────────────────────────────────────────

type serviceImpl struct {
	repo          Repository
	store         storage.Client
	userNames     UserNameReader
	courseTitulos CourseTituloReader
	renderPDF     func(nombre, titulo string, fecha time.Time, codigo string) ([]byte, error)
	presignTTL    time.Duration
	notifier      Notifier // nil-safe; wired via WithNotifier in main.go
}

// New creates a Service backed by the given Repository, storage Client, and cross-module seams.
// Variadic opts preserve backward compatibility: existing 5-arg call sites stay valid.
func New(
	repo Repository,
	store storage.Client,
	userNames UserNameReader,
	courseTitulos CourseTituloReader,
	presignTTL time.Duration,
	opts ...Option,
) Service {
	s := &serviceImpl{
		repo:          repo,
		store:         store,
		userNames:     userNames,
		courseTitulos: courseTitulos,
		renderPDF:     pdf.RenderCertificate,
		presignTTL:    presignTTL,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// ── IssueOnPass ────────────────────────────────────────────────────────────────

// IssueOnPass implements evaluations.CertificateIssuer (FROZEN signature).
// Pipeline (Design §5b):
//  1. Idempotency check — if cert exists → return nil (no regen, no re-award).
//  2. Read user nombre + course titulo.
//  3. Generate codigo (crypto/rand base32); retry once on UNIQUE(codigo) collision.
//  4. Render PDF (fpdf A4 landscape).
//  5. PutObject to storage (key = certificates/{certID}.pdf).
//  6. Persist certificate row (ON CONFLICT (user,course) DO NOTHING — double guard).
//  7. EvaluateBadges — non-fatal (log + swallow).
func (s *serviceImpl) IssueOnPass(ctx context.Context, userID, courseID string) error {
	// 1. Idempotency.
	if _, err := s.repo.GetByUserCourse(ctx, userID, courseID); err == nil {
		return nil // already issued — no regen
	}

	// 2. Read seams.
	nombre, err := s.userNames.GetUserNombre(ctx, userID)
	if err != nil {
		return err
	}
	titulo, err := s.courseTitulos.GetCourseTitulo(ctx, courseID)
	if err != nil {
		return err
	}

	// 3. Generate unique codigo.
	codigo, err := genCodigo()
	if err != nil {
		return err
	}

	// 4. Render PDF.
	now := time.Now()
	certID := uuid.New().String()
	pdfBytes, err := s.renderPDF(nombre, titulo, now, codigo)
	if err != nil {
		return err
	}

	// 5. Upload PDF to storage.
	key := "certificates/" + certID + ".pdf"
	if err := s.store.PutObject(ctx, key, bytes.NewReader(pdfBytes), int64(len(pdfBytes)), "application/pdf"); err != nil {
		return err
	}

	// 6. Persist certificate row.
	cert := &domain.Certificate{
		ID:         certID,
		UserID:     userID,
		CourseID:   courseID,
		Codigo:     codigo,
		StorageKey: key,
		EmitidoEn:  now,
	}
	if err := s.repo.Create(ctx, cert); err != nil {
		// ON CONFLICT (user_id, course_id) DO NOTHING means a concurrent issue already succeeded.
		// The error from GORM for 0 rows affected is not an error — check for real errors.
		if !errors.Is(err, ErrCertificateNotFound) {
			return err
		}
	}

	// 7. Evaluate badges — non-fatal.
	if err := s.EvaluateBadges(ctx, userID); err != nil {
		slog.Error("certificates: EvaluateBadges failed (non-fatal)", "userID", userID, "err", err)
	}

	// 8. NON-FATAL notify — fires ONLY on FIRST issue (not on the idempotency path at step 1).
	if s.notifier != nil {
		if err := s.notifier.Notify(ctx, userID, "certificado_emitido", "Certificado emitido", titulo, certID); err != nil {
			slog.Error("certificates: notify seam failed (non-fatal)", "err", err, "userID", userID)
		}
	}

	return nil
}

// ── ListMyCertificates ────────────────────────────────────────────────────────

func (s *serviceImpl) ListMyCertificates(ctx context.Context, userID string) ([]CertificateModel, error) {
	certs, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]CertificateModel, 0, len(certs))
	for i := range certs {
		out = append(out, toCertModel(&certs[i]))
	}
	return out, nil
}

// ── GetCertificate ────────────────────────────────────────────────────────────

func (s *serviceImpl) GetCertificate(ctx context.Context, certID, userID string) (*CertificateModel, error) {
	cert, err := s.repo.GetByID(ctx, certID)
	if err != nil {
		return nil, ErrCertificateNotFound // sentinel; hides existence
	}
	if cert.UserID != userID {
		return nil, ErrCertificateNotFound // anti-enum: 404 not 403
	}
	m := toCertModel(cert)
	return &m, nil
}

// ── GetDownloadURL ────────────────────────────────────────────────────────────

// ErrNoPDF is returned when a certificate exists but has an empty storage_key
// (e.g. the PDF upload failed previously). Handlers map this to 404 with CERT_NO_PDF.
var ErrNoPDF = errors.New("certificate has no PDF")

func (s *serviceImpl) GetDownloadURL(ctx context.Context, certID, userID string) (DownloadResult, error) {
	cert, err := s.repo.GetByID(ctx, certID)
	if err != nil {
		return DownloadResult{}, ErrCertificateNotFound
	}
	if cert.UserID != userID {
		return DownloadResult{}, ErrCertificateNotFound // anti-enum
	}
	if cert.StorageKey == "" {
		return DownloadResult{}, ErrNoPDF
	}
	expiresAt := time.Now().Add(s.presignTTL)
	url, err := s.store.PresignGetURL(ctx, cert.StorageKey, s.presignTTL)
	if err != nil {
		return DownloadResult{}, err
	}
	return DownloadResult{URL: url, ExpiresAt: expiresAt}, nil
}

// ── ListMyBadges ──────────────────────────────────────────────────────────────

func (s *serviceImpl) ListMyBadges(ctx context.Context, userID string) ([]BadgeModel, error) {
	rows, err := s.repo.ListBadgesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]BadgeModel, 0, len(rows))
	for _, r := range rows {
		out = append(out, BadgeModel(r))
	}
	return out, nil
}

// ── Ranking ───────────────────────────────────────────────────────────────────

func (s *serviceImpl) Ranking(ctx context.Context, n int) ([]RankingModel, error) {
	rows, err := s.repo.Ranking(ctx, n)
	if err != nil {
		return nil, err
	}
	out := make([]RankingModel, 0, len(rows))
	for _, r := range rows {
		out = append(out, RankingModel(r))
	}
	return out, nil
}

// ── EvaluateBadges ────────────────────────────────────────────────────────────

// EvaluateBadges counts the user's certificates and awards all qualifying badges idempotently.
// Uses ON CONFLICT (user_id, badge_id) DO NOTHING at the DB level — safe to call repeatedly.
func (s *serviceImpl) EvaluateBadges(ctx context.Context, userID string) error {
	count, err := s.repo.CountByUser(ctx, userID)
	if err != nil {
		return err
	}
	badges, err := s.repo.ListBadgesUpToThreshold(ctx, count)
	if err != nil {
		return err
	}
	for _, badge := range badges {
		if err := s.repo.AwardBadge(ctx, userID, badge.ID); err != nil {
			return err
		}
	}
	return nil
}

// ── to-model converters ────────────────────────────────────────────────────────

func toCertModel(c *domain.Certificate) CertificateModel {
	return CertificateModel{
		ID:         c.ID,
		UserID:     c.UserID,
		CourseID:   c.CourseID,
		Codigo:     c.Codigo,
		StorageKey: c.StorageKey,
		EmitidoEn:  c.EmitidoEn,
	}
}
