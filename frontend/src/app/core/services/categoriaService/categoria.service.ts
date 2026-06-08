/**
 * categoria.service.ts — Categorias lookup service (course-structure-v2).
 *
 * GET /api/categorias returns the full curated list of categorias.
 * Used in curso-editar to populate the p-multiselect options.
 */
import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type { CategoriaItem, CategoriaWriteRequest } from './categoria.dto';

@Injectable({ providedIn: 'root' })
export class CategoriaService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly base = `${environment.apiBaseUrl}/categorias`;

  /**
   * GET /api/categorias
   * Returns the full list of curated categorias (JWT required — any authenticated user).
   */
  getAll(): Promise<CategoriaItem[]> {
    return this.http
      .request<CategoriaItem[]>()
      .get()
      .url(this.base)
      .send();
  }

  /** POST /api/categorias (admin) — creates a categoria; slug derived server-side. */
  create(body: CategoriaWriteRequest): Promise<CategoriaItem> {
    return this.http
      .request<CategoriaItem>()
      .post()
      .url(this.base)
      .body(body)
      .send();
  }

  /** PATCH /api/categorias/:id (admin) — renames a categoria. */
  update(id: string, body: CategoriaWriteRequest): Promise<CategoriaItem> {
    return this.http
      .request<CategoriaItem>()
      .patch()
      .url(`${this.base}/${id}`)
      .body(body)
      .send();
  }

  /** DELETE /api/categorias/:id (admin) — blocked (409) if assigned to courses. */
  delete(id: string): Promise<void> {
    return this.http
      .request<void>()
      .delete()
      .url(`${this.base}/${id}`)
      .send();
  }
}
