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
// course-structure-v2: Descripcion and Materiales added.
type VideoModel struct {
	ID          string
	SectionID   string
	Titulo      string
	Descripcion string // migration 0011
	URL         string
	Proveedor   string
	DuracionS   int
	Orden       int
	CreatedAt   time.Time
	Materiales  []MaterialModel // populated only in buildContentTree (per-video)
}

// CategoriaModel is the service-layer read model for a categoria.
type CategoriaModel struct {
	ID     string
	Nombre string
	Slug   string
}

// ── C2.4 catalog read models ───────────────────────────────────────────────────

// CatalogFilter is a type alias for repository.CatalogFilter.
// Defined here so handlers can build service.CatalogFilter without importing the repository package
// directly (R-IMPORTS guard: handler→service-only imports). ADR-1.
type CatalogFilter = repository.CatalogFilter

// CatalogCourseModel is the service-layer read model for an approved-course card.
// course-structure-v2: Nivel, MiniaturaURL, HorasPractico, CantidadClases, HorasVideo, Categorias added.
type CatalogCourseModel struct {
	ID             string
	Titulo         string
	Descripcion    string
	CreadorNombre  string
	CreatedAt      time.Time
	Nivel          *string
	MiniaturaURL   string // presigned URL if MiniaturaKey set, else ""
	HorasPractico  float64
	CantidadClases int
	HorasVideo     float64
	Categorias     []CategoriaModel
}

// MyCourseModel is the service-layer read model for GET /users/me/courses.
type MyCourseModel struct {
	CourseID      string
	Titulo        string
	CreadorNombre string
	Completado    bool
	InscritoEn    time.Time
}

