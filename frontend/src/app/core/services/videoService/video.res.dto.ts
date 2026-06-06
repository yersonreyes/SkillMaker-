/**
 * video.res.dto.ts — Response DTOs for the Videos API.
 * Updated in course-structure-v2: VideoItem gains descripcion + materiales[].
 */

import type { VideoProveedor } from './video.req.dto';
import type { MaterialResponse } from '../materialService/material.types';

export interface VideoItem {
  id: string;
  sectionId: string;
  titulo: string;
  url: string;
  proveedor: VideoProveedor;
  duracionS: number;
  orden: number;
  /** Description of this video (default ''). */
  descripcion: string;
  /** Per-video materials (populated in the content tree detail view). */
  materiales: MaterialResponse[];
  createdAt: string; // ISO 8601
}
