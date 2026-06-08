// Package dto defines the HTTP transport layer data-transfer objects for the
// certificates module. All DTOs use camelCase JSON tags and Swagger annotations.
// IMPORTANT: names must NOT collide with existing DTO types in other modules.
//   - DownloadResponse (courses) → use DownloadURLResponse here.
//   - SubmitResponse (evaluations) → not used here, but noted.
//
// Run: grep -rhoE "^type [A-Za-z]+ " backend/internal/modules/*/dto/*.go | sort | uniq -d
// must output nothing after adding this file.
package dto

import "time"

// CertificateListItem is the DTO for a single item in GET /certificates/me response.
//
// swagger:model CertificateListItem
type CertificateListItem struct {
	// ID is the certificate UUID.
	ID string `json:"id"`
	// CourseID is the UUID of the course for which this cert was issued.
	CourseID string `json:"courseId"`
	// CourseTitulo is the display title of the course (from courses seam).
	CourseTitulo string `json:"courseTitulo"`
	// Codigo is the unique verification code printed on the certificate.
	Codigo string `json:"codigo"`
	// EmitidoEn is the ISO-8601 timestamp when the certificate was issued.
	EmitidoEn time.Time `json:"emitidoEn"`
}

// CertificateResponse is the DTO for GET /certificates/:id response.
//
// swagger:model CertificateResponse
type CertificateResponse struct {
	// ID is the certificate UUID.
	ID string `json:"id"`
	// CourseID is the UUID of the course.
	CourseID string `json:"courseId"`
	// CourseTitulo is the display title of the course.
	CourseTitulo string `json:"courseTitulo"`
	// Codigo is the unique verification code.
	Codigo string `json:"codigo"`
	// EmitidoEn is the issue timestamp.
	EmitidoEn time.Time `json:"emitidoEn"`
}

// VerifyCertificateResponse is the PUBLIC DTO for GET /certificates/verify/:codigo.
// Exposes only non-sensitive fields needed to confirm authenticity to a third party
// (e.g. an employer). A 200 means the code is valid; 404 means it does not exist.
//
// swagger:model VerifyCertificateResponse
type VerifyCertificateResponse struct {
	// Codigo is the unique verification code that was looked up.
	Codigo string `json:"codigo"`
	// HolderNombre is the display name of the person the certificate was issued to.
	HolderNombre string `json:"holderNombre"`
	// CourseTitulo is the display title of the completed course.
	CourseTitulo string `json:"courseTitulo"`
	// EmitidoEn is the ISO-8601 timestamp when the certificate was issued.
	EmitidoEn time.Time `json:"emitidoEn"`
}

// DownloadURLResponse is the DTO for GET /certificates/:id/download response.
// NOTE: Named DownloadURLResponse (not DownloadResponse) to avoid collision with courses.DownloadResponse.
//
// swagger:model DownloadURLResponse
type DownloadURLResponse struct {
	// URL is the presigned GET URL for the certificate PDF.
	URL string `json:"url"`
	// ExpiresAt is the UTC timestamp when the presigned URL expires.
	ExpiresAt time.Time `json:"expiresAt"`
}

// BadgeResponse is the DTO for a single earned badge in GET /badges/me response.
//
// swagger:model BadgeResponse
type BadgeResponse struct {
	// ID is the badge UUID.
	ID string `json:"id"`
	// Nombre is the badge display name.
	Nombre string `json:"nombre"`
	// Descripcion is the badge description.
	Descripcion string `json:"descripcion"`
	// OtorgadoEn is the UTC timestamp when the badge was awarded.
	OtorgadoEn time.Time `json:"otorgadoEn"`
}

// RankingItem is the DTO for a single entry in GET /badges/ranking response.
//
// swagger:model RankingItem
type RankingItem struct {
	// Posicion is the 1-based rank position.
	Posicion int `json:"posicion"`
	// UserNombre is the display name of the ranked user.
	UserNombre string `json:"userNombre"`
	// CertCount is the total number of certificates the user has earned.
	CertCount int64 `json:"certCount"`
}

// ListCertificatesResponse wraps the certificate list.
//
// swagger:model ListCertificatesResponse
type ListCertificatesResponse struct {
	Certificates []CertificateListItem `json:"certificates"`
}

// ListBadgesResponse wraps the badge list.
//
// swagger:model ListBadgesResponse
type ListBadgesResponse struct {
	Badges []BadgeResponse `json:"badges"`
}

// RankingResponse wraps the ranking list.
//
// swagger:model RankingResponse
type RankingResponse struct {
	Ranking []RankingItem `json:"ranking"`
}
