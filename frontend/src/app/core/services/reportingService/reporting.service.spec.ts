/**
 * reporting.service.spec.ts — ReportingService unit tests (Strict TDD — RED → GREEN).
 *
 * Strategy: HttpTestingController intercepts real HTTP calls (same pattern as course-catalog.service.spec.ts).
 *
 * Covers:
 *  - getGlobal() calls GET /api/reports/global
 *  - getCourses() calls GET /api/reports/courses
 *  - getTeam() calls GET /api/reports/team
 *  - getUserProgress(id) calls GET /api/reports/users/{id}/progress
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { ReportingService } from './reporting.service';
import type {
  GlobalReportResponse,
  CourseReportItem,
  TeamReportItem,
  UserProgressResponse,
} from './reporting.dto';

const REPORTS_BASE = 'http://localhost:3000/api/reports';

const MOCK_GLOBAL: GlobalReportResponse = {
  activeUsers: 42,
  totalAttempts: 100,
  certificatesIssued: 15,
  coursesByEstado: [
    { estado: 'aprobado', total: 10 },
    { estado: 'borrador', total: 3 },
    { estado: 'en_revision', total: 2 },
    { estado: 'rechazado', total: 1 },
  ],
  topCreators: [{ nombre: 'Ana Lopez', total: 5 }],
  usersPerMonth: [{ month: '2026-01', total: 20 }],
  approvedCoursesPerMonth: [{ month: '2026-01', total: 5 }],
};

const MOCK_COURSES: CourseReportItem[] = [
  {
    courseId: 'course-1',
    titulo: 'Go Avanzado',
    estado: 'aprobado',
    enrollments: 10,
    attempts: 8,
    approvalRate: 0.75,
  },
];

const MOCK_TEAM: TeamReportItem[] = [
  {
    empleadoId: 'user-1',
    empleadoNombre: 'Bob Martinez',
    enrolledCount: 3,
    completedCount: 1,
    lastAttemptDate: '2026-05-15',
  },
];

const MOCK_PROGRESS: UserProgressResponse = {
  enrolledCount: 5,
  completedCount: 3,
  attemptsCount: 10,
  passedAttemptsCount: 7,
  certificatesCount: 2,
};

describe('ReportingService', () => {
  let service: ReportingService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        ConfirmationService,
        MessageService,
      ],
    });
    service = TestBed.inject(ReportingService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── getGlobal ─────────────────────────────────────────────────────────────────

  it('getGlobal() sends GET /api/reports/global', async () => {
    const promise = service.getGlobal();

    const req = httpMock.expectOne(`${REPORTS_BASE}/global`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_GLOBAL);

    const result = await promise;
    expect(result.activeUsers).toBe(42);
    expect(result.coursesByEstado).toHaveLength(4);
  });

  // ── getCourses ────────────────────────────────────────────────────────────────

  it('getCourses() sends GET /api/reports/courses', async () => {
    const promise = service.getCourses();

    const req = httpMock.expectOne(`${REPORTS_BASE}/courses`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_COURSES);

    const result = await promise;
    expect(result).toHaveLength(1);
    expect(result[0].titulo).toBe('Go Avanzado');
  });

  // ── getTeam ───────────────────────────────────────────────────────────────────

  it('getTeam() sends GET /api/reports/team', async () => {
    const promise = service.getTeam();

    const req = httpMock.expectOne(`${REPORTS_BASE}/team`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_TEAM);

    const result = await promise;
    expect(result).toHaveLength(1);
    expect(result[0].empleadoNombre).toBe('Bob Martinez');
  });

  // ── getUserProgress ───────────────────────────────────────────────────────────

  it('getUserProgress(id) sends GET /api/reports/users/{id}/progress', async () => {
    const userId = 'user-abc-123';
    const promise = service.getUserProgress(userId);

    const req = httpMock.expectOne(`${REPORTS_BASE}/users/${userId}/progress`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_PROGRESS);

    const result = await promise;
    expect(result.enrolledCount).toBe(5);
    expect(result.passedAttemptsCount).toBe(7);
  });
});
