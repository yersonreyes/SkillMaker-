// Package handler contains the Gin HTTP handlers for the courses module.
// Handlers are intentionally thin: parse → call service → errors.Is → render.
// No domain logic lives here.
//
// CRITICAL: ErrNotOwner maps to DIFFERENT HTTP statuses depending on the route:
//   - GET  /api/courses/:id  → 404 via renderCourseErrorRead  (hides existence)
//   - PATCH /api/courses/:id → 403 via renderCourseErrorWrite (signals authz failure)
//   - POST/PATCH/DELETE sections/videos → 403 via renderCourseErrorWrite
//
// This asymmetry is enforced by TWO separate render helpers (not a flag or switch)
// and is verified by TWO separate tests in handler_test.go. See REQ-DIVERGENCE.
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/httperr"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// Handler holds the service dependency injected at registration time.
type Handler struct {
	svc service.Service
}

// Register mounts the courses routes onto a pre-built Gin route group.
// The group must already carry JWT + RequireRole("creador") middleware.
func Register(creatorGrp *gin.RouterGroup, svc service.Service) {
	h := &Handler{svc: svc}

	// Course CRUD (C2.1).
	creatorGrp.POST("/courses", h.Create)
	creatorGrp.GET("/courses/:id", h.GetByID)
	creatorGrp.PATCH("/courses/:id", h.Update)
	creatorGrp.GET("/courses", h.List)

	// Section routes (C2.2).
	creatorGrp.POST("/courses/:courseId/sections", h.CreateSection)
	creatorGrp.GET("/courses/:id/sections", h.ListContent) // CRITICAL: read path for content tree
	creatorGrp.PATCH("/courses/:id/sections/reorder", h.ReorderSections)
	creatorGrp.PATCH("/sections/:id", h.UpdateSection)
	creatorGrp.DELETE("/sections/:id", h.DeleteSection)

	// Video routes (C2.2).
	creatorGrp.POST("/sections/:sectionId/videos", h.CreateVideo)
	creatorGrp.PATCH("/videos/:id", h.UpdateVideo)
	creatorGrp.DELETE("/videos/:id", h.DeleteVideo)

	// Material routes (C2.3).
	// POST tree: uses :courseId (matches existing POST /courses/:courseId/sections).
	// GET tree: uses :id (matches existing GET /courses/:id/sections).
	// DELETE: flat /materials/:id (no conflict).
	creatorGrp.POST("/courses/:courseId/materials/presign", h.PresignMaterial)
	creatorGrp.POST("/courses/:courseId/materials", h.ConfirmMaterial)
	creatorGrp.GET("/courses/:id/materials", h.ListMaterials)
	creatorGrp.GET("/courses/:id/materials/:materialId/download", h.DownloadMaterial)
	creatorGrp.DELETE("/materials/:id", h.DeleteMaterial)
}

// ── Course handlers ────────────────────────────────────────────────────────────

// Create godoc
// @Summary     Crea un curso (borrador)
// @Description Crea un nuevo curso con estado=borrador. El creadorID viene del JWT.
// @Tags        courses
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body   dto.CreateCourseRequest true "Datos del curso"
// @Success     201 {object} dto.CourseDetail
// @Failure     400 {object} httperr.Error "titulo faltante o invalido"
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error "rol creador requerido"
// @Failure     500 {object} httperr.Error
// @Router      /courses [post]
func (h *Handler) Create(c *gin.Context) {
	var req dto.CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	creadorID := middleware.UserIDFrom(c)
	if creadorID == "" {
		httperr.Render(c, httperr.Unauthorized("MISSING_IDENTITY", "could not resolve user identity from token"))
		return
	}

	model, err := h.svc.Create(c.Request.Context(), creadorID, service.CreateRequest{
		Titulo:      req.Titulo,
		Descripcion: req.Descripcion,
	})
	if err != nil {
		slog.Error("courses.create: unexpected error", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, dto.ToCourseDetail(model, false))
}

// GetByID godoc
// @Summary     Detalle de curso por ID (solo owner)
// @Description Retorna el detalle de un curso. Solo el creador propietario puede verlo.
//
//	ErrNotOwner → 404 (oculta existencia de borradores ajenos).
//	hasContent se computa via svc.HasContent.
//
// @Tags        courses
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID del curso"
// @Success     200 {object} dto.CourseDetail
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error "rol creador requerido"
// @Failure     404 {object} httperr.Error "curso no encontrado o no pertenece al caller"
// @Failure     500 {object} httperr.Error
// @Router      /courses/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	model, err := h.svc.GetByID(c.Request.Context(), id, creadorID)
	if err != nil {
		h.renderCourseErrorRead(c, err)
		return
	}

	// Compute hasContent for the GET detail response (spec HC-1).
	hasContent, err := h.svc.HasContent(c.Request.Context(), id, creadorID)
	if err != nil {
		// Non-fatal: default to false and log.
		slog.Warn("courses.getbyid: hasContent failed", "courseID", id, "err", err)
		hasContent = false
	}

	c.JSON(http.StatusOK, dto.ToCourseDetail(model, hasContent))
}

