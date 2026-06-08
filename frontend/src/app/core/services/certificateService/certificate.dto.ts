/**
 * certificate.dto.ts — DTOs for the certificates API (C5.1).
 *
 * These mirror the Go backend DTOs exactly (CertificateListItem, CertificateResponse,
 * DownloadURLResponse). Fields are optional to match the generated @api types.ts shape.
 */

/** One certificate entry from GET /api/certificates/me. */
export interface CertificateListItem {
  id?: string;
  courseId?: string;
  courseTitulo?: string;
  codigo?: string;
  emitidoEn?: string; // ISO 8601
}

/** Full certificate detail from GET /api/certificates/:id. */
export interface CertificateResponse {
  id?: string;
  courseId?: string;
  courseTitulo?: string;
  codigo?: string;
  emitidoEn?: string; // ISO 8601
}

/** Response from GET /api/certificates/:id/download. */
export interface DownloadURLResponse {
  url?: string;
  expiresAt?: string; // ISO 8601
}

/**
 * Public verification result from GET /api/certificates/verify/:codigo.
 * Only non-sensitive fields (holder name, course title, code, issue date).
 * A successful resolution means the code is valid; a 404 means it does not exist.
 */
export interface VerifyCertificateResponse {
  codigo?: string;
  holderNombre?: string;
  courseTitulo?: string;
  emitidoEn?: string; // ISO 8601
}
