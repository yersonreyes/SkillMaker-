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

/** Body for POST /api/categorias and PATCH /api/categorias/:id (admin). Slug is derived server-side. */
export interface CategoriaWriteRequest {
  nombre: string;
}
