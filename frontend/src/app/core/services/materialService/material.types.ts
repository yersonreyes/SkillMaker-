/**
 * material.types.ts — DTOs for the Material Attachments API.
 * These mirror the backend camelCase JSON contract exactly.
 * Re-keyed in course-structure-v2: materials now belong to videos, not courses.
 */

/** POST /api/videos/:videoId/materials/presign — request body. */
export interface MaterialPresignRequest {
  nombre: string;
  contentType: string;
  tamanoBytes: number;
}

/** POST /api/videos/:videoId/materials/presign — response body. */
export interface PresignResponse {
  uploadUrl: string;
  key: string;
  expiresAt: string; // ISO-8601
}

/** POST /api/videos/:videoId/materials — request body (confirm). */
export interface MaterialConfirmRequest {
  key: string;
  nombre: string;
  contentType: string;
  tamanoBytes: number;
}

/** Canonical material item returned by list, confirm, etc. */
export interface MaterialResponse {
  id: string;
  nombre: string;
  mimeType: string;
  tamanoBytes: number;
  createdAt: string; // ISO-8601
}

/** GET /materials/:id/download — response body. */
export interface DownloadResponse {
  url: string;
  expiresAt: string; // ISO-8601
}
