/**
 * categoria.dto.ts — DTOs for the Categorias API (course-structure-v2).
 * Mirrors the backend CategoriaResponse exactly.
 */

/** One curated categoria item from GET /api/categorias. */
export interface CategoriaItem {
  id: string;
  nombre: string;
  slug: string;
}
