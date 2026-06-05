// Package service contains the business logic for the courses module.
// It is HTTP-agnostic: it returns domain sentinels and read-models.
// Handlers are responsible for mapping sentinels → HTTP status codes.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"
)

// ── Read models ────────────────────────────────────────────────────────────────

// CourseModel is the service-layer read model returned to handlers and DTOs.
// Callers never import the domain package directly; they use CourseModel.
type CourseModel struct {
	ID          string
	CreadorID   string
	Titulo      string
	Descripcion string
	Estado      domain.Estado
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SectionModel is the service-layer read model for a section.
type SectionModel struct {
	ID        string
	CourseID  string
	Titulo    string
	Orden     int
	CreatedAt time.Time
}

// SectionWithVideosModel is the service-layer read model for the nested content tree.
// Used by ListContent to return sections with their videos already fetched and nested.
type SectionWithVideosModel struct {
	Section SectionModel
	Videos  []VideoModel
}

// VideoModel is the service-layer read model for a video.
type VideoModel struct {
	ID        string
	SectionID string
	Titulo    string
	URL       string
	Proveedor string
	DuracionS int
	Orden     int
	CreatedAt time.Time
}

// CourseSummary is the cross-module read model returned by ListByEstado.
// It is the canonical type for the approvals seam — approvals' CourseStateManager
// declares ListByEstado returning []CourseSummary (matching evaluations' precedent of
// importing the courses facade type rather than defining a local one).
type CourseSummary struct {
	ID          string
	CreadorID   string
	Titulo      string
	Descripcion string
	Estado      string
	CreatedAt   time.Time
	PublicadoEn *time.Time
}

// ── Request types ──────────────────────────────────────────────────────────────

// CreateRequest carries the caller-supplied data for creating a course.
// Estado is intentionally absent — the service always forces borrador (AC1).
type CreateRequest struct {
	Titulo      string
	Descripcion string
}

// UpdateRequest carries optional fields for a partial course update.
// Nil pointer = field not supplied by the caller = leave as-is in DB.
// This matches design §4: field-map so zero-value strings can be written.
type UpdateRequest struct {
	Titulo      *string
	Descripcion *string
}

// SectionCreateRequest carries data for creating a section.
type SectionCreateRequest struct {
	CourseID string
	Titulo   string
}

// SectionUpdateRequest carries optional fields for a partial section update.
type SectionUpdateRequest struct {
	Titulo *string
}

// VideoCreateRequest carries data for creating a video.
type VideoCreateRequest struct {
	SectionID string
	Titulo    string
	URL       string
	Proveedor string
	DuracionS int
}

// VideoUpdateRequest carries optional fields for a partial video update.
type VideoUpdateRequest struct {
	Titulo    *string
	URL       *string
	Proveedor *string
	DuracionS *int
}

// ReorderRequest carries the desired section order for a course.
type ReorderRequest struct {
	CourseID string
	IDs      []string
}

// ── Material types (C2.3) ──────────────────────────────────────────────────────

// MaterialModel is the service-layer read model for a material attachment.
type MaterialModel struct {
	ID          string
	CourseID    string
	Titulo      string
	StorageKey  string
	MimeType    string
	TamanoBytes int64
	CreatedAt   time.Time
}

// PresignInput carries data for requesting a presigned upload URL.
type PresignInput struct {
	Nombre      string
	ContentType string
	TamanoBytes int64
}

// PresignResult is returned by PresignUpload containing the upload URL and object key.
type PresignResult struct {
	UploadURL string
	Key       string
	ExpiresAt time.Time
}

// ConfirmInput carries data for confirming a completed upload.
type ConfirmInput struct {
	Key         string
	Nombre      string
	ContentType string
	TamanoBytes int64
}

// DownloadResult is returned by PresignDownload containing the download URL.
type DownloadResult struct {
	URL       string
	ExpiresAt time.Time
}

// ── Service interface ──────────────────────────────────────────────────────────

// Service is the public interface of the courses domain.
// Other modules (handlers) depend on this interface — never on serviceImpl.
type Service interface {
	// Create persists a new course with estado=borrador (forced, regardless of
	// any client input). creadorID is always taken from the JWT, not from the body.
	Create(ctx context.Context, creadorID string, req CreateRequest) (*CourseModel, error)

	// GetByID returns the course for the given owner.
	// Returns ErrNotOwner if creadorID does not match the course's creador_id.
	// Returns ErrCourseNotFound if no course with that id exists.
	GetByID(ctx context.Context, id, creadorID string) (*CourseModel, error)

	// UpdateByID partially updates a course the caller owns.
	// Ownership is checked BEFORE the estado transition guard (D5 ordering).
	// Returns ErrNotOwner for non-owners; ErrInvalidTransition when estado ∉ {borrador, rechazado}.
	UpdateByID(ctx context.Context, id, creadorID string, req UpdateRequest) (*CourseModel, error)

	// ListByCreator returns a paginated page of courses owned by creadorID.
	ListByCreator(ctx context.Context, creadorID string, p pagination.Params) (pagination.Page[CourseModel], error)

	// ── Section methods (C2.2) ───────────────────────────────────────────────

	// CreateSection creates a new section on a course the caller owns.
	CreateSection(ctx context.Context, creadorID string, req SectionCreateRequest) (*SectionModel, error)

	// GetSectionByID returns the section (no ownership check — read-only, used internally).
	GetSectionByID(ctx context.Context, id string) (*SectionModel, error)

	// UpdateSection partially updates a section the caller owns.
	UpdateSection(ctx context.Context, id, creadorID string, req SectionUpdateRequest) (*SectionModel, error)

	// DeleteSection deletes a section the caller owns (cascades to videos).
	DeleteSection(ctx context.Context, id, creadorID string) error

	// ListSections returns all sections for a course ordered by orden ASC.
	ListSections(ctx context.Context, courseID string) ([]SectionModel, error)

	// ListContent returns the nested content tree for a course the caller owns:
	// sections ordered by orden, each with their videos ordered by orden.
	// Owner-gated: returns ErrNotOwner if creadorID does not own the course.
	// Approach: one ListSectionsByCourse query + one ListVideosBySection per section
	// (N+1 avoided in practice: section counts are tiny; avoids join complexity).
	ListContent(ctx context.Context, courseID, creadorID string) ([]SectionWithVideosModel, error)

	// ReorderSections reorders the sections of a course.
	// ids must be the EXACT full set of section IDs for the course.
	ReorderSections(ctx context.Context, courseID, creadorID string, ids []string) error

	// ── Video methods (C2.2) ─────────────────────────────────────────────────

	// CreateVideo creates a new video in a section the caller owns.
	CreateVideo(ctx context.Context, creadorID string, req VideoCreateRequest) (*VideoModel, error)

	// UpdateVideo partially updates a video the caller owns.
	UpdateVideo(ctx context.Context, id, creadorID string, req VideoUpdateRequest) (*VideoModel, error)

	// DeleteVideo deletes a video the caller owns.
	DeleteVideo(ctx context.Context, id, creadorID string) error

	// ListVideos returns all videos for a section ordered by orden ASC.
	ListVideos(ctx context.Context, sectionID string) ([]VideoModel, error)

	// HasContent returns true if the course has at least one video.
	// Owner-gated: returns ErrNotOwner if creadorID does not own the course.
	HasContent(ctx context.Context, courseID, creadorID string) (bool, error)

	// ── Material methods (C2.3) ──────────────────────────────────────────────

	// PresignUpload generates a presigned PUT URL for uploading a material file.
	// Owner-gated (write): returns ErrNotOwner (403) for non-owners.
	// Estado-gated: returns ErrInvalidTransition (409) for non-editable courses.
	// Validates file size and MIME type — returns ErrFileTooLarge (413) or ErrMIMENotAllowed (415).
	PresignUpload(ctx context.Context, courseID, creadorID string, req PresignInput) (PresignResult, error)

	// ConfirmUpload persists a material row after a successful presigned PUT.
	// Re-validates size and MIME as defense in depth.
	// Returns ErrInvalidMaterialKey (400) if the key prefix is invalid.
	ConfirmUpload(ctx context.Context, courseID, creadorID string, req ConfirmInput) (*MaterialModel, error)

	// ListMaterials returns all materials for a course ordered by created_at ASC.
	// Owner-gated (read): returns ErrNotOwner (→ 404 via read helper) for non-owners.
	ListMaterials(ctx context.Context, courseID, creadorID string) ([]MaterialModel, error)

	// PresignDownload generates a presigned GET URL for downloading a material.
	// Owner-gated (read): returns ErrNotOwner (→ 404 via read helper) for non-owners.
	PresignDownload(ctx context.Context, courseID, materialID, creadorID string) (DownloadResult, error)

	// DeleteMaterial deletes a material row and attempts best-effort object deletion.
	// Owner-gated (write): returns ErrNotOwner (→ 403) for non-owners.
	// If the storage delete fails, the error is logged and swallowed (D5).
	DeleteMaterial(ctx context.Context, materialID, creadorID string) error

	// ── Cross-module seam (C3.1) ─────────────────────────────────────────────────

	// GetCourseOwnership returns the creadorID and estado (as plain string) for the course.
	// This is the narrow read seam for cross-module callers (evaluations).
	// evaluations defines a local CoursesChecker interface that this method satisfies structurally.
	// Returns ErrCourseNotFound if no course with that id exists.
	GetCourseOwnership(ctx context.Context, courseID string) (creadorID, estado string, err error)

	// ── C4.1 additions (cross-module seam for approvals) ─────────────────────────

	// SetEstado transitions a course to the given estado.
	// For "aprobado", stamps publicado_en=now() via UpdateEstadoPublicado.
	// For all other valid states, uses UpdateEstado (publicado_en unchanged).
	// Defense-in-depth: returns ErrInvalidTransition for unknown estado values.
	SetEstado(ctx context.Context, courseID, estado string) error

	// ListByEstado returns all courses with the given estado as CourseSummary read-models,
	// ordered by created_at ASC. Used by approvals to list pending courses.
	ListByEstado(ctx context.Context, estado string) ([]CourseSummary, error)
}

// ── concrete implementation ────────────────────────────────────────────────────

type serviceImpl struct {
	repo           repository.Repository
	store          storage.Client
	presignTTL     time.Duration
	maxUploadBytes int64
}

// New creates a Service backed by the given Repository and storage Client.
func New(repo repository.Repository, store storage.Client, presignTTL time.Duration, maxUploadBytes int64) Service {
	return &serviceImpl{
		repo:           repo,
		store:          store,
		presignTTL:     presignTTL,
		maxUploadBytes: maxUploadBytes,
	}
}

// ── Course methods ─────────────────────────────────────────────────────────────

// Create forces estado=borrador and sets creadorID from the argument (never from request).
func (s *serviceImpl) Create(ctx context.Context, creadorID string, req CreateRequest) (*CourseModel, error) {
	c := &domain.Course{
		ID:          uuid.New().String(),
		CreadorID:   creadorID,
		Titulo:      req.Titulo,
		Descripcion: req.Descripcion,
		Estado:      domain.EstadoBorrador, // FORCED — client cannot influence this
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return toModel(c), nil
}

// GetByID fetches the course then enforces ownership.
func (s *serviceImpl) GetByID(ctx context.Context, id, creadorID string) (*CourseModel, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	if c.CreadorID != creadorID {
		return nil, ErrNotOwner // handler maps to 404 (hides existence)
	}
	return toModel(c), nil
}

// UpdateByID enforces ownership FIRST, then transition guard, then applies partial update.
// Ordering rationale: a non-owner editing an aprobado course must get ErrNotOwner (403),
// not ErrInvalidTransition (409) — authz outranks state. See LOAD-BEARING-B.
func (s *serviceImpl) UpdateByID(ctx context.Context, id, creadorID string, req UpdateRequest) (*CourseModel, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapNotFound(err)
	}

	// 1. Ownership check FIRST (handler → 403 on PATCH).
	if c.CreadorID != creadorID {
		return nil, ErrNotOwner
	}

	// 2. Transition guard: only borrador and rechazado allow edits.
	if c.Estado != domain.EstadoBorrador && c.Estado != domain.EstadoRechazado {
		return nil, ErrInvalidTransition
	}

	// 3. Build the field map — only include keys the caller explicitly provided.
	// Using a map (not struct) ensures zero-value strings are written (T1 trade-off).
	fields := map[string]any{
		"updated_at": time.Now(),
	}
	if req.Titulo != nil {
		fields["titulo"] = *req.Titulo
	}
	if req.Descripcion != nil {
		fields["descripcion"] = *req.Descripcion
	}

	if err := s.repo.UpdateByID(ctx, id, fields); err != nil {
		return nil, wrapNotFound(err)
	}

	// 4. Re-read for a fresh model reflecting DB state (including updated_at). See T2.
	return s.GetByID(ctx, id, creadorID)
}

// ListByCreator delegates to the repository and maps the page items.
func (s *serviceImpl) ListByCreator(ctx context.Context, creadorID string, p pagination.Params) (pagination.Page[CourseModel], error) {
	repoPage, err := s.repo.ListByCreator(ctx, creadorID, p)
	if err != nil {
		return pagination.Page[CourseModel]{}, err
	}

	items := make([]CourseModel, 0, len(repoPage.Items))
	for i := range repoPage.Items {
		items = append(items, *toModel(&repoPage.Items[i]))
	}

	return pagination.Page[CourseModel]{
		Items:      items,
		Page:       repoPage.Page,
		Size:       repoPage.Size,
		Total:      repoPage.Total,
		TotalPages: repoPage.TotalPages,
	}, nil
}

// ── Section methods ────────────────────────────────────────────────────────────

// CreateSection creates a section after verifying ownership and course editability.
func (s *serviceImpl) CreateSection(ctx context.Context, creadorID string, req SectionCreateRequest) (*SectionModel, error) {
	if _, err := s.assertCourseEditable(ctx, req.CourseID, creadorID); err != nil {
		return nil, err
	}

	section := &domain.Section{
		ID:       uuid.New().String(),
		CourseID: req.CourseID,
		Titulo:   req.Titulo,
	}
	if err := s.repo.CreateSection(ctx, section); err != nil {
		return nil, err
	}
	return toSectionModel(section), nil
}

// GetSectionByID returns the section (no ownership guard — read helper).
func (s *serviceImpl) GetSectionByID(ctx context.Context, id string) (*SectionModel, error) {
	sec, err := s.repo.GetSectionByID(ctx, id)
	if err != nil {
		return nil, wrapSectionNotFound(err)
	}
	return toSectionModel(sec), nil
}

// UpdateSection updates a section the caller owns on an editable course.
func (s *serviceImpl) UpdateSection(ctx context.Context, id, creadorID string, req SectionUpdateRequest) (*SectionModel, error) {
	sec, _, err := s.loadOwnedSection(ctx, id, creadorID)
	if err != nil {
		return nil, err
	}

	fields := map[string]any{}
	if req.Titulo != nil {
		fields["titulo"] = *req.Titulo
	}
	if len(fields) == 0 {
		return toSectionModel(sec), nil
	}

	if err := s.repo.UpdateSection(ctx, id, fields); err != nil {
		return nil, wrapSectionNotFound(err)
	}

	updated, err := s.repo.GetSectionByID(ctx, id)
	if err != nil {
		return nil, wrapSectionNotFound(err)
	}
	return toSectionModel(updated), nil
}

// DeleteSection deletes a section the caller owns (FK cascade removes child videos).
func (s *serviceImpl) DeleteSection(ctx context.Context, id, creadorID string) error {
	_, _, err := s.loadOwnedSection(ctx, id, creadorID)
	if err != nil {
		return err
	}
	return s.repo.DeleteSection(ctx, id)
}

// ListSections returns all sections for a course ordered by orden ASC.
func (s *serviceImpl) ListSections(ctx context.Context, courseID string) ([]SectionModel, error) {
	sections, err := s.repo.ListSectionsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}
	result := make([]SectionModel, 0, len(sections))
	for i := range sections {
		result = append(result, *toSectionModel(&sections[i]))
	}
	return result, nil
}

