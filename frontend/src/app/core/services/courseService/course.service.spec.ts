/**
 * course.service.spec.ts — CourseService unit tests (Vitest + Angular TestBed).
 * Uses provideHttpClient + provideHttpClientTesting (modern API).
 * Mock strategy: HttpTestingController intercepts real HTTP calls.
 * Updated in course-structure-v2: thumbnail presign/confirm + metadata fields.
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { CourseService } from './course.service';
import type { CourseListItem, CourseDetail, Page } from './course.res.dto';

const BASE = 'http://localhost:3000/api/courses';

const MOCK_COURSE_LIST_ITEM: CourseListItem = {
  id: 'course-1',
  titulo: 'Go avanzado',
  estado: 'borrador',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

const MOCK_COURSE_DETAIL: CourseDetail = {
  id: 'course-1',
  creadorId: 'user-1',
  titulo: 'Go avanzado',
  descripcion: 'Curso de Go',
  estado: 'borrador',
  hasContent: false,
  nivel: null,
  horasPractico: 0,
  miniaturaUrl: null,
  horasVideo: 0,
  cantidadClases: 0,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

const MOCK_PAGE: Page<CourseListItem> = {
  items: [MOCK_COURSE_LIST_ITEM],
  page: 1,
  size: 20,
  total: 1,
  totalPages: 1,
};

describe('CourseService', () => {
  let service: CourseService;
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
    service = TestBed.inject(CourseService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── listByMe ─────────────────────────────────────────────────────────────────

  it('listByMe() sends GET /api/courses with creator=me param', async () => {
    const promise = service.listByMe({});

    const req = httpMock.expectOne(r => r.url === BASE && r.method === 'GET');
    expect(req.request.params.get('creator')).toBe('me');
    req.flush(MOCK_PAGE);

    const result = await promise;
    expect(result.items).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('listByMe() sends page and size query params', async () => {
    const promise = service.listByMe({ page: 2, size: 10 });

    const req = httpMock.expectOne(r => r.url === BASE && r.method === 'GET');
    expect(req.request.params.get('creator')).toBe('me');
    expect(req.request.params.get('page')).toBe('2');
    expect(req.request.params.get('size')).toBe('10');
    req.flush({ ...MOCK_PAGE, page: 2, size: 10 });

    await promise;
  });

  it('listByMe() omits page/size when not provided', async () => {
    const promise = service.listByMe({});

    const req = httpMock.expectOne(r => r.url === BASE && r.method === 'GET');
    expect(req.request.params.has('page')).toBe(false);
    expect(req.request.params.has('size')).toBe(false);
    req.flush(MOCK_PAGE);

    await promise;
  });

  // ── getById ──────────────────────────────────────────────────────────────────

  it('getById() sends GET /api/courses/:id', async () => {
    const promise = service.getById('course-1');

    const req = httpMock.expectOne(`${BASE}/course-1`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_COURSE_DETAIL);

    const result = await promise;
    expect(result['id']).toBe('course-1');
    expect(result['estado']).toBe('borrador');
  });

  // ── create ───────────────────────────────────────────────────────────────────

  it('create() sends POST /api/courses with titulo and descripcion', async () => {
    const promise = service.create({ titulo: 'Go avanzado', descripcion: 'Curso de Go' });

    const req = httpMock.expectOne(BASE);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ titulo: 'Go avanzado', descripcion: 'Curso de Go' });
    req.flush(MOCK_COURSE_DETAIL, { status: 201, statusText: 'Created' });

    const result = await promise;
    expect(result['estado']).toBe('borrador');
    expect(result['titulo']).toBe('Go avanzado');
  });

  it('create() sends POST with only titulo when descripcion is omitted', async () => {
    const promise = service.create({ titulo: 'Solo titulo' });

    const req = httpMock.expectOne(BASE);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ titulo: 'Solo titulo' });
    req.flush({ ...MOCK_COURSE_DETAIL, titulo: 'Solo titulo', descripcion: '' });

    await promise;
  });

  it('create() sends nivel and categoriaIds when provided', async () => {
    const promise = service.create({
      titulo: 'Go avanzado',
      nivel: 'intermedio',
      categoriaIds: ['cat-1', 'cat-2'],
    });

    const req = httpMock.expectOne(BASE);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({
      titulo: 'Go avanzado',
      nivel: 'intermedio',
      categoriaIds: ['cat-1', 'cat-2'],
    });
    req.flush(MOCK_COURSE_DETAIL, { status: 201, statusText: 'Created' });

    await promise;
  });

  // ── update ───────────────────────────────────────────────────────────────────

  it('update() sends PATCH /api/courses/:id with partial body', async () => {
    const promise = service.update('course-1', { titulo: 'Nuevo titulo' });

    const req = httpMock.expectOne(`${BASE}/course-1`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ titulo: 'Nuevo titulo' });
    req.flush({ ...MOCK_COURSE_DETAIL, titulo: 'Nuevo titulo' });

    const result = await promise;
    expect(result['titulo']).toBe('Nuevo titulo');
  });

  it('update() sends PATCH with descripcion only', async () => {
    const promise = service.update('course-1', { descripcion: 'Nueva descripcion' });

    const req = httpMock.expectOne(`${BASE}/course-1`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ descripcion: 'Nueva descripcion' });
    req.flush({ ...MOCK_COURSE_DETAIL, descripcion: 'Nueva descripcion' });

    await promise;
  });

  it('update() sends PATCH with nivel and categoriaIds', async () => {
    const promise = service.update('course-1', {
      nivel: 'avanzado',
      horasPractico: 4.5,
      categoriaIds: ['cat-3'],
    });

    const req = httpMock.expectOne(`${BASE}/course-1`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({
      nivel: 'avanzado',
      horasPractico: 4.5,
      categoriaIds: ['cat-3'],
    });
    req.flush({ ...MOCK_COURSE_DETAIL, nivel: 'avanzado', horasPractico: 4.5 });

    await promise;
  });

  // ── presignThumbnail ──────────────────────────────────────────────────────────

  it('presignThumbnail() sends POST /api/courses/:courseId/thumbnail/presign', async () => {
    const body = { nombre: 'cover.jpg', contentType: 'image/jpeg', tamanoBytes: 200_000 };
    const promise = service.presignThumbnail('course-1', body);

    const req = httpMock.expectOne(`${BASE}/course-1/thumbnail/presign`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(body);
    req.flush({
      uploadUrl: 'http://minio/presigned-thumb',
      key: 'courses/course-1/thumbnail/uuid-cover.jpg',
      expiresAt: '2026-06-03T16:00:00Z',
    });

    const result = await promise;
    expect(result.uploadUrl).toContain('minio');
    expect(result.key).toContain('thumbnail');
  });

  // ── confirmThumbnail ──────────────────────────────────────────────────────────

  it('confirmThumbnail() sends POST /api/courses/:courseId/thumbnail with key', async () => {
    const body = { key: 'courses/course-1/thumbnail/uuid-cover.jpg' };
    const promise = service.confirmThumbnail('course-1', body);

    const req = httpMock.expectOne(`${BASE}/course-1/thumbnail`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(body);
    req.flush(null, { status: 200, statusText: 'OK' });

    await promise;
  });
});
