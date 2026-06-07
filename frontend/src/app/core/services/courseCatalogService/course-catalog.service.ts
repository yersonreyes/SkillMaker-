/**
 * course-catalog.service.ts — Alumno-facing catalog + enrollment API (C2.4).
 *
 * Uses HttpPromiseBuilderService (same pattern as CourseService).
 * Base URL: /api/catalog. My-courses URL: /api/users/me/courses.
 */
import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type {
  Page,
  CatalogCourseCard,
  CourseDetailResponse,
  EnrollmentResponse,
  MyCourseItem,
} from './course-catalog.dto';

@Injectable({ providedIn: 'root' })
export class CourseCatalogService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly base = `${environment.apiBaseUrl}/catalog`;
  private readonly meCoursesUrl = `${environment.apiBaseUrl}/users/me/courses`;

  /**
   * GET /api/catalog?page=&size=&q=&nivel=&categoria=&sort=
   * Returns paginated list of approved courses.
   * - Empty q is omitted (builder skips empty strings).
   * - nivel/sort: omitted when undefined/empty (builder skips undefined/null/'').
   * - categoriaIds: repeated params via queryParamArray (?categoria=A&categoria=B).
   *   Empty array emits nothing.
   */
  getCatalog(
    page: number,
    size: number,
    q: string,
    nivel?: string,
    categoriaIds?: string[],
    sort?: string,
  ): Promise<Page<CatalogCourseCard>> {
    return this.http
      .request<Page<CatalogCourseCard>>()
      .get()
      .url(this.base)
      .queryParam('page', page)
      .queryParam('size', size)
      .queryParam('q', q)
      .queryParam('nivel', nivel)
      .queryParamArray('categoria', categoriaIds ?? [])
      .queryParam('sort', sort)
      .send();
  }

  /**
   * GET /api/catalog/:id
   * Returns CoursePreviewResponse (enrolled=false) or CourseDetailAlumnoResponse (enrolled=true).
   * Caller discriminates via the `enrolled` boolean flag.
   */
  getDetail(id: string): Promise<CourseDetailResponse> {
    return this.http
      .request<CourseDetailResponse>()
      .get()
      .url(`${this.base}/${id}`)
      .send();
  }

  /**
   * POST /api/catalog/:id/enroll
   * Idempotent: second call returns 200 without duplicating the row.
   */
  enroll(id: string): Promise<EnrollmentResponse> {
    return this.http
      .request<EnrollmentResponse>()
      .post()
      .url(`${this.base}/${id}/enroll`)
      .send();
  }

  /**
   * GET /api/users/me/courses
   * Returns the authenticated caller's enrolled courses.
   * NOTE: URL is /users/me/courses, NOT /catalog/me/courses.
   */
  getMyCourses(): Promise<MyCourseItem[]> {
    return this.http
      .request<MyCourseItem[]>()
      .get()
      .url(this.meCoursesUrl)
      .send();
  }

  /**
   * PUT /api/videos/:id/progress — caller-scoped upsert. 204 No Content.
   * NOTE: uses environment.apiBaseUrl directly (not this.base which points to /catalog).
   * Toggle: completado true→false and false→true both accepted.
   */
  markVideoProgress(videoId: string, completado: boolean, lastPositionS?: number): Promise<void> {
    return this.http
      .request<void>()
      .put()
      .url(`${environment.apiBaseUrl}/videos/${videoId}/progress`)
      .body({ completado, lastPositionS })
      .send();
  }
}