// ListContent returns the nested content tree for a course the caller owns.
// Implementation: ownership check (load course, compare CreadorID) → one
// ListSectionsByCourse query + one ListVideosBySection per section, composed in Go.
// This avoids a complex JOIN while keeping N tiny (sections per course < 50 in MVP).
func (s *serviceImpl) ListContent(ctx context.Context, courseID, creadorID string) ([]SectionWithVideosModel, error) {
	// 1. Load course and verify ownership (read route → ErrNotOwner, handler maps to 404).
	c, err := s.repo.GetByID(ctx, courseID)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	if c.CreadorID != creadorID {
		return nil, ErrNotOwner // handler maps to 404 (hides existence) per ERR-1-A
	}

	// 2. Fetch sections for this course, ordered by orden ASC.
	sections, err := s.repo.ListSectionsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}

	// 3. For each section, fetch its videos ordered by orden ASC.
	result := make([]SectionWithVideosModel, 0, len(sections))
	for i := range sections {
		videos, err := s.repo.ListVideosBySection(ctx, sections[i].ID)
		if err != nil {
			return nil, err
		}
		videoModels := make([]VideoModel, 0, len(videos))
		for j := range videos {
			videoModels = append(videoModels, *toVideoModel(&videos[j]))
		}
		result = append(result, SectionWithVideosModel{
			Section: *toSectionModel(&sections[i]),
			Videos:  videoModels,
		})
	}

	return result, nil
}

