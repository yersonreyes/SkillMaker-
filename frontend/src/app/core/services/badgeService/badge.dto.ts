/**
 * badge.dto.ts — DTOs for the badges API (C5.1).
 *
 * These mirror the Go backend DTOs exactly (BadgeResponse, RankingItem).
 * Fields are optional to match the generated @api types.ts shape.
 */

/** One badge entry from GET /api/badges/me. */
export interface BadgeResponse {
  id?: string;
  nombre?: string;
  descripcion?: string;
  otorgadoEn?: string; // ISO 8601
}

/** One ranking entry from GET /api/badges/ranking. */
export interface RankingItem {
  posicion?: number;
  userNombre?: string;
  certCount?: number;
}