// Update godoc
// @Summary     Actualiza un curso (PATCH parcial, solo owner)
// @Description Actualiza titulo y/o descripcion de un curso. Solo borrador o rechazado.
//
//	ErrNotOwner → 403 (senyala fallo de autorizacion en escritura).
//
// @Tags        courses
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path   string                  true "UUID del curso"
// @Param       body body   dto.UpdateCourseRequest true "Campos a actualizar (parcial)"
// @Success     200 {object} dto.CourseDetail
// @Failure     400 {object} httperr.Error "body invalido o sin campos"
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error "no es propietario del curso"
// @Failure     404 {object} httperr.Error "curso no encontrado"
// @Failure     409 {object} httperr.Error "estado no permite edicion"
// @Failure     500 {object} httperr.Error
// @Router      /courses/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	// Reject requests with no actionable fields.
	if req.Titulo == nil && req.Descripcion == nil {
		httperr.Render(c, httperr.BadRequest("NO_FIELDS", "no fields to update"))
		return
	}

	creadorID := middleware.UserIDFrom(c)

	model, err := h.svc.UpdateByID(c.Request.Context(), id, creadorID, service.UpdateRequest{
		Titulo:      req.Titulo,
		Descripcion: req.Descripcion,
	})
	if err != nil {
		h.renderCourseErrorWrite(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToCourseDetail(model, false))
}