// CatalogDetailModel is a discriminated union for GET /catalog/:id.
// When Enrolled=false, Sections is nil (preview — structural no-leak).
// When Enrolled=true, Sections is populated with per-video materials.
// course-structure-v2: Materiales removed from course level; added per-video in VideoModel.
// Metadata fields (Nivel, Categorias, etc.) always present.
type CatalogDetailModel struct {
	ID             string
	Titulo         string
	Descripcion    string
	CreadorNombre  string
	Enrolled       bool
	Sections       []SectionWithVideosModel // nil when !Enrolled
	Nivel          *string
	MiniaturaURL   string // presigned URL if MiniaturaKey set
	HorasPractico  float64
	CantidadClases int
	HorasVideo     float64
	Categorias     []CategoriaModel
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
// course-structure-v2: Nivel, HorasPractico, CategoriaIDs added.
type CreateRequest struct {
	Titulo        string
	Descripcion   string
	Nivel         *string
	HorasPractico *float64
	CategoriaIDs  []string
}

// UpdateRequest carries optional fields for a partial course update.
// Nil pointer = field not supplied by the caller = leave as-is in DB.
// This matches design §4: field-map so zero-value strings can be written.
// course-structure-v2: Nivel, HorasPractico, CategoriaIDs added.
type UpdateRequest struct {
	Titulo        *string
	Descripcion   *string
	Nivel         *string
	HorasPractico *float64
	CategoriaIDs  []string // nil means "not provided, do not change"; []string{} means "clear all"
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
// course-structure-v2: Descripcion added.
type VideoCreateRequest struct {
	SectionID   string
	Titulo      string
	Descripcion string // optional; defaults to ""
	URL         string
	Proveedor   string
	DuracionS   int
}

// VideoUpdateRequest carries optional fields for a partial video update.
// course-structure-v2: Descripcion added.
type VideoUpdateRequest struct {
	Titulo      *string
	Descripcion *string
	URL         *string
	Proveedor   *string
	DuracionS   *int
}

// ReorderRequest carries the desired section order for a course.
type ReorderRequest struct {
	CourseID string
	IDs      []string
}

// ── Material types (C2.3) ──────────────────────────────────────────────────────

// MaterialModel is the service-layer read model for a material attachment.
// course-structure-v2: VideoID replaces CourseID (material belongs to video now).
type MaterialModel struct {
	ID          string
	VideoID     string // migration 0012: replaces CourseID
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

	// ── Material methods (course-structure-v2: videoID-based) ───────────────────

	// PresignUpload generates a presigned PUT URL for uploading a material file.
	// course-structure-v2: takes videoID; resolves courseID via chain.
	// Owner-gated + Estado-gated. ErrFileTooLarge (413) or ErrMIMENotAllowed (415).
	PresignUpload(ctx context.Context, videoID, callerID string, req PresignInput) (PresignResult, error)

	// ConfirmUpload persists a material row after a successful presigned PUT.
	// course-structure-v2: takes videoID. Re-validates size+MIME (defense in depth).
	ConfirmUpload(ctx context.Context, videoID, callerID string, req ConfirmInput) (*MaterialModel, error)

	// ListMaterialsByVideo returns all materials for a video ordered by created_at ASC.
	// Owner-gated (read): returns ErrNotOwner (→ 404 via read helper) for non-owners.
	ListMaterialsByVideo(ctx context.Context, videoID, callerID string) ([]MaterialModel, error)

	// PresignDownload generates a presigned GET URL for downloading a material.
	// course-structure-v2: no courseID arg; ownership resolved via chain.
	PresignDownload(ctx context.Context, materialID, callerID string) (DownloadResult, error)

	// DeleteMaterial deletes a material row and attempts best-effort object deletion.
	// Owner-gated (write): returns ErrNotOwner (→ 403) for non-owners.
	// If the storage delete fails, the error is logged and swallowed (D5).
	DeleteMaterial(ctx context.Context, materialID, callerID string) error

	// ── Thumbnail methods (course-structure-v2) ──────────────────────────────

	// PresignThumbnail generates a presigned PUT URL for a course thumbnail.
	// Owner-gated + Estado-gated. Image MIME only.
	PresignThumbnail(ctx context.Context, courseID, callerID string, req PresignInput) (PresignResult, error)

	// ConfirmThumbnail sets miniatura_key on the course after a successful thumbnail upload.
	// Owner-gated + Estado-gated.
	ConfirmThumbnail(ctx context.Context, courseID, callerID, key string) error

	// ── Catalog + enrollment methods (C2.4) ─────────────────────────────────────

	// ListCatalog returns a paginated page of aprobado courses filtered by CatalogFilter.
	// Delegates to repo.ListApproved and maps to CatalogCourseModel.
	// The filter is passed verbatim — service does NOT re-validate (handler owns validation, ADR-4).
	ListCatalog(ctx context.Context, p pagination.Params, f CatalogFilter) (pagination.Page[CatalogCourseModel], error)

	// GetCatalogDetail returns the detail for an aprobado course.
	// Branches on enrollment: non-enrolled → preview (Enrolled=false, nil Sections);
	// enrolled → full tree (Enrolled=true, Sections+Materiales populated).
	// Returns ErrCourseNotFound if the course is missing or not aprobado.
	GetCatalogDetail(ctx context.Context, userID, courseID string) (*CatalogDetailModel, error)

	// Enroll enrolls userID in courseID. The course must have estado='aprobado';
	// otherwise returns ErrCourseNotFound (draft-invisibility). Idempotent.
	Enroll(ctx context.Context, userID, courseID string) error

	// ListMyCourses returns all enrollments for userID, ordered by inscritoEn DESC.
	ListMyCourses(ctx context.Context, userID string) ([]MyCourseModel, error)

	// MarkEnrollmentCompleted satisfies the evaluations.EnrollmentCompleter seam.
	// Sets completado=true for (userID, courseID). No-op if no enrollment row exists.
	MarkEnrollmentCompleted(ctx context.Context, userID, courseID string) error

	// ── Categoria methods (course-structure-v2) ─────────────────────────────────

	// ListCategorias returns all curated categorias.
	ListCategorias(ctx context.Context) ([]CategoriaModel, error)

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

	// ── C5.1 additions (cross-module seam for certificates) ──────────────────────

	// GetCourseTitulo returns the course titulo for the cross-module certificates seam.
	// Satisfies certificates/service.CourseTituloReader structurally — pass coursesSvc directly.
	// Returns ErrCourseNotFound if no course with that id exists.
	GetCourseTitulo(ctx context.Context, courseID string) (string, error)
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
// course-structure-v2: validates and sets categoriaIDs, nivel, horasPractico.
func (s *serviceImpl) Create(ctx context.Context, creadorID string, req CreateRequest) (*CourseModel, error) {
	// Validate categoriaIDs if provided.
	if len(req.CategoriaIDs) > 0 {
		ok, err := s.repo.CategoriasExist(ctx, req.CategoriaIDs)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrInvalidCategoria
		}
	}

	c := &domain.Course{
		ID:          uuid.New().String(),
		CreadorID:   creadorID,
		Titulo:      req.Titulo,
		Descripcion: req.Descripcion,
		Estado:      domain.EstadoBorrador, // FORCED — client cannot influence this
	}
	if req.Nivel != nil {
		c.Nivel = req.Nivel
	}
	if req.HorasPractico != nil {
		c.HorasPractico = *req.HorasPractico
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}

	// Set categoriaIDs if provided.
	if len(req.CategoriaIDs) > 0 {
		if err := s.repo.SetCourseCategorias(ctx, c.ID, req.CategoriaIDs); err != nil {
			return nil, err
		}
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
	if req.Nivel != nil {
		fields["nivel"] = *req.Nivel
	}
	if req.HorasPractico != nil {
		fields["horas_practico"] = *req.HorasPractico
	}

	if len(fields) > 1 { // > 1 because updated_at is always added
		if err := s.repo.UpdateByID(ctx, id, fields); err != nil {
			return nil, wrapNotFound(err)
		}
	}

	// Update categoriaIDs if the caller explicitly provided the field (nil = "not set, leave as-is").
	// Empty slice = "clear all associations".
	if req.CategoriaIDs != nil {
		// Validate non-empty ID lists before applying.
		if len(req.CategoriaIDs) > 0 {
			ok, err := s.repo.CategoriasExist(ctx, req.CategoriaIDs)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, ErrInvalidCategoria
			}
		}
		if err := s.repo.SetCourseCategorias(ctx, id, req.CategoriaIDs); err != nil {
			return nil, err
		}
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
//
//nolint:gocritic // hugeParam: VideoCreateRequest is 88b (Descripcion field added in course-structure-v2); value-passing is intentional interface contract.
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
		ID:          uuid.New().String(),
		SectionID:   req.SectionID,
		Titulo:      req.Titulo,
		Descripcion: req.Descripcion, // migration 0011
		URL:         req.URL,
		Proveedor:   req.Proveedor,
		DuracionS:   req.DuracionS,
		Orden:       len(existingVideos),
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
	if req.Descripcion != nil {
		fields["descripcion"] = *req.Descripcion
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

// ── C2.4 catalog + enrollment implementations ─────────────────────────────────

// ListCatalog delegates to repo.ListApproved and maps repository.CatalogCourseModel
// → service.CatalogCourseModel. Handlers never import repository types.
// course-structure-v2: loads categorias in one batch query, presigns miniatura if set.
// catalog-filters: filter is passed verbatim to repo; service does not validate (handler's job).
func (s *serviceImpl) ListCatalog(ctx context.Context, p pagination.Params, f CatalogFilter) (pagination.Page[CatalogCourseModel], error) {
	rp, err := s.repo.ListApproved(ctx, p, f)
	if err != nil {
		return pagination.Page[CatalogCourseModel]{}, err
	}

	// Batch load categorias for all courses (no N+1).
	courseIDs := make([]string, 0, len(rp.Items))
	for i := range rp.Items {
		courseIDs = append(courseIDs, rp.Items[i].ID)
	}
	catsByID, err := s.repo.ListCategoriasForCourses(ctx, courseIDs)
	if err != nil {
		return pagination.Page[CatalogCourseModel]{}, err
	}

	items := make([]CatalogCourseModel, 0, len(rp.Items))
	for i := range rp.Items {
		r := &rp.Items[i]
		miniaturaURL := ""
		if r.MiniaturaKey != nil && *r.MiniaturaKey != "" {
			if u, err := s.store.PresignGetURL(ctx, *r.MiniaturaKey, s.presignTTL); err == nil {
				miniaturaURL = u
			}
		}
		cats := toCategoriaModels(catsByID[r.ID])
		items = append(items, CatalogCourseModel{
			ID:             r.ID,
			Titulo:         r.Titulo,
			Descripcion:    r.Descripcion,
			CreadorNombre:  r.CreadorNombre,
			CreatedAt:      r.CreatedAt,
			Nivel:          r.Nivel,
			MiniaturaURL:   miniaturaURL,
			HorasPractico:  r.HorasPractico,
			CantidadClases: r.CantidadClases,
			HorasVideo:     roundHorasVideo(r.HorasVideo),
			Categorias:     cats,
		})
	}
	return pagination.Page[CatalogCourseModel]{
		Items:      items,
		Page:       rp.Page,
		Size:       rp.Size,
		Total:      rp.Total,
		TotalPages: rp.TotalPages,
	}, nil
}

// GetCatalogDetail returns the course detail for an alumno:
//   - ErrCourseNotFound if missing or not aprobado (draft-invisibility).
//   - Non-enrolled → CatalogDetailModel{Enrolled:false, Sections:nil} + metadata always present.
//   - Enrolled → CatalogDetailModel{Enrolled:true} with full content tree + per-video materials.
//
// course-structure-v2: metadata (nivel, categorias, horasVideo, etc.) always populated.
// IMPORTANT: does NOT reuse ListContent (owner-gated). Uses buildContentTree instead.
func (s *serviceImpl) GetCatalogDetail(ctx context.Context, userID, courseID string) (*CatalogDetailModel, error) {
	d, err := s.repo.GetApprovedDetail(ctx, courseID)
	if err != nil {
		return nil, wrapNotFound(err)
	}

	enrolled, err := s.repo.IsEnrolled(ctx, userID, courseID)
	if err != nil {
		return nil, err
	}

	// Load categorias for this course.
	cats, err := s.repo.GetCourseCategorias(ctx, courseID)
	if err != nil {
		return nil, err
	}

	// Presign miniatura if set.
	miniaturaURL := ""
	if d.MiniaturaKey != nil && *d.MiniaturaKey != "" {
		if u, err := s.store.PresignGetURL(ctx, *d.MiniaturaKey, s.presignTTL); err == nil {
			miniaturaURL = u
		}
	}

	out := &CatalogDetailModel{
		ID:             d.ID,
		Titulo:         d.Titulo,
		Descripcion:    d.Descripcion,
		CreadorNombre:  d.CreadorNombre,
		Enrolled:       enrolled,
		Nivel:          d.Nivel,
		MiniaturaURL:   miniaturaURL,
		HorasPractico:  d.HorasPractico,
		CantidadClases: d.CantidadClases,
		HorasVideo:     roundHorasVideo(d.HorasVideo),
		Categorias:     toCategoriaModels(cats),
	}

	if enrolled {
		sections, err := s.buildContentTree(ctx, courseID)
		if err != nil {
			return nil, err
		}
		out.Sections = sections
	}

	return out, nil
}

// buildContentTree fetches sections+videos+per-video-materials for a course WITHOUT an ownership check.
// Called only from GetCatalogDetail (alumno enrolled → already verified).
// course-structure-v2: materials are now nested per video (no course-level materiales).
// Uses ONE ListMaterialsByCourseVideos query to avoid N+1.
// GOTCHA: do NOT reuse ListContent — it is owner-gated and returns ErrNotOwner.
func (s *serviceImpl) buildContentTree(ctx context.Context, courseID string) ([]SectionWithVideosModel, error) {
	sections, err := s.repo.ListSectionsByCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}

	// Load all materials for all course videos in ONE query (no N+1).
	rawMats, err := s.repo.ListMaterialsByCourseVideos(ctx, courseID)
	if err != nil {
		return nil, err
	}
	// Group materials by video_id in Go.
	matByVideo := make(map[string][]MaterialModel, len(rawMats))
	for i := range rawMats {
		vm := toMaterialModel(&rawMats[i])
		matByVideo[rawMats[i].VideoID] = append(matByVideo[rawMats[i].VideoID], *vm)
	}

	sectionModels := make([]SectionWithVideosModel, 0, len(sections))
	for i := range sections {
		videos, err := s.repo.ListVideosBySection(ctx, sections[i].ID)
		if err != nil {
			return nil, err
		}
		videoModels := make([]VideoModel, 0, len(videos))
		for j := range videos {
			vm := toVideoModel(&videos[j])
			vm.Materiales = matByVideo[videos[j].ID] // attach per-video materials
			videoModels = append(videoModels, *vm)
		}
		sectionModels = append(sectionModels, SectionWithVideosModel{
			Section: *toSectionModel(&sections[i]),
			Videos:  videoModels,
		})
	}

	return sectionModels, nil
}

// Enroll enrolls userID in courseID.
// The course must have estado='aprobado'; otherwise returns ErrCourseNotFound (D5).
// CreateEnrollment is idempotent (ON CONFLICT DO NOTHING).
func (s *serviceImpl) Enroll(ctx context.Context, userID, courseID string) error {
	c, err := s.repo.GetByID(ctx, courseID)
	if err != nil {
		return wrapNotFound(err)
	}
	if c.Estado != domain.EstadoAprobado {
		return ErrCourseNotFound // 404 — draft-invisibility (D5)
	}
	return s.repo.CreateEnrollment(ctx, userID, courseID)
}

// ListMyCourses returns all enrollments for userID, ordered by inscritoEn DESC.
func (s *serviceImpl) ListMyCourses(ctx context.Context, userID string) ([]MyCourseModel, error) {
	rows, err := s.repo.ListEnrollmentsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]MyCourseModel, 0, len(rows))
	for _, r := range rows {
		out = append(out, MyCourseModel{
			CourseID:      r.CourseID,
			Titulo:        r.Titulo,
			CreadorNombre: r.CreadorNombre,
			Completado:    r.Completado,
			InscritoEn:    r.InscritoEn,
		})
	}
	return out, nil
}

// MarkEnrollmentCompleted satisfies evaluations.EnrollmentCompleter.
// Delegates to repo.MarkCompleted which is no-op when no enrollment row exists (D4).
func (s *serviceImpl) MarkEnrollmentCompleted(ctx context.Context, userID, courseID string) error {
	return s.repo.MarkCompleted(ctx, userID, courseID)
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

// ── C5.1 additions ─────────────────────────────────────────────────────────────

// GetCourseTitulo returns the course titulo for the cross-module certificates seam.
// certificates/service declares CourseTituloReader with this method — coursesSvc satisfies structurally.
// Returns ErrCourseNotFound if no course with that id exists.
func (s *serviceImpl) GetCourseTitulo(ctx context.Context, courseID string) (string, error) {
	c, err := s.repo.GetByID(ctx, courseID)
	if err != nil {
		return "", wrapNotFound(err)
	}
	return c.Titulo, nil
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
// course-structure-v2: Descripcion included. Materiales is left nil; buildContentTree populates it.
func toVideoModel(v *domain.Video) *VideoModel {
	return &VideoModel{
		ID:          v.ID,
		SectionID:   v.SectionID,
		Titulo:      v.Titulo,
		Descripcion: v.Descripcion,
		URL:         v.URL,
		Proveedor:   v.Proveedor,
		DuracionS:   v.DuracionS,
		Orden:       v.Orden,
		CreatedAt:   v.CreatedAt,
	}
}

// roundHorasVideo rounds horas_video to 1 decimal place.
// Formula: ROUND(SUM(duracion_s)/3600, 1). Matches REQ-COMPUTED spec.
func roundHorasVideo(raw float64) float64 {
	// Round to 1 decimal: multiply by 10, round to nearest int, divide by 10.
	return float64(int(raw*10+0.5)) / 10
}

// toCategoriaModels converts a slice of domain.Categoria to CategoriaModel read models.
func toCategoriaModels(cats []domain.Categoria) []CategoriaModel {
	if len(cats) == 0 {
		return []CategoriaModel{}
	}
	out := make([]CategoriaModel, 0, len(cats))
	for _, c := range cats {
		out = append(out, CategoriaModel{ID: c.ID, Nombre: c.Nombre, Slug: c.Slug})
	}
	return out
}

// ListCategorias returns all curated categorias.
func (s *serviceImpl) ListCategorias(ctx context.Context) ([]CategoriaModel, error) {
	cats, err := s.repo.ListCategorias(ctx)
	if err != nil {
		return nil, err
	}
	return toCategoriaModels(cats), nil
}

// PresignThumbnail generates a presigned PUT URL for a course thumbnail.
// Owner-gated + Estado-gated. Image MIME only.
func (s *serviceImpl) PresignThumbnail(ctx context.Context, courseID, callerID string, req PresignInput) (PresignResult, error) {
	if _, err := s.assertCourseEditable(ctx, courseID, callerID); err != nil {
		return PresignResult{}, err
	}
	if !allowedImageMIME[req.ContentType] {
		return PresignResult{}, ErrMIMENotAllowed
	}
	key := thumbnailKey(courseID, req.Nombre)
	uploadURL, err := s.store.PresignPutURL(ctx, key, s.presignTTL)
	if err != nil {
		return PresignResult{}, err
	}
	return PresignResult{
		UploadURL: uploadURL,
		Key:       key,
		ExpiresAt: time.Now().Add(s.presignTTL),
	}, nil
}

// ConfirmThumbnail sets miniatura_key on the course after a successful thumbnail upload.
// Owner-gated + Estado-gated.
func (s *serviceImpl) ConfirmThumbnail(ctx context.Context, courseID, callerID, key string) error {
	if _, err := s.assertCourseEditable(ctx, courseID, callerID); err != nil {
		return err
	}
	expectedPrefix := "courses/" + courseID + "/thumbnail/"
	if !strings.HasPrefix(key, expectedPrefix) {
		return ErrInvalidMaterialKey
	}
	return s.repo.UpdateByID(ctx, courseID, map[string]any{
		"miniatura_key": key,
		"updated_at":    time.Now(),
	})
}
