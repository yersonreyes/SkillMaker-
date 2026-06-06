package service

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/repository"
)

// allowedMIME is the service-level MIME whitelist (ADR-3).
// Not config-driven — fixed business rule per proposal D3.
var allowedMIME = map[string]bool{
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/zip":              true,
	"application/x-zip-compressed": true,
	"image/jpeg":                   true,
	"image/png":                    true,
	"image/gif":                    true,
	"image/webp":                   true,
}

// allowedImageMIME is the MIME whitelist for thumbnail uploads.
var allowedImageMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

// sanitizeFilename strips path separators, replaces spaces with underscores, and removes any
// character that is not alphanumeric, dot, underscore, or hyphen. Returns "file" for empty results.
func sanitizeFilename(name string) string {
	base := filepath.Base(name) // strip path separators; Base("") returns "."
	// If filepath.Base returned only "." (empty input or pure dots) start fresh.
	if base == "." {
		base = ""
	}
	base = strings.ReplaceAll(base, " ", "_")
	base = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			return r
		default:
			return -1 // drop everything else
		}
	}, base)
	// Collapse any runs of underscores that may result from consecutive special chars.
	for strings.Contains(base, "__") {
		base = strings.ReplaceAll(base, "__", "_")
	}
	// Trim leading/trailing underscores.
	base = strings.Trim(base, "_")
	if base == "" || base == "." {
		base = "file"
	}
	return base
}

// materialKey generates a unique, collision-resistant storage key for a material.
// course-structure-v2: Format: courses/{courseID}/videos/{videoID}/materials/{uuid}-{sanitized-nombre}
// courseID is resolved from the video ownership chain.
func materialKey(courseID, videoID, nombre string) string {
	return fmt.Sprintf("courses/%s/videos/%s/materials/%s-%s",
		courseID, videoID, uuid.New().String(), sanitizeFilename(nombre))
}

// thumbnailKey generates a unique storage key for a course thumbnail.
// Format: courses/{courseID}/thumbnail/{uuid}-{sanitized-nombre}
func thumbnailKey(courseID, nombre string) string {
	return fmt.Sprintf("courses/%s/thumbnail/%s-%s",
		courseID, uuid.New().String(), sanitizeFilename(nombre))
}

// toMaterialModel converts a domain.Material to a MaterialModel.
// course-structure-v2: maps VideoID instead of CourseID.
func toMaterialModel(m *domain.Material) *MaterialModel {
	return &MaterialModel{
		ID:          m.ID,
		VideoID:     m.VideoID,
		Titulo:      m.Titulo,
		StorageKey:  m.StorageKey,
		MimeType:    m.MimeType,
		TamanoBytes: m.TamanoBytes,
		CreatedAt:   m.CreatedAt,
	}
}

// wrapMaterialNotFound converts repository.ErrMaterialNotFound to service.ErrMaterialNotFound.
func wrapMaterialNotFound(err error) error {
	if err == repository.ErrMaterialNotFound {
		return ErrMaterialNotFound
	}
	return err
}

// ── Material service methods (course-structure-v2: videoID-based) ──────────────

