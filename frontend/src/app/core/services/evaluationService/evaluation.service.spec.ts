/**
 * evaluation.service.spec.ts — EvaluationService unit tests (Vitest + Angular TestBed).
 * Verifies correct URLs, HTTP verbs, and request bodies.
 * Covers: getByCourse (200 + 404→null), create, update, getCourseEvaluationSummary.
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';
import { EvaluationService } from './evaluation.service';
import type { EvaluationDetail, EvaluationResponse, EvaluationSummary } from './evaluation.dto';

const COURSES_BASE = 'http://localhost:3000/api/courses';
const EVALUATIONS_BASE = 'http://localhost:3000/api/evaluations';

const MOCK_DETAIL: EvaluationDetail = {
  id: 'eval-1',
  courseId: 'course-1',
  notaMinima: 70,
  intentosMax: 3,
  questions: [],
};

const MOCK_RESPONSE: EvaluationResponse = {
  id: 'eval-1',
  courseId: 'course-1',
  notaMinima: 70,
  intentosMax: 3,
  createdAt: '2026-01-01T00:00:00Z',
};

describe('EvaluationService', () => {
  let service: EvaluationService;
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
    service = TestBed.inject(EvaluationService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── getByCourse ───────────────────────────────────────────────────────────────

  it('getByCourse() sends GET /api/courses/:courseId/evaluation and returns detail', async () => {
    const promise = service.getByCourse('course-1');

    const req = httpMock.expectOne(`${COURSES_BASE}/course-1/evaluation`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_DETAIL);

    const result = await promise;
    expect(result).not.toBeNull();
    expect(result!.id).toBe('eval-1');
    expect(result!.courseId).toBe('course-1');
    expect(result!.notaMinima).toBe(70);
  });

  it('getByCourse() returns null when the server returns 404 (no evaluation exists)', async () => {
    const promise = service.getByCourse('course-1');

    const req = httpMock.expectOne(`${COURSES_BASE}/course-1/evaluation`);
    req.flush({ code: 'not_found', message: 'evaluation not found' }, {
      status: 404,
      statusText: 'Not Found',
    });

    const result = await promise;
    expect(result).toBeNull();
  });

  // ── create ────────────────────────────────────────────────────────────────────

  it('create() sends POST /api/courses/:courseId/evaluation with body', async () => {
    const body = { notaMinima: 70, intentosMax: 3 };
    const promise = service.create('course-1', body);

    const req = httpMock.expectOne(`${COURSES_BASE}/course-1/evaluation`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(body);
    req.flush(MOCK_RESPONSE, { status: 201, statusText: 'Created' });

    const result = await promise;
    expect(result.id).toBe('eval-1');
  });

  it('create() sends default values when body fields are omitted', async () => {
    const promise = service.create('course-1', {});

    const req = httpMock.expectOne(`${COURSES_BASE}/course-1/evaluation`);
    expect(req.request.body).toEqual({});
    req.flush(MOCK_RESPONSE, { status: 201, statusText: 'Created' });

    await promise;
  });

  // ── update ────────────────────────────────────────────────────────────────────

  it('update() sends PATCH /api/evaluations/:id with partial body', async () => {
    const body = { notaMinima: 80 };
    const promise = service.update('eval-1', body);

    const req = httpMock.expectOne(`${EVALUATIONS_BASE}/eval-1`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual(body);
    req.flush({ ...MOCK_RESPONSE, notaMinima: 80 });

    const result = await promise;
    expect(result.notaMinima).toBe(80);
  });

  it('update() sends intentosMax-only patch', async () => {
    const body = { intentosMax: 5 };
    const promise = service.update('eval-1', body);

    const req = httpMock.expectOne(`${EVALUATIONS_BASE}/eval-1`);
    expect(req.request.body).toEqual(body);
    req.flush({ ...MOCK_RESPONSE, intentosMax: 5 });

    const result = await promise;
    expect(result.intentosMax).toBe(5);
  });

  // ── getCourseEvaluationSummary ─────────────────────────────────────────────────

  it('getCourseEvaluationSummary() sends GET /api/courses/:courseId/evaluation/summary', async () => {
    const mockSummary: EvaluationSummary = {
      evaluationId: 'eval-1',
      notaMinima: 75,
      intentosMax: 2,
    };
    const promise = service.getCourseEvaluationSummary('course-1');

    const req = httpMock.expectOne(`${COURSES_BASE}/course-1/evaluation/summary`);
    expect(req.request.method).toBe('GET');
    req.flush(mockSummary);

    const result = await promise;
    expect(result).not.toBeNull();
    expect(result!.evaluationId).toBe('eval-1');
    expect(result!.notaMinima).toBe(75);
    expect(result!.intentosMax).toBe(2);
  });

  it('getCourseEvaluationSummary() returns null when server returns 404', async () => {
    const promise = service.getCourseEvaluationSummary('course-1');

    const req = httpMock.expectOne(`${COURSES_BASE}/course-1/evaluation/summary`);
    req.flush({ code: 'not_found', message: 'evaluation not found' }, {
      status: 404,
      statusText: 'Not Found',
    });

    const result = await promise;
    expect(result).toBeNull();
  });
});