// ReorderSections reorders sections. ids must be the EXACT full set of section IDs.
func (s *serviceImpl) ReorderSections(ctx context.Context, courseID, creadorID string, ids []string) error {
	if _, err := s.assertCourseEditable(ctx, courseID, creadorID); err != nil {
		return err
	}

	// Validate set-equality: fetch current sections and compare.
	existing, err := s.repo.ListSectionsByCourse(ctx, courseID)
	if err != nil {
		return err
	}

	if len(ids) != len(existing) {
		return fmt.Errorf("%w: ids count %d does not match section count %d", ErrInvalidReorderSet, len(ids), len(existing))
	}

	existingSet := make(map[string]bool, len(existing))
	for _, sec := range existing {
		existingSet[sec.ID] = true
	}
	for _, id := range ids {
		if !existingSet[id] {
			return fmt.Errorf("%w: id %s is not a section of course %s", ErrInvalidReorderSet, id, courseID)
		}
	}

	return s.repo.ReorderSections(ctx, courseID, ids)
}

// ── Video methods ──────────────────────────────────────────────────────────────

// CreateVideo creates a video after verifying section ownership and URL validation.
func (s *serviceImpl) CreateVideo(ctx context.Context, creadorID string, req VideoCreateRequest) (*VideoModel, error) {
	sec, _, err := s.loadOwnedSection(ctx, req.SectionID, creadorID)
	if err != nil {
		return nil, err
	}

	if err := validateVideoURL(req.URL, req.Proveedor); err != nil {
		return nil, err
	}

	// Count existing videos in section for default orden.
	existingVideos, err := s.repo.ListVideosBySection(ctx, sec.ID)
	if err != nil {
		return nil, err
	}

	v := &domain.Video{
		ID:        uuid.New().String(),
		SectionID: req.SectionID,
		Titulo:    req.Titulo,
		URL:       req.URL,
		Proveedor: req.Proveedor,
		DuracionS: req.DuracionS,
		Orden:     len(existingVideos),
	}
	if err := s.repo.CreateVideo(ctx, v); err != nil {
		return nil, err
	}
	return toVideoModel(v), nil
}

