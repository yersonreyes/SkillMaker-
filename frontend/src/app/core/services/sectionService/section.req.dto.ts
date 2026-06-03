/**
 * section.req.dto.ts — Request DTOs for the Sections API.
 */

export interface SectionCreateRequest {
  titulo: string;
  orden?: number;
}

export interface SectionUpdateRequest {
  titulo?: string;
  orden?: number;
}

export interface SectionReorderRequest {
  ids: string[];
}
