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
// Format: courses/{courseID}/materials/{uuid}-{sanitized-nombre}
func materialKey(courseID, nombre string) string {
	return fmt.Sprintf("courses/%s/materials/%s-%s", courseID, uuid.New().String(), sanitizeFilename(nombre))
}

// toMaterialModel converts a domain.Material to a MaterialModel.
func toMaterialModel(m *domain.Material) *MaterialModel {
	return &MaterialModel{
		ID:          m.ID,
		CourseID:    m.CourseID,
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

// canAccessMaterial checks whether creadorID is the owner of the course that
// owns the given material. This predicate is isolated so C2.4 can broaden it
// to owner-OR-enrolled by editing ONLY this method.
func (s *serviceImpl) canAccessMaterial(ctx context.Context, m *domain.Material, creadorID string) error {
	c, err := s.repo.GetByID(ctx, m.CourseID)
	if err != nil {
		return wrapNotFound(err)
	}
	if c.CreadorID != creadorID {
		return ErrNotOwner
	}
	return nil
}

// ── Material service methods ───────────────────────────────────────────────────

// PresignUpload generates a presigned PUT URL for uploading a material file.
// Validates ownership+estado, size, and MIME before calling storage.
func (s *serviceImpl) PresignUpload(ctx context.Context, courseID, creadorID string, req PresignInput) (PresignResult, error) {
	// 1. Assert course is editable (owner FIRST, then estado).
	if _, err := s.assertCourseEditable(ctx, courseID, creadorID); err != nil {
		return PresignResult{}, err
	}

	// 2. Validate file size.
	if req.TamanoBytes > s.maxUploadBytes {
		return PresignResult{}, ErrFileTooLarge
	}

	// 3. Validate MIME type.
	if !allowedMIME[req.ContentType] {
		return PresignResult{}, ErrMIMENotAllowed
	}

	// 4. Generate storage key.
	key := materialKey(courseID, req.Nombre)

	// 5. Get presigned PUT URL from storage.
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
// Re-validates size+MIME as defense in depth (dual validation per design §3, ADR-3).
func (s *serviceImpl) ConfirmUpload(ctx context.Context, courseID, creadorID string, req ConfirmInput) (*MaterialModel, error) {
	// 1. Assert course is editable.
	if _, err := s.assertCourseEditable(ctx, courseID, creadorID); err != nil {
		return nil, err
	}

	// 2. Validate key prefix — prevents injection of arbitrary storage keys.
	expectedPrefix := "courses/" + courseID + "/materials/"
	if !strings.HasPrefix(req.Key, expectedPrefix) {
		return nil, ErrInvalidMaterialKey
	}

	// 3. Re-validate size (defense in depth — tampered confirm rejected even if presign passed).
	if req.TamanoBytes > s.maxUploadBytes {
		return nil, ErrFileTooLarge
	}

	// 4. Re-validate MIME (defense in depth).
	if !allowedMIME[req.ContentType] {
		return nil, ErrMIMENotAllowed
	}

	// 5. Persist the material row.
	m := &domain.Material{
		ID:          uuid.New().String(),
		CourseID:    courseID,
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

// ListMaterials returns all materials for a course ordered by created_at ASC.
// Owner-gated (read semantics — non-owner returns ErrNotOwner → handler renders as 404).
func (s *serviceImpl) ListMaterials(ctx context.Context, courseID, creadorID string) ([]MaterialModel, error) {
	// Ownership check (read semantics).
	c, err := s.repo.GetByID(ctx, courseID)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	if c.CreadorID != creadorID {
		return nil, ErrNotOwner
	}

	materials, err := s.repo.ListMaterialsByCourse(ctx, courseID)
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
// Owner-gated via canAccessMaterial (read semantics — non-owner → ErrNotOwner → 404).
// courseID is part of the interface contract (C2.4 may use it for enrollment checks).
func (s *serviceImpl) PresignDownload(ctx context.Context, _ /* courseID */, materialID, creadorID string) (DownloadResult, error) {
	// 1. Load the material.
	m, err := s.repo.GetMaterialByID(ctx, materialID)
	if err != nil {
		return DownloadResult{}, wrapMaterialNotFound(err)
	}

	// 2. Ownership check via isolated predicate (C2.4 broadens this method only).
	if err := s.canAccessMaterial(ctx, m, creadorID); err != nil {
		return DownloadResult{}, err
	}

	// 3. Generate presigned GET URL.
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
// Owner-gated (write semantics — non-owner → ErrNotOwner → 403).
// Per D5: if storage.Delete fails, the error is logged and swallowed — the request still succeeds.
func (s *serviceImpl) DeleteMaterial(ctx context.Context, materialID, creadorID string) error {
	// 1. Load the material.
	m, err := s.repo.GetMaterialByID(ctx, materialID)
	if err != nil {
		return wrapMaterialNotFound(err)
	}

	// 2. Load course and assert ownership (write semantics → ErrNotOwner → 403).
	c, err := s.repo.GetByID(ctx, m.CourseID)
	if err != nil {
		return wrapNotFound(err)
	}
	if c.CreadorID != creadorID {
		return ErrNotOwner
	}

	// 3. Delete the DB row FIRST (so a subsequent confirm cannot re-create a dangling row).
	if err := s.repo.DeleteMaterial(ctx, materialID); err != nil {
		return err
	}

	// 4. Best-effort object deletion — log on error, never fail the request (D5).
	if err := s.store.Delete(ctx, m.StorageKey); err != nil {
		slog.Warn("material object delete failed — orphaned object in storage",
			"key", m.StorageKey,
			"materialID", materialID,
			"err", err,
		)
	}

	return nil
}
