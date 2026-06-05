/**
 * course-detail.component.spec.ts — CourseDetailAlumnoComponent unit tests (Strict TDD — RED → GREEN).
 *
 * Strategy: spy on CourseCatalogService; call component methods directly
 * (same pattern as evaluacion-tomar.component.spec.ts — provideRouter
 * overrides ActivatedRoute snapshot so params are injected via (comp as any)).
 *
 * Covers:
 *  - Preview branch: renders titulo, "Inscribirme" button — ZERO secciones DOM nodes
 *  - Enrolled branch: renders secciones; no "Inscribirme" visible
 *  - Enroll flow: onEnroll() calls enroll() then re-fetches getDetail
 *  - getDetail is called with correct course id
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
        },
      ],
    },
  ],
  materiales: [],
};

const MOCK_ENROLL: EnrollmentResponse = {
  courseId: 'course-1',
  enrolled: true,
};

// ── Helper ────────────────────────────────────────────────────────────────────

async function createComponent(catalogSpy: Partial<CourseCatalogService>) {
  await TestBed.configureTestingModule({
    imports: [CourseDetailAlumnoComponent],
    providers: [
      { provide: CourseCatalogService, useValue: catalogSpy },
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

  // Inject courseId directly (provideRouter may override ActivatedRoute snapshot)
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
      // getDetail called again after enroll (re-fetch, not full reload)
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
});
