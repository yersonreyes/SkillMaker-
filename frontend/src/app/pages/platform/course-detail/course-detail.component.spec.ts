/**
 * course-detail.component.spec.ts — CourseDetailAlumnoComponent unit tests (Strict TDD).
 *
 * Updated in course-structure-v2:
 * - MOCK_ENROLLED uses per-video materiales[] (no course-level materiales)
 * - MOCK_PREVIEW + MOCK_ENROLLED gain metadata fields (nivel, categorias, etc.)
 * - Tests added for per-video materiales render + course-level absence
 * - downloadMaterial() calls MaterialService.downloadUrl(materialId) with materialId only
 *
 * Covers:
 *  - Preview branch: renders titulo, "Inscribirme" button — ZERO secciones DOM nodes
 *  - Enrolled branch: renders secciones; no "Inscribirme" visible
 *  - Enroll flow: onEnroll() calls enroll() then re-fetches getDetail
 *  - getDetail is called with correct course id
 *  - Per-video materiales rendered in enrolled branch
 *  - No course-level materiales block
 *  - downloadMaterial() calls MaterialService.downloadUrl(materialId)
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, afterEach, vi } from 'vitest';
import {
  provideRouter,
  ActivatedRoute,
  convertToParamMap,
} from '@angular/router';
import { of } from 'rxjs';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { CourseDetailAlumnoComponent } from './course-detail.component';
import { CourseCatalogService } from '@core/services/courseCatalogService/course-catalog.service';
import { CertificateService } from '@core/services/certificateService/certificate.service';
import { MaterialService } from '@core/services/materialService/material.service';
import type {
  CoursePreviewResponse,
  CourseDetailAlumnoResponse,
  EnrollmentResponse,
} from '@core/services/courseCatalogService/course-catalog.dto';

// ── Fixtures ──────────────────────────────────────────────────────────────────

const MOCK_PREVIEW: CoursePreviewResponse = {
  enrolled: false,
  id: 'course-1',
  titulo: 'Go Avanzado',
  descripcion: 'Aprende Go de verdad',
  creadorNombre: 'Yerson Reyes',
  nivel: 'intermedio',
  categorias: [{ id: 'cat-1', nombre: 'Backend' }],
  cantidadClases: 5,
  horasVideo: 2.5,
  horasPractico: 1.0,
  miniaturaUrl: '',
};

const MOCK_ENROLLED: CourseDetailAlumnoResponse = {
  enrolled: true,
  id: 'course-1',
  titulo: 'Go Avanzado',
  descripcion: 'Aprende Go de verdad',
  creadorNombre: 'Yerson Reyes',
  secciones: [
    {
      id: 'sec-1',
      titulo: 'Introduccion',
      orden: 1,
      videos: [
        {
          id: 'vid-1',
          titulo: 'Video 1',
          url: 'https://www.youtube.com/watch?v=dQw4w9WgXcQ',
          proveedor: 'youtube',
          orden: 1,
          descripcion: 'Descripcion del video 1',
          materiales: [
            {
              id: 'mat-1',
              nombre: 'slides.pdf',
              mimeType: 'application/pdf',
              tamanoBytes: 5_000_000,
              createdAt: '2026-06-01T00:00:00Z',
            },
          ],
        },
      ],
    },
  ],
  nivel: 'avanzado',
  categorias: [{ id: 'cat-1', nombre: 'Backend' }],
  cantidadClases: 1,
  horasVideo: 1.0,
  horasPractico: 2.0,
  miniaturaUrl: '',
};

const MOCK_ENROLL: EnrollmentResponse = {
  courseId: 'course-1',
  enrolled: true,
};

// ── Helper ────────────────────────────────────────────────────────────────────

/** Minimal CertificateService stub — returns empty list. */
const noCertSpy: Partial<CertificateService> = {
  getMyCertificates: vi.fn().mockResolvedValue([]),
  getDownloadUrl: vi.fn().mockResolvedValue({ url: '', expiresAt: '' }),
};

/** Minimal MaterialService stub. */
const materialServiceSpy: Partial<MaterialService> = {
  downloadUrl: vi.fn().mockResolvedValue({ url: 'http://minio/dl', expiresAt: '2026-06-04' }),
};