// UpdateVideo partially updates a video the caller owns, re-validating URL if changed.
func (s *serviceImpl) UpdateVideo(ctx context.Context, id, creadorID string, req VideoUpdateRequest) (*VideoModel, error) {
	v, _, _, err := s.loadOwnedVideo(ctx, id, creadorID)
	if err != nil {
		return nil, err
	}

	// Re-validate url/proveedor if either is being changed.
	newURL := v.URL
	newProveedor := v.Proveedor
	if req.URL != nil {
		newURL = *req.URL
	}
	if req.Proveedor != nil {
		newProveedor = *req.Proveedor
	}
	if req.URL != nil || req.Proveedor != nil {
		if err := validateVideoURL(newURL, newProveedor); err != nil {
			return nil, err
		}
	}

	fields := map[string]any{}
	if req.Titulo != nil {
		fields["titulo"] = *req.Titulo
	}
	if req.URL != nil {
		fields["url"] = *req.URL
	}
	if req.Proveedor != nil {
		fields["proveedor"] = *req.Proveedor
	}
	if req.DuracionS != nil {
		fields["duracion_s"] = *req.DuracionS
	}

	if len(fields) == 0 {
		return toVideoModel(v), nil
	}

	if err := s.repo.UpdateVideo(ctx, id, fields); err != nil {
		return nil, wrapVideoNotFound(err)
	}

	updated, err := s.repo.GetVideoByID(ctx, id)
	if err != nil {
		return nil, wrapVideoNotFound(err)
	}
	return toVideoModel(updated), nil
}

