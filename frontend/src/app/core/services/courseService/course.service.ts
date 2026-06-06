import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type { Page, CourseListItem, CourseDetail } from './course.res.dto';
import type {
  CourseListParams,
  CreateCourseRequest,
  UpdateCourseRequest,
  ThumbnailPresignRequest,
  ThumbnailPresignResponse,
  ThumbnailConfirmRequest,
} from './course.req.dto';

@Injectable({ providedIn: 'root' })
export class CourseService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly base = `${environment.apiBaseUrl}/courses`;

  /** GET /api/courses?creator=me — paginated list of the caller's courses. */
  listByMe(params: CourseListParams): Promise<Page<CourseListItem>> {
    return this.http
      .request<Page<CourseListItem>>()
      .get()
      .url(this.base)
      .queryParam('creator', 'me')
      .queryParam('page', params.page)
      .queryParam('size', params.size)
      .send();
  }

  /** GET /api/courses/:id — full course detail (owner only; non-owner → 404). */
  getById(id: string): Promise<CourseDetail> {
    return this.http
      .request<CourseDetail>()
      .get()
      .url(`${this.base}/${id}`)
      .send();
  }

  /** POST /api/courses — creates a new course in estado borrador. */
  create(body: CreateCourseRequest): Promise<CourseDetail> {
    return this.http
      .request<CourseDetail>()
      .post()
      .url(this.base)
      .body(body)
      .send();
  }

  /**
   * PATCH /api/courses/:id — partial update; only permitted when
   * estado ∈ {borrador, rechazado} and the caller is the owner.
   */
  update(id: string, body: UpdateCourseRequest): Promise<CourseDetail> {
    return this.http
      .request<CourseDetail>()
      .patch()
      .url(`${this.base}/${id}`)
      .body(body)
      .send();
  }

  /**
   * POST /api/courses/:courseId/thumbnail/presign
   * Requests a presigned PUT URL for uploading the course thumbnail.
   * Owner + assertCourseEditable gated on the backend.
   */
  presignThumbnail(courseId: string, body: ThumbnailPresignRequest): Promise<ThumbnailPresignResponse> {
    return this.http
      .request<ThumbnailPresignResponse>()
      .post()
      .url(`${this.base}/${courseId}/thumbnail/presign`)
      .body(body)
      .send();
  }

  /**
   * POST /api/courses/:courseId/thumbnail
   * Confirms the thumbnail upload and sets miniatura_key on the course.
   */
  confirmThumbnail(courseId: string, body: ThumbnailConfirmRequest): Promise<void> {
    return this.http
      .request<void>()
      .post()
      .url(`${this.base}/${courseId}/thumbnail`)
      .body(body)
      .send();
  }
}