async function createComponent(catalogSpy: Partial<CourseCatalogService>) {
  await TestBed.configureTestingModule({
    imports: [CourseDetailAlumnoComponent],
    providers: [
      { provide: CourseCatalogService, useValue: catalogSpy },
      { provide: CertificateService, useValue: noCertSpy },
      { provide: MaterialService, useValue: materialServiceSpy },
      {
        provide: ActivatedRoute,
        useValue: {
          snapshot: { paramMap: convertToParamMap({ id: 'course-1' }) },
          params: of({ id: 'course-1' }),
        },
      },
      provideRouter([{ path: '**', component: CourseDetailAlumnoComponent }]),
      provideAnimationsAsync(),
      ConfirmationService,
      MessageService,
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(CourseDetailAlumnoComponent);
  const comp = fixture.componentInstance;

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (comp as any)['courseId'] = 'course-1';

  return { fixture, comp };
}

// ── Specs ─────────────────────────────────────────────────────────────────────

describe('CourseDetailAlumnoComponent', () => {
  afterEach(() => {
    TestBed.resetTestingModule();
  });

  describe('Preview branch (enrolled=false)', () => {
    it('renders titulo from preview response', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_PREVIEW),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
      };

      const { fixture, comp } = await createComponent(spy);

      await comp.loadDetail();
      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();

      const el: HTMLElement = fixture.nativeElement;
      expect(el.textContent).toContain('Go Avanzado');
    });

    it('renders "Inscribirme" button in preview branch', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_PREVIEW),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
      };

      const { fixture, comp } = await createComponent(spy);
      await comp.loadDetail();
      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();

      const el: HTMLElement = fixture.nativeElement;
      const enrollBtn = Array.from(el.querySelectorAll('button')).find(b =>
        b.textContent?.trim().includes('Inscribirme'),
      );
      expect(enrollBtn).toBeDefined();
    });

    it('renders ZERO .course-detail__seccion DOM nodes in preview branch (structural absence)', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_PREVIEW),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
      };

      const { fixture, comp } = await createComponent(spy);
      await comp.loadDetail();
      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();

      const el: HTMLElement = fixture.nativeElement;
      const secciones = el.querySelectorAll('.course-detail__seccion');
      expect(secciones.length).toBe(0);
    });
  });

  describe('Enrolled branch (enrolled=true)', () => {
    it('renders secciones in enrolled branch', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
      };

      const { fixture, comp } = await createComponent(spy);
      await comp.loadDetail();
      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();

      const el: HTMLElement = fixture.nativeElement;
      const secciones = el.querySelectorAll('.course-detail__seccion');
      expect(secciones.length).toBe(1);
    });

    it('does NOT render "Inscribirme" button in enrolled branch', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
      };

      const { fixture, comp } = await createComponent(spy);
      await comp.loadDetail();
      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();

      const el: HTMLElement = fixture.nativeElement;
      const enrollBtn = Array.from(el.querySelectorAll('button')).find(b =>
        b.textContent?.trim().includes('Inscribirme'),
      );
      expect(enrollBtn).toBeUndefined();
    });

    it('renders per-video materiales in enrolled branch (course-structure-v2)', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
      };

      const { fixture, comp } = await createComponent(spy);
      await comp.loadDetail();
      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();

      const el: HTMLElement = fixture.nativeElement;
      // Per-video material item should be present
      const matItems = el.querySelectorAll('.course-detail__material-item');
      expect(matItems.length).toBe(1);
      expect(matItems[0].textContent).toContain('slides.pdf');
    });

    it('renders video.descripcion when present', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
      };

      const { fixture, comp } = await createComponent(spy);
      await comp.loadDetail();
      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();

      const el: HTMLElement = fixture.nativeElement;
      expect(el.textContent).toContain('Descripcion del video 1');
    });

    it('does NOT render course-level materiales block (removed in course-structure-v2)', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
      };

      const { fixture, comp } = await createComponent(spy);
      await comp.loadDetail();
      fixture.detectChanges();
      await fixture.whenStable();
      fixture.detectChanges();

      const el: HTMLElement = fixture.nativeElement;
      // The old .course-detail__materiales section at course level should not exist
      const courseMatSection = el.querySelector('.course-detail__materiales');
      expect(courseMatSection).toBeNull();
    });
  });

  describe('Enroll flow', () => {
    it('onEnroll() calls enroll() then re-fetches getDetail', async () => {
      const getDetailFn = vi.fn().mockResolvedValue(MOCK_PREVIEW);
      const enrollFn = vi.fn().mockResolvedValue(MOCK_ENROLL);

      const spy: Partial<CourseCatalogService> = {
        getDetail: getDetailFn,
        enroll: enrollFn,
      };

      const { comp } = await createComponent(spy);
      await comp.loadDetail();
      expect(getDetailFn).toHaveBeenCalledTimes(1);

      await comp.onEnroll();

      expect(enrollFn).toHaveBeenCalledWith('course-1');
      expect(getDetailFn).toHaveBeenCalledTimes(2);
    });
  });

  describe('Course id routing', () => {
    it('getDetail is called with the injected course id', async () => {
      const getDetailFn = vi.fn().mockResolvedValue(MOCK_PREVIEW);
      const spy: Partial<CourseCatalogService> = {
        getDetail: getDetailFn,
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
      };

      const { comp } = await createComponent(spy);
      await comp.loadDetail();

      expect(getDetailFn).toHaveBeenCalledWith('course-1');
    });
  });

  describe('Material download', () => {
    it('downloadMaterial() calls MaterialService.downloadUrl with materialId only (re-keyed)', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
      };

      const { comp } = await createComponent(spy);

      await comp.downloadMaterial('mat-1');

      expect(materialServiceSpy.downloadUrl).toHaveBeenCalledWith('mat-1');
    });
  });
});
