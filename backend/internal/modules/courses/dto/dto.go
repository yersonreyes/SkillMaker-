// Package dto contains the wire shapes for the courses module API.
// DTOs are kept separate from GORM models (domain) and service read-models
// so that JSON concerns do not leak into the domain or service layers.
package dto

import (
	"time"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// ── Course request bodies ──────────────────────────────────────────────────────

// CreateCourseRequest is the body for POST /api/courses.
// titulo is required (min=3). descripcion is optional.
// estado is intentionally absent — the service always forces borrador.
// course-structure-v2: nivel, horasPractico, categoriaIds added.
type CreateCourseRequest struct {
	Titulo        string   `json:"titulo"        binding:"required,min=3,max=200"`
	Descripcion   string   `json:"descripcion"   binding:"omitempty,max=5000"`
	Nivel         *string  `json:"nivel"         binding:"omitempty,oneof=basico intermedio avanzado"`
	HorasPractico *float64 `json:"horasPractico" binding:"omitempty,min=0"`
	CategoriaIDs  []string `json:"categoriaIds"  binding:"omitempty,dive,uuid"`
}

// UpdateCourseRequest is the body for PATCH /api/courses/:id.
// Both fields use pointers to distinguish "not provided" (nil) from "set to empty".
// This enables true partial updates at the service layer.
// course-structure-v2: nivel, horasPractico, categoriaIds added.
type UpdateCourseRequest struct {
	Titulo        *string  `json:"titulo"        binding:"omitempty,min=3,max=200"`
	Descripcion   *string  `json:"descripcion"   binding:"omitempty,max=5000"`
	Nivel         *string  `json:"nivel"         binding:"omitempty,oneof=basico intermedio avanzado"`
	HorasPractico *float64 `json:"horasPractico" binding:"omitempty,min=0"`
	CategoriaIDs  []string `json:"categoriaIds"  binding:"omitempty,dive,uuid"`
}

// ── Section request bodies ─────────────────────────────────────────────────────

// SectionCreateRequest is the body for POST /api/courses/:courseId/sections.
type SectionCreateRequest struct {
	Titulo string `json:"titulo" binding:"required,min=1,max=200"`
}

// SectionUpdateRequest is the body for PATCH /api/sections/:id.
// Pointer — nil means "not provided, do not update".
type SectionUpdateRequest struct {
	Titulo *string `json:"titulo" binding:"omitempty,min=1,max=200"`
}

// ReorderRequest is the body for PATCH /api/courses/:id/sections/reorder.
// ids must contain the EXACT full set of section IDs for the course.
type ReorderRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// ── Video request bodies ───────────────────────────────────────────────────────

// VideoCreateRequest is the body for POST /api/sections/:sectionId/videos.
// proveedor is validated by the DTO (oneof=youtube vimeo).
// url/proveedor cross-validation is done at the service layer (design §4 D4).
// course-structure-v2: descripcion added.
type VideoCreateRequest struct {
	Titulo      string `json:"titulo"      binding:"required,min=1,max=200"`
	Descripcion string `json:"descripcion" binding:"omitempty,max=5000"`
	URL         string `json:"url"         binding:"required,url"`
	Proveedor   string `json:"proveedor"   binding:"required,oneof=youtube vimeo"`
	DuracionS   int    `json:"duracionS"   binding:"omitempty,min=0"`
}

// VideoUpdateRequest is the body for PATCH /api/videos/:id.
// All fields are optional — nil means "not provided, do not update".
// course-structure-v2: descripcion added.
type VideoUpdateRequest struct {
	Titulo      *string `json:"titulo"      binding:"omitempty,min=1,max=200"`
	Descripcion *string `json:"descripcion" binding:"omitempty,max=5000"`
	URL         *string `json:"url"         binding:"omitempty,url"`
	Proveedor   *string `json:"proveedor"   binding:"omitempty,oneof=youtube vimeo"`
	DuracionS   *int    `json:"duracionS"   binding:"omitempty,min=0"`
}

// ── Course response shapes ─────────────────────────────────────────────────────

// CourseDetail is the full course representation returned by POST and GET single.
// hasContent (D5) is computed via svc.HasContent on GET detail; false on Create/Update.
type CourseDetail struct {
	ID          string    `json:"id"`
	CreadorID   string    `json:"creadorId"`
	Titulo      string    `json:"titulo"`
	Descripcion string    `json:"descripcion"`
	Estado      string    `json:"estado"`
	HasContent  bool      `json:"hasContent"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CourseListItem is the compact representation used in paginated list responses.
type CourseListItem struct {
	ID        string    `json:"id"`
	Titulo    string    `json:"titulo"`
	Estado    string    `json:"estado"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ── Section response shapes ────────────────────────────────────────────────────

// SectionResponse is the wire shape for a section returned by the API.
type SectionResponse struct {
	ID        string    `json:"id"`
	CourseID  string    `json:"courseId"`
	Titulo    string    `json:"titulo"`
	Orden     int       `json:"orden"`
	CreatedAt time.Time `json:"createdAt"`
}

// SectionWithVideosResponse is the wire shape for GET /courses/:courseId/sections.
// Returns the full nested content tree: sections with their videos embedded.
// Spec: ERR-1-A (read ownership → 404); frontend curso-editar calls this on page load.
type SectionWithVideosResponse struct {
	ID        string          `json:"id"`
	CourseID  string          `json:"courseId"`
	Titulo    string          `json:"titulo"`
	Orden     int             `json:"orden"`
	CreatedAt time.Time       `json:"createdAt"`
	Videos    []VideoResponse `json:"videos"`
}

// ── Video response shapes ──────────────────────────────────────────────────────

// VideoResponse is the wire shape for a video returned by the API.
// course-structure-v2: Descripcion and Materiales added.
// course-player-progress (Change 2): Completado added (caller-scoped, migration 0014).
type VideoResponse struct {
	ID          string             `json:"id"`
	SectionID   string             `json:"sectionId"`
	Titulo      string             `json:"titulo"`
	Descripcion string             `json:"descripcion"`
	URL         string             `json:"url"`
	Proveedor   string             `json:"proveedor"`
	DuracionS   int                `json:"duracionS"`
	Orden       int                `json:"orden"`
	CreatedAt   time.Time          `json:"createdAt"`
	Materiales  []MaterialResponse `json:"materiales"`
	Completado  bool               `json:"completado"`
}

// ── Mapping functions ──────────────────────────────────────────────────────────

// ToCourseDetail converts a service.CourseModel to the full CourseDetail wire shape.
// hasContent must be computed by the handler via svc.HasContent before calling this.
// Create/Update handlers pass hasContent=false (borrador — content state is re-fetched on reload).
func ToCourseDetail(m *service.CourseModel, hasContent bool) CourseDetail {
	return CourseDetail{
		ID:          m.ID,
		CreadorID:   m.CreadorID,
		Titulo:      m.Titulo,
		Descripcion: m.Descripcion,
		Estado:      string(m.Estado),
		HasContent:  hasContent,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// ToCourseListItem converts a service.CourseModel to the compact CourseListItem shape.
func ToCourseListItem(m *service.CourseModel) CourseListItem {
	return CourseListItem{
		ID:        m.ID,
		Titulo:    m.Titulo,
		Estado:    string(m.Estado),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// ToCourseListPage maps a pagination.Page[service.CourseModel] to a
// pagination.Page[CourseListItem], converting each item with ToCourseListItem.
func ToCourseListPage(p pagination.Page[service.CourseModel]) pagination.Page[CourseListItem] {
	items := make([]CourseListItem, 0, len(p.Items))
	for i := range p.Items {
		items = append(items, ToCourseListItem(&p.Items[i]))
	}
	return pagination.Page[CourseListItem]{
		Items:      items,
		Page:       p.Page,
		Size:       p.Size,
		Total:      p.Total,
		TotalPages: p.TotalPages,
	}
}

// ToSection converts a service.SectionModel to the SectionResponse wire shape.
func ToSection(m *service.SectionModel) SectionResponse {
	return SectionResponse{
		ID:        m.ID,
		CourseID:  m.CourseID,
		Titulo:    m.Titulo,
		Orden:     m.Orden,
		CreatedAt: m.CreatedAt,
	}
}

// ToSectionWithVideos converts a service.SectionWithVideosModel to the nested wire shape.
// Videos are already ordered by orden ASC from the service layer.
// m is passed by pointer to avoid copying the Videos slice (gocritic hugeParam).
func ToSectionWithVideos(m *service.SectionWithVideosModel) SectionWithVideosResponse {
	videos := make([]VideoResponse, 0, len(m.Videos))
	for i := range m.Videos {
		videos = append(videos, ToVideo(&m.Videos[i]))
	}
	return SectionWithVideosResponse{
		ID:        m.Section.ID,
		CourseID:  m.Section.CourseID,
		Titulo:    m.Section.Titulo,
		Orden:     m.Section.Orden,
		CreatedAt: m.Section.CreatedAt,
		Videos:    videos,
	}
}

// ToVideo converts a service.VideoModel to the VideoResponse wire shape.
// course-structure-v2: Descripcion and Materiales included.
// course-player-progress (Change 2): Completado included (caller-scoped, migration 0014).
func ToVideo(m *service.VideoModel) VideoResponse {
	materiales := make([]MaterialResponse, 0, len(m.Materiales))
	for i := range m.Materiales {
		materiales = append(materiales, ToMaterial(&m.Materiales[i]))
	}
	return VideoResponse{
		ID:          m.ID,
		SectionID:   m.SectionID,
		Titulo:      m.Titulo,
		Descripcion: m.Descripcion,
		URL:         m.URL,
		Proveedor:   m.Proveedor,
		DuracionS:   m.DuracionS,
		Orden:       m.Orden,
		CreatedAt:   m.CreatedAt,
		Materiales:  materiales,
		Completado:  m.Completado,
	}
}

// ── Material request bodies (C2.3) ────────────────────────────────────────────

// VideoProgressRequest is the body for PUT /api/videos/:id/progress.
// LastPositionS is forward-compat (D5): the manual flow leaves it 0/absent.
// Caller-scoped: the body MUST NOT contain a userId field; userID comes from JWT only (REQ-SEC).
type VideoProgressRequest struct {
	Completado    bool `json:"completado"`
	LastPositionS *int `json:"lastPositionS"`
}

// MaterialPresignRequest is the body for POST /api/videos/:id/materials/presign.
type MaterialPresignRequest struct {
	Nombre      string `json:"nombre"      binding:"required,min=1,max=255"`
	ContentType string `json:"contentType" binding:"required"`
	TamanoBytes int64  `json:"tamanoBytes" binding:"required,min=1"`
}

// MaterialConfirmRequest is the body for POST /api/videos/:id/materials.
type MaterialConfirmRequest struct {
	Key         string `json:"key"         binding:"required"`
	Nombre      string `json:"nombre"      binding:"required,min=1,max=255"`
	ContentType string `json:"contentType" binding:"required"`
	TamanoBytes int64  `json:"tamanoBytes" binding:"required,min=1"`
}

// ThumbnailConfirmRequest is the body for POST /api/courses/:courseId/thumbnail.
type ThumbnailConfirmRequest struct {
	Key string `json:"key" binding:"required"`
}

// ── Material response shapes (C2.3) ──────────────────────────────────────────

// PresignResponse is the wire shape for the presign upload response.
// Reused for both material and thumbnail presign (shapes are identical).
type PresignResponse struct {
	UploadURL string    `json:"uploadUrl"`
	Key       string    `json:"key"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// MaterialResponse is the wire shape for a material returned by the API.
// Nombre is mapped from MaterialModel.Titulo (D1: "nombre" is the wire label,
// "titulo" is the persisted field).
type MaterialResponse struct {
	ID          string    `json:"id"`
	Nombre      string    `json:"nombre"` // mapped from MaterialModel.Titulo
	MimeType    string    `json:"mimeType"`
	TamanoBytes int64     `json:"tamanoBytes"`
	CreatedAt   time.Time `json:"createdAt"`
}

// DownloadResponse is the wire shape for the presign download response.
type DownloadResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// ── Material mapping functions ─────────────────────────────────────────────────

// ToMaterial converts a service.MaterialModel to the MaterialResponse wire shape.
// nombre is the wire label; Titulo is the persisted field (D1).
func ToMaterial(m *service.MaterialModel) MaterialResponse {
	return MaterialResponse{
		ID:          m.ID,
		Nombre:      m.Titulo, // nombre is the wire label; Titulo is the persisted field (D1)
		MimeType:    m.MimeType,
		TamanoBytes: m.TamanoBytes,
		CreatedAt:   m.CreatedAt,
	}
}

// ── Categoria DTO ─────────────────────────────────────────────────────────────

// CategoriaResponse is the wire shape for a categoria.
// Verified unique across all module DTOs (collision guard T-5.1).
type CategoriaResponse struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Slug   string `json:"slug"`
}

// CategoriaCreateRequest is the body for POST /api/categorias (admin).
// Slug is auto-derived from nombre by the service — clients only send the name.
type CategoriaCreateRequest struct {
	Nombre string `json:"nombre" binding:"required,min=2,max=60"`
}

// CategoriaUpdateRequest is the body for PATCH /api/categorias/:id (admin).
type CategoriaUpdateRequest struct {
	Nombre string `json:"nombre" binding:"required,min=2,max=60"`
}

// ToCategoria converts a service.CategoriaModel to the CategoriaResponse wire shape.
func ToCategoria(m service.CategoriaModel) CategoriaResponse {
	return CategoriaResponse{ID: m.ID, Nombre: m.Nombre, Slug: m.Slug}
}

// ToCategorias converts a slice of service.CategoriaModel to a slice of CategoriaResponse.
func ToCategorias(models []service.CategoriaModel) []CategoriaResponse {
	out := make([]CategoriaResponse, 0, len(models))
	for _, m := range models {
		out = append(out, ToCategoria(m))
	}
	return out
}

// ── C2.4 catalog + enrollment DTOs (structural no-leak — design §6) ───────────

// CatalogCourseCard — one approved-course card (list item). NO content fields.
// Used in GET /catalog pagination envelope.
// course-structure-v2: metadata fields added.
type CatalogCourseCard struct {
	ID             string              `json:"id"`
	Titulo         string              `json:"titulo"`
	Descripcion    string              `json:"descripcion"`
	CreadorNombre  string              `json:"creadorNombre"`
	CreatedAt      time.Time           `json:"createdAt"`
	Nivel          *string             `json:"nivel"`
	MiniaturaURL   string              `json:"miniaturaUrl"`
	HorasPractico  float64             `json:"horasPractico"`
	CantidadClases int                 `json:"cantidadClases"`
	HorasVideo     float64             `json:"horasVideo"`
	Categorias     []CategoriaResponse `json:"categorias"`
}

// CoursePreviewResponse — non-enrolled detail. NO tree field AT THE STRUCT LEVEL (D6).
// The structural absence (not omitempty) is the compile-time guarantee for AC-9.
// course-structure-v2: metadata fields added; NO secciones/videos/materiales.
type CoursePreviewResponse struct {
	Enrolled       bool                `json:"enrolled"` // always false
	ID             string              `json:"id"`
	Titulo         string              `json:"titulo"`
	Descripcion    string              `json:"descripcion"`
	CreadorNombre  string              `json:"creadorNombre"`
	Nivel          *string             `json:"nivel"`
	MiniaturaURL   string              `json:"miniaturaUrl"`
	HorasPractico  float64             `json:"horasPractico"`
	CantidadClases int                 `json:"cantidadClases"`
	HorasVideo     float64             `json:"horasVideo"`
	Categorias     []CategoriaResponse `json:"categorias"`
}

// CourseDetailAlumnoResponse — enrolled detail WITH the full content tree.
// course-structure-v2: Materiales REMOVED from course level (now nested in VideoResponse).
// Metadata fields added.
type CourseDetailAlumnoResponse struct {
	Enrolled       bool                        `json:"enrolled"` // always true
	ID             string                      `json:"id"`
	Titulo         string                      `json:"titulo"`
	Descripcion    string                      `json:"descripcion"`
	CreadorNombre  string                      `json:"creadorNombre"`
	Secciones      []SectionWithVideosResponse `json:"secciones"`
	Nivel          *string                     `json:"nivel"`
	MiniaturaURL   string                      `json:"miniaturaUrl"`
	HorasPractico  float64                     `json:"horasPractico"`
	CantidadClases int                         `json:"cantidadClases"`
	HorasVideo     float64                     `json:"horasVideo"`
	Categorias     []CategoriaResponse         `json:"categorias"`
}

// EnrollmentResponse — POST /catalog/:id/enroll result.
type EnrollmentResponse struct {
	CourseID string `json:"courseId"`
	Enrolled bool   `json:"enrolled"` // always true on 200
}

// MyCourseItem — one row in GET /users/me/courses.
type MyCourseItem struct {
	CourseID      string    `json:"courseId"`
	Titulo        string    `json:"titulo"`
	CreadorNombre string    `json:"creadorNombre"`
	Completado    bool      `json:"completado"`
	InscritoEn    time.Time `json:"inscritoEn"`
}

// ── C2.4 catalog mapping functions ────────────────────────────────────────────

// ToCatalogCardPage maps a pagination.Page[service.CatalogCourseModel] to
// pagination.Page[CatalogCourseCard] for the wire envelope.
func ToCatalogCardPage(p pagination.Page[service.CatalogCourseModel]) pagination.Page[CatalogCourseCard] {
	items := make([]CatalogCourseCard, 0, len(p.Items))
	for i := range p.Items {
		items = append(items, CatalogCourseCard{
			ID:             p.Items[i].ID,
			Titulo:         p.Items[i].Titulo,
			Descripcion:    p.Items[i].Descripcion,
			CreadorNombre:  p.Items[i].CreadorNombre,
			CreatedAt:      p.Items[i].CreatedAt,
			Nivel:          p.Items[i].Nivel,
			MiniaturaURL:   p.Items[i].MiniaturaURL,
			HorasPractico:  p.Items[i].HorasPractico,
			CantidadClases: p.Items[i].CantidadClases,
			HorasVideo:     p.Items[i].HorasVideo,
			Categorias:     ToCategorias(p.Items[i].Categorias),
		})
	}
	return pagination.Page[CatalogCourseCard]{
		Items:      items,
		Page:       p.Page,
		Size:       p.Size,
		Total:      p.Total,
		TotalPages: p.TotalPages,
	}
}

// ToCoursePreview converts a non-enrolled CatalogDetailModel to the preview wire shape.
// The preview struct has NO tree fields — structural absence (not omitempty).
// course-structure-v2: metadata fields included; content fields (secciones/videos/materiales) absent.
func ToCoursePreview(d *service.CatalogDetailModel) CoursePreviewResponse {
	return CoursePreviewResponse{
		Enrolled:       false,
		ID:             d.ID,
		Titulo:         d.Titulo,
		Descripcion:    d.Descripcion,
		CreadorNombre:  d.CreadorNombre,
		Nivel:          d.Nivel,
		MiniaturaURL:   d.MiniaturaURL,
		HorasPractico:  d.HorasPractico,
		CantidadClases: d.CantidadClases,
		HorasVideo:     d.HorasVideo,
		Categorias:     ToCategorias(d.Categorias),
	}
}

// ToCourseDetailAlumno converts an enrolled CatalogDetailModel to the full wire shape.
// course-structure-v2: no course-level Materiales; per-video materials nested in VideoResponse.
func ToCourseDetailAlumno(d *service.CatalogDetailModel) CourseDetailAlumnoResponse {
	secciones := make([]SectionWithVideosResponse, 0, len(d.Sections))
	for i := range d.Sections {
		secciones = append(secciones, ToSectionWithVideos(&d.Sections[i]))
	}
	return CourseDetailAlumnoResponse{
		Enrolled:       true,
		ID:             d.ID,
		Titulo:         d.Titulo,
		Descripcion:    d.Descripcion,
		CreadorNombre:  d.CreadorNombre,
		Secciones:      secciones,
		Nivel:          d.Nivel,
		MiniaturaURL:   d.MiniaturaURL,
		HorasPractico:  d.HorasPractico,
		CantidadClases: d.CantidadClases,
		HorasVideo:     d.HorasVideo,
		Categorias:     ToCategorias(d.Categorias),
	}
}

// ToMyCourseItems converts a slice of service.MyCourseModel to the wire shape.
func ToMyCourseItems(rows []service.MyCourseModel) []MyCourseItem {
	items := make([]MyCourseItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, MyCourseItem{
			CourseID:      r.CourseID,
			Titulo:        r.Titulo,
			CreadorNombre: r.CreadorNombre,
			Completado:    r.Completado,
			InscritoEn:    r.InscritoEn,
		})
	}
	return items
}
