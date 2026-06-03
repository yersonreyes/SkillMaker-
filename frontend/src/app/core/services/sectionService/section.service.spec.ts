/**
 * section.service.spec.ts — SectionService unit tests (Vitest + Angular TestBed).
 * Uses provideHttpClient + provideHttpClientTesting.
 * Covers URL shapes, HTTP methods, and request bodies.
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { SectionService } from './section.service';
import type { SectionItem } from './section.res.dto';

const COURSES_BASE = 'http://localhost:3000/api/courses';
const SECTIONS_BASE = 'http://localhost:3000/api/sections';

const MOCK_SECTION: SectionItem = {
  id: 'sec-1',
  courseId: 'course-1',
  titulo: 'Introduccion',
  orden: 0,
  createdAt: '2026-01-01T00:00:00Z',
};

describe('SectionService', () => {
  let service: SectionService;
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
    service = TestBed.inject(SectionService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── create ────────────────────────────────────────────────────────────────────

  it('create() sends POST /api/courses/:courseId/sections with titulo', async () => {
    const promise = service.create('course-1', { titulo: 'Introduccion' });

    const req = httpMock.expectOne(`${COURSES_BASE}/course-1/sections`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ titulo: 'Introduccion' });
    req.flush(MOCK_SECTION, { status: 201, statusText: 'Created' });

    const result = await promise;
    expect(result['id']).toBe('sec-1');
  });

  it('create() sends optional orden field when provided', async () => {
    const promise = service.create('course-1', { titulo: 'Cap 1', orden: 2 });

    const req = httpMock.expectOne(`${COURSES_BASE}/course-1/sections`);
    expect(req.request.body).toEqual({ titulo: 'Cap 1', orden: 2 });
    req.flush({ ...MOCK_SECTION, orden: 2 });

    await promise;
  });

  // ── update ────────────────────────────────────────────────────────────────────

  it('update() sends PATCH /api/sections/:id with partial body', async () => {
    const promise = service.update('sec-1', { titulo: 'Nuevo titulo' });

    const req = httpMock.expectOne(`${SECTIONS_BASE}/sec-1`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ titulo: 'Nuevo titulo' });
    req.flush({ ...MOCK_SECTION, titulo: 'Nuevo titulo' });

    const result = await promise;
    expect(result['titulo']).toBe('Nuevo titulo');
  });

  // ── delete ────────────────────────────────────────────────────────────────────

  it('delete() sends DELETE /api/sections/:id', async () => {
    const promise = service.delete('sec-1');

    const req = httpMock.expectOne(`${SECTIONS_BASE}/sec-1`);
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });

    await promise;
  });

  // ── reorder ───────────────────────────────────────────────────────────────────

  it('reorder() sends PATCH /api/courses/:courseId/sections/reorder with ids array', async () => {
    const ids = ['sec-3', 'sec-1', 'sec-2'];
    const promise = service.reorder('course-1', ids);

    const req = httpMock.expectOne(`${COURSES_BASE}/course-1/sections/reorder`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ ids });
    req.flush([]);

    await promise;
  });

  // ── listByCourse ──────────────────────────────────────────────────────────────

  it('listByCourse() sends GET /api/courses/:courseId/sections', async () => {
    const promise = service.listByCourse('course-1');

    const req = httpMock.expectOne(`${COURSES_BASE}/course-1/sections`);
    expect(req.request.method).toBe('GET');
    req.flush([MOCK_SECTION]);

    const result = await promise;
    expect(result).toHaveLength(1);
    expect(result[0]['id']).toBe('sec-1');
  });
});
