import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type { SectionItem, SectionWithVideos } from './section.res.dto';
import type { SectionCreateRequest, SectionUpdateRequest, SectionReorderRequest } from './section.req.dto';

@Injectable({ providedIn: 'root' })
export class SectionService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly coursesBase = `${environment.apiBaseUrl}/courses`;
  private readonly sectionsBase = `${environment.apiBaseUrl}/sections`;

  /** POST /api/courses/:courseId/sections — creates a new section. */
  create(courseId: string, body: SectionCreateRequest): Promise<SectionItem> {
    return this.http
      .request<SectionItem>()
      .post()
      .url(`${this.coursesBase}/${courseId}/sections`)
      .body(body)
      .send();
  }

  /** PATCH /api/sections/:id — partial update (titulo and/or orden). */
  update(id: string, body: SectionUpdateRequest): Promise<SectionItem> {
    return this.http
      .request<SectionItem>()
      .patch()
      .url(`${this.sectionsBase}/${id}`)
      .body(body)
      .send();
  }

  /** DELETE /api/sections/:id — deletes section and cascade-deletes its videos. */
  delete(id: string): Promise<void> {
    return this.http
      .request<void>()
      .delete()
      .url(`${this.sectionsBase}/${id}`)
      .send();
  }

  /** PATCH /api/courses/:courseId/sections/reorder — reorders sections by full ids array. */
  reorder(courseId: string, ids: string[]): Promise<SectionItem[]> {
    const body: SectionReorderRequest = { ids };
    return this.http
      .request<SectionItem[]>()
      .patch()
      .url(`${this.coursesBase}/${courseId}/sections/reorder`)
      .body(body)
      .send();
  }

  /** GET /api/courses/:courseId/sections — returns sections with nested videos ordered by orden. */
  listByCourse(courseId: string): Promise<SectionWithVideos[]> {
    return this.http
      .request<SectionWithVideos[]>()
      .get()
      .url(`${this.coursesBase}/${courseId}/sections`)
      .send();
  }
}
