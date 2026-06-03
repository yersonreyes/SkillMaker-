/**
 * video.req.dto.ts — Request DTOs for the Videos API.
 */

export type VideoProveedor = 'youtube' | 'vimeo';

export interface VideoCreateRequest {
  titulo: string;
  url: string;
  proveedor: VideoProveedor;
  duracionS?: number;
}

export interface VideoUpdateRequest {
  titulo?: string;
  url?: string;
  proveedor?: VideoProveedor;
  duracionS?: number;
}