// DeleteVideo deletes a video the caller owns.
func (s *serviceImpl) DeleteVideo(ctx context.Context, id, creadorID string) error {
	_, _, _, err := s.loadOwnedVideo(ctx, id, creadorID)
	if err != nil {
		return err
	}
	return s.repo.DeleteVideo(ctx, id)
}

// ListVideos returns all videos for a section ordered by orden ASC.
func (s *serviceImpl) ListVideos(ctx context.Context, sectionID string) ([]VideoModel, error) {
	videos, err := s.repo.ListVideosBySection(ctx, sectionID)
	if err != nil {
		return nil, err
	}
	result := make([]VideoModel, 0, len(videos))
	for i := range videos {
		result = append(result, *toVideoModel(&videos[i]))
	}
	return result, nil
}

// HasContent returns true if the course has at least one video (owner-gated).
func (s *serviceImpl) HasContent(ctx context.Context, courseID, creadorID string) (bool, error) {
	c, err := s.repo.GetByID(ctx, courseID)
	if err != nil {
		return false, wrapNotFound(err)
	}
	if c.CreadorID != creadorID {
		return false, ErrNotOwner
	}
	return s.repo.HasContent(ctx, courseID)
}

// ── Cross-module seam (C3.1) ───────────────────────────────────────────────────

