/**
 * approval.service.spec.ts — ApprovalService unit tests (Vitest + Angular TestBed).
 *
 * Strict TDD — specs written FIRST (RED), then service implemented (GREEN).
 *
 * Covers:
 *  - submitToReview(courseId) → POST /api/courses/:courseId/submit
 *  - listPending()            → GET  /api/approvals/pending
 *  - approve(courseId, comentario?) → POST /api/courses/:courseId/approve
 *  - reject(courseId, comentario)   → POST /api/courses/:courseId/reject
 *  - history(courseId)        → GET  /api/courses/:courseId/approvals
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { ApprovalService } from './approval.service';
import type { PendingItem, ApprovalHistoryItem, ApprovalSubmitResponse } from './approval.dto';

const BASE = 'http://localhost:3000/api';

// ── Fixtures ──────────────────────────────────────────────────────────────────

const MOCK_PENDING_ITEM: PendingItem = {
  id: 'course-1',
  titulo: 'Go Avanzado',
  creadorId: 'user-1',
  estado: 'en_revision',
  fechaEnvio: '2026-06-01T10:00:00Z',
};

const MOCK_HISTORY_ITEM: ApprovalHistoryItem = {
  id: 'approval-1',
  resultado: 'rechazado',
  comentario: 'Falta contenido en la evaluacion',
  adminId: 'admin-1',
  resueltoEn: '2026-06-02T12:00:00Z',
};

const MOCK_SUBMIT_RESPONSE: ApprovalSubmitResponse = {
  courseId: 'course-1',
  estado: 'en_revision',
};

// ── ApprovalService HTTP specs ────────────────────────────────────────────────

describe('ApprovalService', () => {
  let service: ApprovalService;
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
    service = TestBed.inject(ApprovalService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── submitToReview ────────────────────────────────────────────────────────────

  it('submitToReview() sends POST /api/courses/:courseId/submit with empty body', async () => {
    const promise = service.submitToReview('course-1');

    const req = httpMock.expectOne(`${BASE}/courses/course-1/submit`);
    expect(req.request.method).toBe('POST');
    req.flush(MOCK_SUBMIT_RESPONSE);

    const result = await promise;
    expect(result.courseId).toBe('course-1');
    expect(result.estado).toBe('en_revision');
  });

  it('submitToReview() sends to correct URL for a different courseId', async () => {
    const promise = service.submitToReview('course-99');

    const req = httpMock.expectOne(`${BASE}/courses/course-99/submit`);
    expect(req.request.method).toBe('POST');
    req.flush({ courseId: 'course-99', estado: 'en_revision' });

    const result = await promise;
    expect(result.courseId).toBe('course-99');
  });

  // ── listPending ───────────────────────────────────────────────────────────────

  it('listPending() sends GET /api/approvals/pending and returns PendingItem[]', async () => {
    const promise = service.listPending();

    const req = httpMock.expectOne(`${BASE}/approvals/pending`);
    expect(req.request.method).toBe('GET');
    req.flush([MOCK_PENDING_ITEM]);

    const result = await promise;
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('course-1');
    expect(result[0].titulo).toBe('Go Avanzado');
    expect(result[0].estado).toBe('en_revision');
  });

  it('listPending() returns empty array when no courses are pending', async () => {
    const promise = service.listPending();

    const req = httpMock.expectOne(`${BASE}/approvals/pending`);
    req.flush([]);

    const result = await promise;
    expect(result).toHaveLength(0);
  });

  // ── approve ───────────────────────────────────────────────────────────────────

  it('approve() sends POST /api/courses/:courseId/approve with empty body when no comentario', async () => {
    const promise = service.approve('course-1');

    const req = httpMock.expectOne(`${BASE}/courses/course-1/approve`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({});
    req.flush(null, { status: 200, statusText: 'OK' });

    await promise; // resolves without error
  });

  it('approve() sends POST /api/courses/:courseId/approve with comentario when provided', async () => {
    const promise = service.approve('course-1', 'Excelente contenido');

    const req = httpMock.expectOne(`${BASE}/courses/course-1/approve`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ comentario: 'Excelente contenido' });
    req.flush(null, { status: 200, statusText: 'OK' });

    await promise;
  });

  it('approve() sends to correct URL for a different courseId', async () => {
    const promise = service.approve('course-77');

    const req = httpMock.expectOne(`${BASE}/courses/course-77/approve`);
    expect(req.request.method).toBe('POST');
    req.flush(null, { status: 200, statusText: 'OK' });

    await promise;
  });

  // ── reject ────────────────────────────────────────────────────────────────────

  it('reject() sends POST /api/courses/:courseId/reject with required comentario', async () => {
    const promise = service.reject('course-1', 'Falta evaluacion completa');

    const req = httpMock.expectOne(`${BASE}/courses/course-1/reject`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ comentario: 'Falta evaluacion completa' });
    req.flush(null, { status: 200, statusText: 'OK' });

    await promise;
  });

  it('reject() sends to correct URL for a different courseId', async () => {
    const promise = service.reject('course-55', 'Contenido insuficiente');

    const req = httpMock.expectOne(`${BASE}/courses/course-55/reject`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ comentario: 'Contenido insuficiente' });
    req.flush(null, { status: 200, statusText: 'OK' });

    await promise;
  });

  // ── history ───────────────────────────────────────────────────────────────────

  it('history() sends GET /api/courses/:courseId/approvals and returns ApprovalHistoryItem[]', async () => {
    const promise = service.history('course-1');

    const req = httpMock.expectOne(`${BASE}/courses/course-1/approvals`);
    expect(req.request.method).toBe('GET');
    req.flush([MOCK_HISTORY_ITEM]);

    const result = await promise;
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('approval-1');
    expect(result[0].resultado).toBe('rechazado');
    expect(result[0].comentario).toBe('Falta contenido en la evaluacion');
    expect(result[0].adminId).toBe('admin-1');
  });

  it('history() returns empty array when course has no approval history', async () => {
    const promise = service.history('course-1');

    const req = httpMock.expectOne(`${BASE}/courses/course-1/approvals`);
    req.flush([]);

    const result = await promise;
    expect(result).toHaveLength(0);
  });

  it('history() sends to correct URL for a different courseId', async () => {
    const promise = service.history('course-42');

    const req = httpMock.expectOne(`${BASE}/courses/course-42/approvals`);
    expect(req.request.method).toBe('GET');
    req.flush([{ ...MOCK_HISTORY_ITEM, id: 'approval-2' }]);

    const result = await promise;
    expect(result[0].id).toBe('approval-2');
  });
});
