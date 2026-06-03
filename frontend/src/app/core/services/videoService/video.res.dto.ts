/**
 * video.res.dto.ts — Response DTOs for the Videos API.
 */

import type { VideoProveedor } from './video.req.dto';

export interface VideoItem {
  id: string;
  sectionId: string;
  titulo: string;
  url: string;
  proveedor: VideoProveedor;
  duracionS: number;
  orden: number;
  createdAt: string; // ISO 8601
}
