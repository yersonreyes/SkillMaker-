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
   * GET /api/catalog?page=&size=&q=
   * Returns paginated list of approved courses.
   * Empty q is omitted (HttpPromiseBuilderService.queryParam skips empty strings).
   */
  getCatalog(page: number, size: number, q: string): Promise<Page<CatalogCourseCard>> {
    return this.http
      .request<Page<CatalogCourseCard>>()
      .get()
      .url(this.base)
      .queryParam('page', page)
      .queryParam('size', size)
      .queryParam('q', q) // omitted when q === '' (builder skips empty strings)
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
}
