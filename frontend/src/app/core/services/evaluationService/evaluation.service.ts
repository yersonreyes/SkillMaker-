import { Injectable, inject } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type {
  EvaluationCreateRequest,
  EvaluationUpdateRequest,
  EvaluationResponse,
  EvaluationDetail,
  EvaluationSummary,
} from './evaluation.dto';

@Injectable({ providedIn: 'root' })
export class EvaluationService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly coursesBase = `${environment.apiBaseUrl}/courses`;
  private readonly evaluationsBase = `${environment.apiBaseUrl}/evaluations`;

  /**
   * GET /api/courses/:courseId/evaluation
   * Returns the full nested EvaluationDetail, or null when the course has no
   * evaluation yet (404 is mapped to null — expected "empty state" signal).
   */
  async getByCourse(courseId: string): Promise<EvaluationDetail | null> {
    try {
      return await this.http
        .request<EvaluationDetail>()
        .get()
        .url(`${this.coursesBase}/${courseId}/evaluation`)
        .silent()
        .send();
    } catch (err) {
      if (err instanceof HttpErrorResponse && err.status === 404) {
        return null;
      }
      throw err;
    }
  }

  /** POST /api/courses/:courseId/evaluation — create the 1-1 evaluation. */
  create(courseId: string, body: EvaluationCreateRequest): Promise<EvaluationResponse> {
    return this.http
      .request<EvaluationResponse>()
      .post()
      .url(`${this.coursesBase}/${courseId}/evaluation`)
      .body(body)
      .send();
  }

  /** PATCH /api/evaluations/:id — partial update (notaMinima and/or intentosMax). */
  update(evalId: string, body: EvaluationUpdateRequest): Promise<EvaluationResponse> {
    return this.http
      .request<EvaluationResponse>()
      .patch()
      .url(`${this.evaluationsBase}/${evalId}`)
      .body(body)
      .send();
  }

  /**
   * GET /api/courses/:courseId/evaluation/summary
   * Returns the slim evaluation summary for a student, or null when the course has
   * no evaluation or is not in "aprobado" estado (404 mapped to null).
   */
  async getCourseEvaluationSummary(courseId: string): Promise<EvaluationSummary | null> {
    try {
      return await this.http
        .request<EvaluationSummary>()
        .get()
        .url(`${this.coursesBase}/${courseId}/evaluation/summary`)
        .silent()
        .send();
    } catch (err) {
      if (err instanceof HttpErrorResponse && err.status === 404) {
        return null;
      }
      throw err;
    }
  }
}