// List godoc
// @Summary     Lista mis cursos (paginado)
// @Description Retorna una pagina de cursos del caller. ?creator=me es obligatorio.
// @Tags        courses
// @Produce     json
// @Security    BearerAuth
// @Param       creator query string true  "Debe ser 'me' (solo cursos propios)"
// @Param       page    query int    false "Pagina (default 1)"
// @Param       size    query int    false "Tamano de pagina (max 100, default 20)"
// @Success     200 {object} object "pagina de cursos (items, page, size, total, totalPages)"
// @Failure     400 {object} httperr.Error "creator debe ser 'me'"
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error "rol creador requerido"
// @Failure     500 {object} httperr.Error
// @Router      /courses [get]
func (h *Handler) List(c *gin.Context) {
	creator := c.Query("creator")
	if creator != "me" {
		httperr.Render(c, httperr.BadRequest("INVALID_PARAM", "creator must be 'me'"))
		return
	}

	creadorID := middleware.UserIDFrom(c)
	p := pagination.ParseParams(c)

	page, err := h.svc.ListByCreator(c.Request.Context(), creadorID, p)
	if err != nil {
		slog.Error("courses.list: unexpected error", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.ToCourseListPage(page))
}

// ── Section handlers (C2.2) ───────────────────────────────────────────────────

// CreateSection godoc
// @Summary     Crea una seccion en un curso
// @Description Crea una seccion. El curso debe estar en borrador o rechazado y pertenecer al caller.
// @Tags        sections
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       courseId path   string                    true "UUID del curso"
// @Param       body     body   dto.SectionCreateRequest  true "Datos de la seccion"
// @Success     201 {object} dto.SectionResponse
// @Failure     400 {object} httperr.Error
// @Failure     403 {object} httperr.Error "no es propietario del curso"
// @Failure     404 {object} httperr.Error "curso no encontrado"
// @Failure     409 {object} httperr.Error "estado no permite edicion"
// @Router      /courses/{courseId}/sections [post]
func (h *Handler) CreateSection(c *gin.Context) {
	courseID := c.Param("courseId")
	creadorID := middleware.UserIDFrom(c)

	var req dto.SectionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	model, err := h.svc.CreateSection(c.Request.Context(), creadorID, service.SectionCreateRequest{
		CourseID: courseID,
		Titulo:   req.Titulo,
	})
	if err != nil {
		h.renderCourseErrorWrite(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToSection(model))
}

// ListContent godoc
// @Summary     Lista las secciones de un curso con sus videos anidados
// @Description Retorna el arbol de contenido del curso: secciones ordenadas por orden,
//
//	cada una con sus videos ordenados por orden. Solo el propietario del curso puede verlo.
//	ErrNotOwner → 404 (oculta existencia, consistente con GET /courses/:id).
//
// @Tags        sections
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID del curso"
// @Success     200 {array}  dto.SectionWithVideosResponse
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error "rol creador requerido"
// @Failure     404 {object} httperr.Error "curso no encontrado o no pertenece al caller"
// @Failure     500 {object} httperr.Error
// @Router      /courses/{id}/sections [get]
func (h *Handler) ListContent(c *gin.Context) {
	courseID := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	content, err := h.svc.ListContent(c.Request.Context(), courseID, creadorID)
	if err != nil {
		h.renderCourseErrorRead(c, err)
		return
	}

	resp := make([]dto.SectionWithVideosResponse, 0, len(content))
	for i := range content {
		resp = append(resp, dto.ToSectionWithVideos(&content[i]))
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateSection godoc
// @Summary     Actualiza una seccion (PATCH parcial)
// @Description Actualiza titulo de una seccion. Requiere ser propietario del curso.
// @Tags        sections
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path   string                    true "UUID de la seccion"
// @Param       body body   dto.SectionUpdateRequest  true "Campos a actualizar"
// @Success     200 {object} dto.SectionResponse
// @Failure     400 {object} httperr.Error
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error "seccion no encontrada"
// @Failure     409 {object} httperr.Error
// @Router      /sections/{id} [patch]
func (h *Handler) UpdateSection(c *gin.Context) {
	id := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	var req dto.SectionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	model, err := h.svc.UpdateSection(c.Request.Context(), id, creadorID, service.SectionUpdateRequest{
		Titulo: req.Titulo,
	})
	if err != nil {
		h.renderCourseErrorWrite(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToSection(model))
}

// DeleteSection godoc
// @Summary     Elimina una seccion (y sus videos en cascada)
// @Description Elimina la seccion y todos sus videos. Requiere ser propietario.
// @Tags        sections
// @Security    BearerAuth
// @Param       id path string true "UUID de la seccion"
// @Success     204
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Failure     409 {object} httperr.Error
// @Router      /sections/{id} [delete]
func (h *Handler) DeleteSection(c *gin.Context) {
	id := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	if err := h.svc.DeleteSection(c.Request.Context(), id, creadorID); err != nil {
		h.renderCourseErrorWrite(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ReorderSections godoc
// @Summary     Reordena las secciones de un curso
// @Description Actualiza el orden de las secciones. ids debe ser el conjunto exacto de secciones del curso.
// @Tags        sections
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path   string             true "UUID del curso"
// @Param       body body   dto.ReorderRequest true "IDs en el nuevo orden"
// @Success     200
// @Failure     400 {object} httperr.Error "IDs invalidos o incompletos"
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Router      /courses/{id}/sections/reorder [patch]
func (h *Handler) ReorderSections(c *gin.Context) {
	courseID := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	var req dto.ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	if err := h.svc.ReorderSections(c.Request.Context(), courseID, creadorID, req.IDs); err != nil {
		h.renderCourseErrorWrite(c, err)
		return
	}

	c.Status(http.StatusOK)
}

// ── Video handlers (C2.2) ─────────────────────────────────────────────────────

// CreateVideo godoc
// @Summary     Crea un video en una seccion
// @Description Crea un video. La seccion y el curso deben pertenecer al caller. URL y proveedor se validan cruzadamente.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       sectionId path   string                  true "UUID de la seccion"
// @Param       body      body   dto.VideoCreateRequest  true "Datos del video"
// @Success     201 {object} dto.VideoResponse
// @Failure     400 {object} httperr.Error "url/proveedor invalidos"
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Failure     409 {object} httperr.Error
// @Router      /sections/{sectionId}/videos [post]
func (h *Handler) CreateVideo(c *gin.Context) {
	sectionID := c.Param("sectionId")
	creadorID := middleware.UserIDFrom(c)

	var req dto.VideoCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	model, err := h.svc.CreateVideo(c.Request.Context(), creadorID, service.VideoCreateRequest{
		SectionID: sectionID,
		Titulo:    req.Titulo,
		URL:       req.URL,
		Proveedor: req.Proveedor,
		DuracionS: req.DuracionS,
	})
	if err != nil {
		h.renderCourseErrorWrite(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToVideo(model))
}

// UpdateVideo godoc
// @Summary     Actualiza un video (PATCH parcial)
// @Description Actualiza campos de un video. Si url o proveedor cambia, se re-validan cruzadamente.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path   string                  true "UUID del video"
// @Param       body body   dto.VideoUpdateRequest  true "Campos a actualizar"
// @Success     200 {object} dto.VideoResponse
// @Failure     400 {object} httperr.Error
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Failure     409 {object} httperr.Error
// @Router      /videos/{id} [patch]
func (h *Handler) UpdateVideo(c *gin.Context) {
	id := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	var req dto.VideoUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	model, err := h.svc.UpdateVideo(c.Request.Context(), id, creadorID, service.VideoUpdateRequest{
		Titulo:    req.Titulo,
		URL:       req.URL,
		Proveedor: req.Proveedor,
		DuracionS: req.DuracionS,
	})
	if err != nil {
		h.renderCourseErrorWrite(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToVideo(model))
}

// DeleteVideo godoc
// @Summary     Elimina un video
// @Description Elimina un video. Requiere ser propietario del curso.
// @Tags        videos
// @Security    BearerAuth
// @Param       id path string true "UUID del video"
// @Success     204
// @Failure     403 {object} httperr.Error
// @Failure     404 {object} httperr.Error
// @Router      /videos/{id} [delete]
func (h *Handler) DeleteVideo(c *gin.Context) {
	id := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	if err := h.svc.DeleteVideo(c.Request.Context(), id, creadorID); err != nil {
		h.renderCourseErrorWrite(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ── Material handlers (C2.3) ──────────────────────────────────────────────────

// PresignMaterial godoc
// @Summary     Genera URL presignada para subir un archivo
// @Description Genera una URL PUT presignada para que el browser suba el archivo directamente a MinIO.
//
//	ErrFileTooLarge → 413, ErrMIMENotAllowed → 415.
//
// @Tags        materials
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       courseId path   string                         true "UUID del curso"
// @Param       body     body   dto.MaterialPresignRequest     true "Datos del archivo a subir"
// @Success     200 {object} dto.PresignResponse
// @Failure     400 {object} httperr.Error "body invalido o campos faltantes"
// @Failure     403 {object} httperr.Error "no es propietario del curso"
// @Failure     404 {object} httperr.Error "curso no encontrado"
// @Failure     409 {object} httperr.Error "estado no permite edicion"
// @Failure     413 {object} httperr.Error "archivo demasiado grande"
// @Failure     415 {object} httperr.Error "tipo de contenido no permitido"
// @Router      /courses/{courseId}/materials/presign [post]
func (h *Handler) PresignMaterial(c *gin.Context) {
	courseID := c.Param("courseId")
	creadorID := middleware.UserIDFrom(c)

	var req dto.MaterialPresignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	result, err := h.svc.PresignUpload(c.Request.Context(), courseID, creadorID, service.PresignInput{
		Nombre:      req.Nombre,
		ContentType: req.ContentType,
		TamanoBytes: req.TamanoBytes,
	})
	if err != nil {
		h.renderCourseErrorWrite(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.PresignResponse{
		UploadURL: result.UploadURL,
		Key:       result.Key,
		ExpiresAt: result.ExpiresAt,
	})
}

// ConfirmMaterial godoc
// @Summary     Confirma la subida de un material (crea la fila en DB)
// @Description Persiste el material tras un PUT presignado exitoso. Re-valida tamano y MIME.
//
//	ErrInvalidMaterialKey → 400, ErrFileTooLarge → 413, ErrMIMENotAllowed → 415.
//
// @Tags        materials
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       courseId path   string                         true "UUID del curso"
// @Param       body     body   dto.MaterialConfirmRequest     true "Datos del material confirmado"
// @Success     201 {object} dto.MaterialResponse
// @Failure     400 {object} httperr.Error "clave de objeto invalida o campos faltantes"
// @Failure     403 {object} httperr.Error "no es propietario del curso"
// @Failure     404 {object} httperr.Error "curso no encontrado"
// @Failure     409 {object} httperr.Error "estado no permite edicion"
// @Failure     413 {object} httperr.Error "archivo demasiado grande"
// @Failure     415 {object} httperr.Error "tipo de contenido no permitido"
// @Router      /courses/{courseId}/materials [post]
func (h *Handler) ConfirmMaterial(c *gin.Context) {
	courseID := c.Param("courseId")
	creadorID := middleware.UserIDFrom(c)

	var req dto.MaterialConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.Render(c, httperr.BadRequest("INVALID_BODY", "body invalido: "+err.Error()))
		return
	}

	model, err := h.svc.ConfirmUpload(c.Request.Context(), courseID, creadorID, service.ConfirmInput{
		Key:         req.Key,
		Nombre:      req.Nombre,
		ContentType: req.ContentType,
		TamanoBytes: req.TamanoBytes,
	})
	if err != nil {
		h.renderCourseErrorWrite(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToMaterial(model))
}

// ListMaterials godoc
// @Summary     Lista los materiales de un curso
// @Description Retorna todos los materiales del curso ordenados por fecha de creacion ASC. Solo el propietario.
// @Tags        materials
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "UUID del curso"
// @Success     200 {array}  dto.MaterialResponse
// @Failure     401 {object} httperr.Error
// @Failure     403 {object} httperr.Error "rol creador requerido"
// @Failure     404 {object} httperr.Error "curso no encontrado o no pertenece al caller"
// @Failure     500 {object} httperr.Error
// @Router      /courses/{id}/materials [get]
func (h *Handler) ListMaterials(c *gin.Context) {
	courseID := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	materials, err := h.svc.ListMaterials(c.Request.Context(), courseID, creadorID)
	if err != nil {
		h.renderCourseErrorRead(c, err)
		return
	}

	resp := make([]dto.MaterialResponse, 0, len(materials))
	for i := range materials {
		resp = append(resp, dto.ToMaterial(&materials[i]))
	}
	c.JSON(http.StatusOK, resp)
}

// DownloadMaterial godoc
// @Summary     Genera URL presignada para descargar un material
// @Description Retorna una URL GET presignada con TTL = PresignTTL. Solo el propietario del curso.
// @Tags        materials
// @Produce     json
// @Security    BearerAuth
// @Param       id         path string true "UUID del curso"
// @Param       materialId path string true "UUID del material"
// @Success     200 {object} dto.DownloadResponse
// @Failure     401 {object} httperr.Error
// @Failure     404 {object} httperr.Error "material o curso no encontrado"
// @Failure     500 {object} httperr.Error
// @Router      /courses/{id}/materials/{materialId}/download [get]
func (h *Handler) DownloadMaterial(c *gin.Context) {
	courseID := c.Param("id")
	materialID := c.Param("materialId")
	creadorID := middleware.UserIDFrom(c)

	result, err := h.svc.PresignDownload(c.Request.Context(), courseID, materialID, creadorID)
	if err != nil {
		h.renderCourseErrorRead(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.DownloadResponse{
		URL:       result.URL,
		ExpiresAt: result.ExpiresAt,
	})
}

// DeleteMaterial godoc
// @Summary     Elimina un material
// @Description Elimina la fila de material y hace un best-effort delete del objeto en storage.
//
//	Si el objeto ya no existe, la peticion igual retorna 204 (D5).
//
// @Tags        materials
// @Security    BearerAuth
// @Param       id path string true "UUID del material"
// @Success     204
// @Failure     403 {object} httperr.Error "no es propietario del curso"
// @Failure     404 {object} httperr.Error "material no encontrado"
// @Router      /materials/{id} [delete]
func (h *Handler) DeleteMaterial(c *gin.Context) {
	materialID := c.Param("id")
	creadorID := middleware.UserIDFrom(c)

	if err := h.svc.DeleteMaterial(c.Request.Context(), materialID, creadorID); err != nil {
		h.renderCourseErrorWrite(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ── Error render helpers (TWO — deliberate, enforces REQ-DIVERGENCE) ───────────

// renderCourseErrorRead maps service sentinels to HTTP statuses for READ routes (GET).
// CRITICAL: ErrNotOwner → 404 here (hides existence of private drafts from other creadores).
func (h *Handler) renderCourseErrorRead(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCourseNotFound):
		httperr.Render(c, httperr.NotFound("COURSE_NOT_FOUND", "course not found"))
	case errors.Is(err, service.ErrSectionNotFound):
		httperr.Render(c, httperr.NotFound("SECTION_NOT_FOUND", "section not found"))
	case errors.Is(err, service.ErrVideoNotFound):
		httperr.Render(c, httperr.NotFound("VIDEO_NOT_FOUND", "video not found"))
	case errors.Is(err, service.ErrMaterialNotFound):
		httperr.Render(c, httperr.NotFound("MATERIAL_NOT_FOUND", "material not found"))
	case errors.Is(err, service.ErrNotOwner):
		httperr.Render(c, httperr.NotFound("COURSE_NOT_FOUND", "course not found")) // 404: hide existence
	case errors.Is(err, service.ErrInvalidTransition):
		httperr.Render(c, httperr.Conflict("INVALID_TRANSITION", "course estado does not permit this edit"))
	case errors.Is(err, service.ErrInvalidReorderSet):
		httperr.Render(c, httperr.BadRequest("INVALID_REORDER_SET", "reorder ids must exactly match the course's section set"))
	default:
		slog.Error("courses: unexpected error (read)", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
	}
}

// renderCourseErrorWrite maps service sentinels to HTTP statuses for WRITE routes (PATCH/POST/DELETE).
// CRITICAL: ErrNotOwner → 403 here (signals authz failure per AC3 / REQ-DIVERGENCE).
// The ONLY difference from renderCourseErrorRead is the ErrNotOwner case.
// ErrInvalidReorderSet → 400 (validation error: wrong section IDs sent).
// ErrInvalidTransition → 409 (state error: course not editable).
// ErrFileTooLarge → 413, ErrMIMENotAllowed → 415 (material upload validation).
// These are DISTINCT: one is about the input set, the other about course estado.
func (h *Handler) renderCourseErrorWrite(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCourseNotFound):
		httperr.Render(c, httperr.NotFound("COURSE_NOT_FOUND", "course not found"))
	case errors.Is(err, service.ErrSectionNotFound):
		httperr.Render(c, httperr.NotFound("SECTION_NOT_FOUND", "section not found"))
	case errors.Is(err, service.ErrVideoNotFound):
		httperr.Render(c, httperr.NotFound("VIDEO_NOT_FOUND", "video not found"))
	case errors.Is(err, service.ErrMaterialNotFound):
		httperr.Render(c, httperr.NotFound("MATERIAL_NOT_FOUND", "material not found"))
	case errors.Is(err, service.ErrNotOwner):
		httperr.Render(c, httperr.Forbidden("NOT_OWNER", "you do not own this course")) // 403: authz signal
	case errors.Is(err, service.ErrInvalidReorderSet):
		httperr.Render(c, httperr.BadRequest("INVALID_REORDER_SET", "reorder ids must exactly match the course's section set"))
	case errors.Is(err, service.ErrInvalidTransition):
		httperr.Render(c, httperr.Conflict("INVALID_TRANSITION", "course estado does not permit this edit"))
	case errors.Is(err, service.ErrURLProviderMismatch):
		httperr.Render(c, httperr.BadRequest("URL_PROVIDER_MISMATCH", "url host does not match declared proveedor"))
	case errors.Is(err, service.ErrFileTooLarge):
		httperr.Render(c, httperr.PayloadTooLarge("FILE_TOO_LARGE", "file exceeds max upload size"))
	case errors.Is(err, service.ErrMIMENotAllowed):
		httperr.Render(c, httperr.UnsupportedMediaType("MIME_NOT_ALLOWED", "content type not allowed"))
	case errors.Is(err, service.ErrInvalidMaterialKey):
		httperr.Render(c, httperr.BadRequest("INVALID_KEY", "material key prefix mismatch"))
	default:
		slog.Error("courses: unexpected error (write)", "err", err)
		httperr.Render(c, httperr.Internal(err.Error()))
	}
}