// PresignUpload generates a presigned PUT URL for uploading a material file.
// course-structure-v2: takes videoID instead of courseID.
// Resolves courseID+creadorID via ResolveVideoCourse, then asserts owner+editable.
func (s *serviceImpl) PresignUpload(ctx context.Context, videoID, callerID string, req PresignInput) (PresignResult, error) {
	// 1. Resolve video → course (owner + estado).
	courseID, ownerID, estado, err := s.repo.ResolveVideoCourse(ctx, videoID)
	if err != nil {
		if err == repository.ErrVideoNotFound {
			return PresignResult{}, ErrVideoNotFound
		}
		return PresignResult{}, err
	}

	// 2. Ownership check FIRST.
	if ownerID != callerID {
		return PresignResult{}, ErrNotOwner
	}

	// 3. Estado check (assertCourseEditable logic, inline for resolved values).
	if domain.Estado(estado) != domain.EstadoBorrador && domain.Estado(estado) != domain.EstadoRechazado {
		return PresignResult{}, ErrInvalidTransition
	}

	// 4. Validate file size.
	if req.TamanoBytes > s.maxUploadBytes {
		return PresignResult{}, ErrFileTooLarge
	}

	// 5. Validate MIME type.
	if !allowedMIME[req.ContentType] {
		return PresignResult{}, ErrMIMENotAllowed
	}

	// 6. Generate storage key (encodes courseID+videoID per design D5).
	key := materialKey(courseID, videoID, req.Nombre)

	// 7. Get presigned PUT URL from storage.
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

// ConfirmUpload persists a material row after a successful presigned PUT.
// course-structure-v2: takes videoID; resolves courseID for key-prefix validation.
func (s *serviceImpl) ConfirmUpload(ctx context.Context, videoID, callerID string, req ConfirmInput) (*MaterialModel, error) {
	// 1. Resolve video → course.
	courseID, ownerID, estado, err := s.repo.ResolveVideoCourse(ctx, videoID)
	if err != nil {
		if err == repository.ErrVideoNotFound {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}

	// 2. Ownership check FIRST.
	if ownerID != callerID {
		return nil, ErrNotOwner
	}

	// 3. Estado check.
	if domain.Estado(estado) != domain.EstadoBorrador && domain.Estado(estado) != domain.EstadoRechazado {
		return nil, ErrInvalidTransition
	}

	// 4. Validate key prefix — prevents injection of arbitrary storage keys.
	expectedPrefix := "courses/" + courseID + "/videos/" + videoID + "/materials/"
	if !strings.HasPrefix(req.Key, expectedPrefix) {
		return nil, ErrInvalidMaterialKey
	}

	// 5. Re-validate size (defense in depth — tampered confirm rejected even if presign passed).
	if req.TamanoBytes > s.maxUploadBytes {
		return nil, ErrFileTooLarge
	}

	// 6. Re-validate MIME (defense in depth).
	if !allowedMIME[req.ContentType] {
		return nil, ErrMIMENotAllowed
	}

	// 7. Persist the material row.
	m := &domain.Material{
		ID:          uuid.New().String(),
		VideoID:     videoID,    // course-structure-v2: VideoID (not CourseID)
		Titulo:      req.Nombre, // wire label "nombre" → persisted field "titulo" (D1)
		StorageKey:  req.Key,
		MimeType:    req.ContentType,
		TamanoBytes: req.TamanoBytes,
	}
	if err := s.repo.CreateMaterial(ctx, m); err != nil {
		return nil, err
	}

	return toMaterialModel(m), nil
}

// ListMaterialsByVideo returns all materials for a video ordered by created_at ASC.
// Owner-gated: resolves courseID via ResolveVideoCourse, checks ownership.
func (s *serviceImpl) ListMaterialsByVideo(ctx context.Context, videoID, callerID string) ([]MaterialModel, error) {
	// Resolve video → course for ownership check.
	_, ownerID, _, err := s.repo.ResolveVideoCourse(ctx, videoID)
	if err != nil {
		if err == repository.ErrVideoNotFound {
			return nil, ErrVideoNotFound
		}
		return nil, err
	}
	if ownerID != callerID {
		return nil, ErrNotOwner
	}

	materials, err := s.repo.ListMaterialsByVideo(ctx, videoID)
	if err != nil {
		return nil, err
	}

	result := make([]MaterialModel, 0, len(materials))
	for i := range materials {
		result = append(result, *toMaterialModel(&materials[i]))
	}
	return result, nil
}

// PresignDownload generates a presigned GET URL for downloading a material.
// course-structure-v2: uses GetMaterialOwnership for the chain check (no courseID arg).
//
// OQ3 resolution (orchestrator decision): a material is course CONTENT that a learner
// consumes. Therefore GET /materials/:id/download allows the caller if they are EITHER:
//   - the course owner (creador), OR
//   - an enrolled user of the material's course.
//
// Non-enrolled non-owner → ErrNotOwner (→ 403).
// The enrollment check reuses repo.IsEnrolled (courses module; no new cross-module dep).
// The course is resolved via the existing chain: material → video → section → course.
//
// GET /videos/:id/materials (LIST) remains OWNER-ONLY — it is the creator-editor endpoint;
// enrolled alumnos receive materials nested in the GET /catalog/:id tree, not via this endpoint.
func (s *serviceImpl) PresignDownload(ctx context.Context, materialID, callerID string) (DownloadResult, error) {
	// 1. Resolve ownership chain via GetMaterialOwnership.
	//    Returns courseID so we can check enrollment if callerID != ownerID.
	courseID, ownerID, _, err := s.repo.GetMaterialOwnership(ctx, materialID)
	if err != nil {
		return DownloadResult{}, wrapMaterialNotFound(err)
	}

	// 2. Authorization: owner OR enrolled.
	//    Check owner first (fast path — no extra DB call for the common creator case).
	if ownerID != callerID {
		// Not the owner — check enrollment.
		enrolled, err := s.repo.IsEnrolled(ctx, callerID, courseID)
		if err != nil {
			return DownloadResult{}, err
		}
		if !enrolled {
			return DownloadResult{}, ErrNotOwner
		}
	}

	// 3. Load the material to get StorageKey.
	m, err := s.repo.GetMaterialByID(ctx, materialID)
	if err != nil {
		return DownloadResult{}, wrapMaterialNotFound(err)
	}

	// 4. Generate presigned GET URL.
	downloadURL, err := s.store.PresignGetURL(ctx, m.StorageKey, s.presignTTL)
	if err != nil {
		return DownloadResult{}, err
	}

	return DownloadResult{
		URL:       downloadURL,
		ExpiresAt: time.Now().Add(s.presignTTL),
	}, nil
}

// DeleteMaterial deletes a material row and performs best-effort object deletion.
// course-structure-v2: uses GetMaterialOwnership for chain check.
// Per D5: if storage.Delete fails, the error is logged and swallowed — the request still succeeds.
func (s *serviceImpl) DeleteMaterial(ctx context.Context, materialID, callerID string) error {
	// 1. Resolve ownership chain.
	_, ownerID, estado, err := s.repo.GetMaterialOwnership(ctx, materialID)
	if err != nil {
		return wrapMaterialNotFound(err)
	}

	// 2. Ownership check FIRST.
	if ownerID != callerID {
		return ErrNotOwner
	}

	// 3. assertCourseEditable (estado check only — owner already verified above).
	if domain.Estado(estado) != domain.EstadoBorrador && domain.Estado(estado) != domain.EstadoRechazado {
		return ErrInvalidTransition
	}

	// 4. Load the material to get StorageKey.
	m, err := s.repo.GetMaterialByID(ctx, materialID)
	if err != nil {
		return wrapMaterialNotFound(err)
	}

	// 5. Delete the DB row FIRST (so a subsequent confirm cannot re-create a dangling row).
	if err := s.repo.DeleteMaterial(ctx, materialID); err != nil {
		return err
	}

	// 6. Best-effort object deletion — log on error, never fail the request (D5).
	if err := s.store.Delete(ctx, m.StorageKey); err != nil {
		slog.Warn("material object delete failed — orphaned object in storage",
			"key", m.StorageKey,
			"materialID", materialID,
			"err", err,
		)
	}

	return nil
}

// ── ListMaterials is removed — replaced by ListMaterialsByVideo ───────────────
// The old courseID-based method is replaced. Any remaining references in the
// handler must use videoID instead. Handler tests updated accordingly.
