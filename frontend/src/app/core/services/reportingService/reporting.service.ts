/**
 * reporting.service.ts — Reporting API client (C6.1).
 *
 * Exposes 4 methods for the 4 reporting endpoints.
 * Uses HttpPromiseBuilderService (same pattern as CourseCatalogService).
 * Base URL: /api/reports.
 */
import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type {
  GlobalReportResponse,
  CourseReportItem,
  UserProgressResponse,
  TeamReportItem,
} from './reporting.dto';

@Injectable({ providedIn: 'root' })
export class ReportingService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly base = `${environment.apiBaseUrl}/reports`;

  /**
   * GET /api/reports/global
   * Returns system-wide aggregate metrics. Admin only (enforced server-side).
   */
  getGlobal(): Promise<GlobalReportResponse> {
    return this.http
      .request<GlobalReportResponse>()
      .get()
      .url(`${this.base}/global`)
      .send();
  }

  /**
   * GET /api/reports/courses
   * Returns all courses with report stats. Admin only.
   */
  getCourses(): Promise<CourseReportItem[]> {
    return this.http
      .request<CourseReportItem[]>()
      .get()
      .url(`${this.base}/courses`)
      .send();
  }

  /**
   * GET /api/reports/users/:id/progress
   * Returns a user's progress aggregate. Admin or same user (enforced server-side).
   */
  getUserProgress(id: string): Promise<UserProgressResponse> {
    return this.http
      .request<UserProgressResponse>()
      .get()
      .url(`${this.base}/users/${id}/progress`)
      .send();
  }

  /**
   * GET /api/reports/team
   * Returns the caller's supervised employees with progress. Supervisor only.
   */
  getTeam(): Promise<TeamReportItem[]> {
    return this.http
      .request<TeamReportItem[]>()
      .get()
      .url(`${this.base}/team`)
      .send();
  }
}