// GetCourseOwnership returns the creadorID and estado (as plain string) for the course.
// It is the narrow read seam consumed by the evaluations module via the CoursesChecker interface.
// Returns ErrCourseNotFound if no course with that id exists.
func (s *serviceImpl) GetCourseOwnership(ctx context.Context, courseID string) (creadorID, estado string, err error) {
	c, e := s.repo.GetByID(ctx, courseID)
	if e != nil {
		return "", "", wrapNotFound(e) // repository.ErrCourseNotFound → service.ErrCourseNotFound
	}
	return c.CreadorID, string(c.Estado), nil
}

// ── C4.1 additions ─────────────────────────────────────────────────────────────

// SetEstado transitions a course to the given estado string.
// Routes to UpdateEstadoPublicado when estado=="aprobado" (stamps publicado_en=now()).
// Routes to UpdateEstado for all other valid states (publicado_en unchanged).
// Defense-in-depth: rejects unknown estado values with ErrInvalidTransition.
func (s *serviceImpl) SetEstado(ctx context.Context, courseID, estado string) error {
	est := domain.Estado(estado)
	if !est.Valid() {
		return ErrInvalidTransition
	}
	if est == domain.EstadoAprobado {
		return s.repo.UpdateEstadoPublicado(ctx, courseID, est, time.Now())
	}
	return s.repo.UpdateEstado(ctx, courseID, est)
}

// ListByEstado returns all courses with the given estado as CourseSummary read-models,
// ordered by created_at ASC. Maps domain.Course → CourseSummary via toSummary.
func (s *serviceImpl) ListByEstado(ctx context.Context, estado string) ([]CourseSummary, error) {
	rows, err := s.repo.ListByEstado(ctx, domain.Estado(estado))
	if err != nil {
		return nil, err
	}
	out := make([]CourseSummary, 0, len(rows))
	for i := range rows {
		out = append(out, toSummary(&rows[i]))
	}
	return out, nil
}

// toSummary converts a domain.Course to a CourseSummary read-model.
func toSummary(c *domain.Course) CourseSummary {
	return CourseSummary{
		ID:          c.ID,
		CreadorID:   c.CreadorID,
		Titulo:      c.Titulo,
		Descripcion: c.Descripcion,
		Estado:      string(c.Estado),
		CreatedAt:   c.CreatedAt,
		PublicadoEn: c.PublicadoEn,
	}
}

// ── Private helpers ────────────────────────────────────────────────────────────

// assertCourseEditable loads a course and checks: owner FIRST, then estado.
// LOAD-BEARING: non-owner → ErrNotOwner (403); wrong estado → ErrInvalidTransition (409).
// The order is intentional and locked (C2.1 LOAD-BEARING-B preserved).
func (s *serviceImpl) assertCourseEditable(ctx context.Context, courseID, creadorID string) (*domain.Course, error) {
	c, err := s.repo.GetByID(ctx, courseID)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	// 1. Ownership FIRST.
	if c.CreadorID != creadorID {
		return nil, ErrNotOwner
	}
	// 2. Estado check.
	if c.Estado != domain.EstadoBorrador && c.Estado != domain.EstadoRechazado {
		return nil, ErrInvalidTransition
	}
	return c, nil
}

