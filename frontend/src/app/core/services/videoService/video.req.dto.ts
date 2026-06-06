/**
 * video.req.dto.ts — Request DTOs for the Videos API.
 * Updated in course-structure-v2: added descripcion field.
 */

export type VideoProveedor = 'youtube' | 'vimeo';

export interface VideoCreateRequest {
  titulo: string;
  url: string;
  proveedor: VideoProveedor;
  duracionS?: number;
  /** Optional description for the video (max 5000 chars). */
  descripcion?: string;
}

export interface VideoUpdateRequest {
  titulo?: string;
  url?: string;
  proveedor?: VideoProveedor;
  duracionS?: number;
  /** Optional description for the video (max 5000 chars). */
  descripcion?: string;
}
