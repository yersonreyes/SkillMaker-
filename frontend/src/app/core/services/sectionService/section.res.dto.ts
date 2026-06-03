/**
 * section.res.dto.ts — Response DTOs for the Sections API.
 */

import type { VideoItem } from '../videoService/video.res.dto';

export interface SectionItem {
  id: string;
  courseId: string;
  titulo: string;
  orden: number;
  createdAt: string; // ISO 8601
}

/**
 * SectionWithVideos — response shape for GET /api/courses/:id/sections.
 * The backend returns sections with their videos nested and ordered by orden ASC.
 * Used by curso-editar to render the existing content tree on page load (CRITICAL fix).
 */
export interface SectionWithVideos extends SectionItem {
  videos: VideoItem[];
}