// loadOwnedSection resolves section → course and asserts editability.
func (s *serviceImpl) loadOwnedSection(ctx context.Context, sectionID, creadorID string) (*domain.Section, *domain.Course, error) {
	sec, err := s.repo.GetSectionByID(ctx, sectionID)
	if err != nil {
		return nil, nil, wrapSectionNotFound(err)
	}
	c, err := s.assertCourseEditable(ctx, sec.CourseID, creadorID)
	if err != nil {
		return nil, nil, err
	}
	return sec, c, nil
}

// loadOwnedVideo resolves video → section → course and asserts editability.
func (s *serviceImpl) loadOwnedVideo(ctx context.Context, videoID, creadorID string) (*domain.Video, *domain.Section, *domain.Course, error) {
	v, err := s.repo.GetVideoByID(ctx, videoID)
	if err != nil {
		return nil, nil, nil, wrapVideoNotFound(err)
	}
	sec, c, err := s.loadOwnedSection(ctx, v.SectionID, creadorID)
	if err != nil {
		return nil, nil, nil, err
	}
	return v, sec, c, nil
}

// validateVideoURL verifies that the URL host matches the declared proveedor.
// youtube → youtube.com or youtu.be; vimeo → vimeo.com. Mismatch → ErrURLProviderMismatch.
// This is a CROSS-FIELD validation (service layer, not DTO) per design §4 D4.
func validateVideoURL(rawURL, proveedor string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ErrURLProviderMismatch
	}
	host := strings.ToLower(parsed.Hostname())
	// Strip www. prefix for normalisation.
	host = strings.TrimPrefix(host, "www.")

	switch proveedor {
	case "youtube":
		if host == "youtube.com" || host == "youtu.be" {
			return nil
		}
		return ErrURLProviderMismatch
	case "vimeo":
		if host == "vimeo.com" {
			return nil
		}
		return ErrURLProviderMismatch
	default:
		// Proveedor already validated by DTO binding (oneof=youtube vimeo).
		// This branch is a defence-in-depth guard.
		return ErrURLProviderMismatch
	}
}

// ── wrap helpers ───────────────────────────────────────────────────────────────

// wrapNotFound converts repository.ErrCourseNotFound to service.ErrCourseNotFound.
func wrapNotFound(err error) error {
	if errors.Is(err, repository.ErrCourseNotFound) {
		return ErrCourseNotFound
	}
	return err
}

// wrapSectionNotFound converts repository.ErrSectionNotFound to service.ErrSectionNotFound.
func wrapSectionNotFound(err error) error {
	if errors.Is(err, repository.ErrSectionNotFound) {
		return ErrSectionNotFound
	}
	return err
}

// wrapVideoNotFound converts repository.ErrVideoNotFound to service.ErrVideoNotFound.
func wrapVideoNotFound(err error) error {
	if errors.Is(err, repository.ErrVideoNotFound) {
		return ErrVideoNotFound
	}
	return err
}

// ── to-model converters ────────────────────────────────────────────────────────

// toModel converts a domain.Course GORM model to a CourseModel read-model.
func toModel(c *domain.Course) *CourseModel {
	return &CourseModel{
		ID:          c.ID,
		CreadorID:   c.CreadorID,
		Titulo:      c.Titulo,
		Descripcion: c.Descripcion,
		Estado:      c.Estado,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

// toSectionModel converts a domain.Section to a SectionModel.
func toSectionModel(s *domain.Section) *SectionModel {
	return &SectionModel{
		ID:        s.ID,
		CourseID:  s.CourseID,
		Titulo:    s.Titulo,
		Orden:     s.Orden,
		CreatedAt: s.CreatedAt,
	}
}

// toVideoModel converts a domain.Video to a VideoModel.
func toVideoModel(v *domain.Video) *VideoModel {
	return &VideoModel{
		ID:        v.ID,
		SectionID: v.SectionID,
		Titulo:    v.Titulo,
		URL:       v.URL,
		Proveedor: v.Proveedor,
		DuracionS: v.DuracionS,
		Orden:     v.Orden,
		CreatedAt: v.CreatedAt,
	}
}
