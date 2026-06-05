/**
 * approval.service.ts — Approvals API client (C4.1).
 *
 * Mirrors the HttpPromiseBuilderService pattern used by AttemptService,
 * EvaluationService, and SupervisionService — async/await, no Observables.
 */
import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type {
  PendingItem,
  ApprovalHistoryItem,
  ApprovalSubmitResponse,
} from './approval.dto';

@Injectable({ providedIn: 'root' })
export class ApprovalService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly coursesBase = `${environment.apiBaseUrl}/courses`;
  private readonly approvalsBase = `${environment.apiBaseUrl}/approvals`;

  /**
   * POST /api/courses/:courseId/submit
   * Submits a course for admin review. Course must be in borrador or rechazado estado.
   * Transitions estado to en_revision on success.
   */
  submitToReview(courseId: string): Promise<ApprovalSubmitResponse> {
    return this.http
      .request<ApprovalSubmitResponse>()
      .post()
      .url(`${this.coursesBase}/${courseId}/submit`)
      .send();
  }

  /**
   * GET /api/approvals/pending
   * Returns all courses with estado=en_revision. Admin only.
   */
  listPending(): Promise<PendingItem[]> {
    return this.http
      .request<PendingItem[]>()
      .get()
      .url(`${this.approvalsBase}/pending`)
      .send();
  }

  /**
   * POST /api/courses/:courseId/approve
   * Approves a course in review. comentario is optional.
   * Transitions estado to aprobado and stamps publicado_en.
   */
  approve(courseId: string, comentario?: string): Promise<void> {
    const body: { comentario?: string } = comentario !== undefined ? { comentario } : {};
    return this.http
      .request<void>()
      .post()
      .url(`${this.coursesBase}/${courseId}/approve`)
      .body(body)
      .send();
  }

  /**
   * POST /api/courses/:courseId/reject
   * Rejects a course in review. comentario is required (enforced on both FE and BE).
   * Transitions estado to rechazado.
   */
  reject(courseId: string, comentario: string): Promise<void> {
    return this.http
      .request<void>()
      .post()
      .url(`${this.coursesBase}/${courseId}/reject`)
      .body({ comentario })
      .send();
  }

  /**
   * GET /api/courses/:courseId/approvals
   * Returns approval/rejection history for a course.
   * Accessible by the course owner (creador) or an admin.
   */
  history(courseId: string): Promise<ApprovalHistoryItem[]> {
    return this.http
      .request<ApprovalHistoryItem[]>()
      .get()
      .url(`${this.coursesBase}/${courseId}/approvals`)
      .send();
  }
}
