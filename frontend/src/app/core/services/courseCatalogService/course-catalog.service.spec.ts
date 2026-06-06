/**
 * course-catalog.service.spec.ts — CourseCatalogService unit tests (Strict TDD — RED → GREEN).
 *
 * Strategy: HttpTestingController intercepts real HTTP calls (same pattern as course.service.spec.ts).
 *
 * Covers:
 *  - getCatalog() builds correct URL + page/size/q params; omits q when empty
 *  - getDetail() builds correct URL
 *  - enroll() sends POST to correct URL
 *  - getMyCourses() builds correct URL (/users/me/courses, not /catalog/me)
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { CourseCatalogService } from './course-catalog.service';
import type {
  CatalogCourseCard,
  Page,
  CoursePreviewResponse,
  CourseDetailAlumnoResponse,
  EnrollmentResponse,
  MyCourseItem,
} from './course-catalog.dto';

const CATALOG_BASE = 'http://localhost:3000/api/catalog';
const MY_COURSES_URL = 'http://localhost:3000/api/users/me/courses';

const MOCK_CARD: CatalogCourseCard = {
  id: 'course-1',
  titulo: 'Go Avanzado',
  descripcion: 'Aprende Go de verdad',
  creadorNombre: 'Yerson Reyes',
  createdAt: '2026-01-01T00:00:00Z',
  // course-structure-v2 metadata fields
  miniaturaUrl: '',
  nivel: null,
  categorias: [],
  cantidadClases: 0,
  horasVideo: 0,
  horasPractico: 0,
};

const MOCK_PAGE: Page<CatalogCourseCard> = {
  items: [MOCK_CARD],
  page: 1,
  size: 12,
  total: 1,
  totalPages: 1,
};

const MOCK_PREVIEW: CoursePreviewResponse = {
  enrolled: false,
  id: 'course-1',
  titulo: 'Go Avanzado',
  descripcion: 'Aprende Go de verdad',
  creadorNombre: 'Yerson Reyes',
  // course-structure-v2 metadata fields
  nivel: null,
  categorias: [],
  cantidadClases: 0,
  horasVideo: 0,
  horasPractico: 0,
  miniaturaUrl: '',
};

const MOCK_ENROLLED: CourseDetailAlumnoResponse = {
  enrolled: true,
  id: 'course-1',
  titulo: 'Go Avanzado',
  descripcion: 'Aprende Go de verdad',
  creadorNombre: 'Yerson Reyes',
  secciones: [],
  // course-structure-v2: materiales moved to per-video; course-level REMOVED
  nivel: null,
  categorias: [],
  cantidadClases: 0,
  horasVideo: 0,
  horasPractico: 0,
  miniaturaUrl: '',
};

const MOCK_ENROLL_RESP: EnrollmentResponse = {
  courseId: 'course-1',
  enrolled: true,
};

const MOCK_MY_COURSES: MyCourseItem[] = [
  {
    courseId: 'course-1',
    titulo: 'Go Avanzado',
    creadorNombre: 'Yerson Reyes',
    completado: false,
    inscritoEn: '2026-01-01T00:00:00Z',
  },
];

describe('CourseCatalogService', () => {
  let service: CourseCatalogService;
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
    service = TestBed.inject(CourseCatalogService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── getCatalog ────────────────────────────────────────────────────────────────

  it('getCatalog() sends GET /api/catalog with page and size params', async () => {
    const promise = service.getCatalog(1, 12, '');

    const req = httpMock.expectOne(r => r.url === CATALOG_BASE && r.method === 'GET');
    expect(req.request.params.get('page')).toBe('1');
    expect(req.request.params.get('size')).toBe('12');
    req.flush(MOCK_PAGE);

    const result = await promise;
    expect(result.items).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('getCatalog() omits q param when q is empty string', async () => {
    const promise = service.getCatalog(1, 12, '');

    const req = httpMock.expectOne(r => r.url === CATALOG_BASE && r.method === 'GET');
    expect(req.request.params.has('q')).toBe(false);
    req.flush(MOCK_PAGE);

    await promise;
  });

  it('getCatalog() includes q param when q is non-empty', async () => {
    const promise = service.getCatalog(1, 12, 'angular');

    const req = httpMock.expectOne(r => r.url === CATALOG_BASE && r.method === 'GET');
    expect(req.request.params.get('q')).toBe('angular');
    req.flush(MOCK_PAGE);

    await promise;
  });

  // ── getDetail ─────────────────────────────────────────────────────────────────

  it('getDetail() sends GET /api/catalog/:id', async () => {
    const promise = service.getDetail('course-1');

    const req = httpMock.expectOne(`${CATALOG_BASE}/course-1`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_PREVIEW);

    const result = await promise;
    expect(result.enrolled).toBe(false);
  });

  it('getDetail() returns enrolled response when enrolled=true', async () => {
    const promise = service.getDetail('course-1');

    const req = httpMock.expectOne(`${CATALOG_BASE}/course-1`);
    req.flush(MOCK_ENROLLED);

    const result = await promise;
    expect(result.enrolled).toBe(true);
  });

  // ── enroll ────────────────────────────────────────────────────────────────────

  it('enroll() sends POST /api/catalog/:id/enroll', async () => {
    const promise = service.enroll('course-1');

    const req = httpMock.expectOne(`${CATALOG_BASE}/course-1/enroll`);
    expect(req.request.method).toBe('POST');
    req.flush(MOCK_ENROLL_RESP);

    const result = await promise;
    expect(result.enrolled).toBe(true);
    expect(result.courseId).toBe('course-1');
  });

  // ── getMyCourses ──────────────────────────────────────────────────────────────

  it('getMyCourses() sends GET /api/users/me/courses (NOT /api/catalog/me/courses)', async () => {
    const promise = service.getMyCourses();

    const req = httpMock.expectOne(MY_COURSES_URL);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_MY_COURSES);

    const result = await promise;
    expect(result).toHaveLength(1);
    expect(result[0].courseId).toBe('course-1');
  });
});
