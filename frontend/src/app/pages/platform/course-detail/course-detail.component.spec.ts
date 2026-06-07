/**
 * course-detail.component.spec.ts — CourseDetailAlumnoComponent unit tests (Strict TDD).
 *
 * Updated in course-structure-v2:
 * - MOCK_ENROLLED uses per-video materiales[] (no course-level materiales)
 * - MOCK_PREVIEW + MOCK_ENROLLED gain metadata fields (nivel, categorias, etc.)
 * - Tests added for per-video materiales render + course-level absence
 * - downloadMaterial() calls MaterialService.downloadUrl(materialId) with materialId only
 *
 * Updated in course-player-progress:
 * - MOCK_ENROLLED videos gain completado: false (required by VideoResponseItem)
 * - MOCK_ENROLLED_WITH_PROGRESS: vid-1 complete, vid-2 incomplete (for initActiveVideo test)
 * - MOCK_ENROLLED_ALL_DONE: all videos completado=true (for all-complete fallback test)
 * - New tests: initActiveVideo selects first incomplete; selectVideo switches; markCompleted calls PUT + toggles
 * - W-1: markCompleted on already-complete video calls markVideoProgress(id, false) + decrements progreso
 * - W-2: initActiveVideo with all videos complete falls back to first video
 *
 * Covers:
 *  - Preview branch: renders titulo, "Inscribirme" button — ZERO secciones DOM nodes
 *  - Enrolled branch: renders secciones; no "Inscribirme" visible
 *  - Enroll flow: onEnroll() calls enroll() then re-fetches getDetail
 *  - getDetail is called with correct course id
 *  - Per-video materiales rendered in enrolled branch
 *  - No course-level materiales block
 *  - downloadMaterial() calls MaterialService.downloadUrl(materialId)
 *  - initActiveVideo selects first incomplete video (player-progress WU-2)
 *  - initActiveVideo falls back to first video when ALL videos are complete (W-2)
 *  - selectVideo() switches activeVideoId signal
 *  - markCompleted() calls markVideoProgress + optimistically toggles completado
 *  - markCompleted() un-marks (true→false) and decrements progreso (W-1)
 *  - progreso% updates after markCompleted
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
          completado: false,
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

/**
 * Fixture for initActiveVideo test: vid-1 complete, vid-2 incomplete.
 * initActiveVideo() must select vid-2 (first incomplete).
 */
const MOCK_ENROLLED_WITH_PROGRESS: CourseDetailAlumnoResponse = {
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
          url: 'https://youtube.com/watch?v=AAA',
          proveedor: 'youtube',
          orden: 1,
          descripcion: 'Descripcion del video 1',
          materiales: [],
          completado: true,
        },
        {
          id: 'vid-2',
          titulo: 'Video 2',
          url: 'https://youtube.com/watch?v=BBB',
          proveedor: 'youtube',
          orden: 2,
          descripcion: 'Descripcion del video 2',
          materiales: [],
          completado: false,
        },
      ],
    },
  ],
  nivel: 'avanzado',
  categorias: [{ id: 'cat-1', nombre: 'Backend' }],
  cantidadClases: 2,
  horasVideo: 1.0,
  horasPractico: 0.0,
  miniaturaUrl: '',
};

/**
 * Fixture for W-2: ALL videos are completado=true.
 * initActiveVideo() must fall back to vids[0] (vid-1) when no incomplete video exists.
 */
const MOCK_ENROLLED_ALL_DONE: CourseDetailAlumnoResponse = {
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
          url: 'https://youtube.com/watch?v=AAA',
          proveedor: 'youtube',
          orden: 1,
          descripcion: 'Descripcion del video 1',
          materiales: [],
          completado: true,
        },
        {
          id: 'vid-2',
          titulo: 'Video 2',
          url: 'https://youtube.com/watch?v=BBB',
          proveedor: 'youtube',
          orden: 2,
          descripcion: 'Descripcion del video 2',
          materiales: [],
          completado: true,
        },
      ],
    },
  ],
  nivel: 'avanzado',
  categorias: [{ id: 'cat-1', nombre: 'Backend' }],
  cantidadClases: 2,
  horasVideo: 1.0,
  horasPractico: 0.0,
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

  // ── Player state (course-player-progress WU-2) ────────────────────────────────

  describe('initActiveVideo — selects first incomplete video on load', () => {
    it('sets activeVideoId to vid-2 when vid-1 is complete and vid-2 is incomplete', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED_WITH_PROGRESS),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
        markVideoProgress: vi.fn().mockResolvedValue(undefined),
      };

      const { comp } = await createComponent(spy);
      await comp.loadDetail();

      // After loadDetail, initActiveVideo should have set activeVideoId to vid-2 (first incomplete)
      expect(comp.activeVideoId()).toBe('vid-2');
    });

    it('sets activeVideoId to vid-1 (first) when vid-1 is the only video and is incomplete', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
        markVideoProgress: vi.fn().mockResolvedValue(undefined),
      };

      const { comp } = await createComponent(spy);
      await comp.loadDetail();

      // MOCK_ENROLLED has only vid-1 (completado: false) → it is the first incomplete → selected
      expect(comp.activeVideoId()).toBe('vid-1');
    });
  });

  describe('selectVideo() — switches activeVideoId', () => {
    it('selectVideo(vid-2) sets activeVideoId to vid-2', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED_WITH_PROGRESS),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
        markVideoProgress: vi.fn().mockResolvedValue(undefined),
      };

      const { comp } = await createComponent(spy);
      await comp.loadDetail();

      comp.selectVideo('vid-2');
      expect(comp.activeVideoId()).toBe('vid-2');

      comp.selectVideo('vid-1');
      expect(comp.activeVideoId()).toBe('vid-1');
    });
  });

  describe('markCompleted() — calls PUT + optimistic toggle + progreso update', () => {
    it('calls markVideoProgress with toggled completado (false → true)', async () => {
      const markProgressSpy = vi.fn().mockResolvedValue(undefined);
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED_WITH_PROGRESS),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
        markVideoProgress: markProgressSpy,
      };

      const { comp } = await createComponent(spy);
      await comp.loadDetail();

      // vid-2 starts as completado=false; marking it should call with !false = true
      const vid2 = comp.flatVideos().find(v => v.id === 'vid-2')!;
      await comp.markCompleted(vid2);

      expect(markProgressSpy).toHaveBeenCalledWith('vid-2', true);
    });

    it('optimistically flips completado in the detail signal after markCompleted', async () => {
      const markProgressSpy = vi.fn().mockResolvedValue(undefined);
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED_WITH_PROGRESS),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
        markVideoProgress: markProgressSpy,
      };

      const { comp } = await createComponent(spy);
      await comp.loadDetail();

      const vid2 = comp.flatVideos().find(v => v.id === 'vid-2')!;
      expect(vid2.completado).toBe(false);

      await comp.markCompleted(vid2);

      const updated = comp.flatVideos().find(v => v.id === 'vid-2')!;
      expect(updated.completado).toBe(true);
    });

    it('progreso% increments after marking a video complete', async () => {
      const markProgressSpy = vi.fn().mockResolvedValue(undefined);
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED_WITH_PROGRESS),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
        markVideoProgress: markProgressSpy,
      };

      const { comp } = await createComponent(spy);
      await comp.loadDetail();

      // Initial: vid-1 done (1/2 = 50%)
      expect(comp.progreso()).toBe(50);

      const vid2 = comp.flatVideos().find(v => v.id === 'vid-2')!;
      await comp.markCompleted(vid2);

      // After marking vid-2: 2/2 = 100%
      expect(comp.progreso()).toBe(100);
    });

    it('progress bar shows X/N format correctly', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED_WITH_PROGRESS),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
        markVideoProgress: vi.fn().mockResolvedValue(undefined),
      };

      const { comp } = await createComponent(spy);
      await comp.loadDetail();

      // vid-1 complete, vid-2 incomplete → completedCount=1, total=2
      expect(comp.completedCount()).toBe(1);
      expect(comp.enrolledDetail()?.cantidadClases).toBe(2);
    });
  });

  // ── W-1: un-mark (true→false) at component level ─────────────────────────────

  describe('markCompleted() — un-mark (true→false) decrements progreso (W-1)', () => {
    it('calls markVideoProgress(videoId, false) when video is already complete (completado=true)', async () => {
      const markProgressSpy = vi.fn().mockResolvedValue(undefined);
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED_WITH_PROGRESS),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
        markVideoProgress: markProgressSpy,
      };

      const { comp } = await createComponent(spy);
      await comp.loadDetail();

      // vid-1 starts as completado=true; un-marking it must pass false to the service
      const vid1 = comp.flatVideos().find(v => v.id === 'vid-1')!;
      expect(vid1.completado).toBe(true);

      await comp.markCompleted(vid1);

      expect(markProgressSpy).toHaveBeenCalledWith('vid-1', false);
    });

    it('optimistically flips completado to false and progreso DECREMENTS after un-mark', async () => {
      const markProgressSpy = vi.fn().mockResolvedValue(undefined);
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED_WITH_PROGRESS),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
        markVideoProgress: markProgressSpy,
      };

      const { comp } = await createComponent(spy);
      await comp.loadDetail();

      // Initial state: vid-1 done, vid-2 incomplete → 1/2 = 50%
      expect(comp.progreso()).toBe(50);

      const vid1 = comp.flatVideos().find(v => v.id === 'vid-1')!;
      await comp.markCompleted(vid1);

      // After un-mark: vid-1 completado flipped to false → 0/2 = 0%
      const updated = comp.flatVideos().find(v => v.id === 'vid-1')!;
      expect(updated.completado).toBe(false);
      expect(comp.progreso()).toBe(0);
    });
  });

  // ── W-2: all-videos-completed fallback to first video ─────────────────────────

  describe('initActiveVideo() — all-complete fallback to first video (W-2)', () => {
    it('sets activeVideoId to vid-1 (first video) when ALL videos are completado=true', async () => {
      const spy: Partial<CourseCatalogService> = {
        getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED_ALL_DONE),
        enroll: vi.fn().mockResolvedValue(MOCK_ENROLL),
        markVideoProgress: vi.fn().mockResolvedValue(undefined),
      };

      const { comp } = await createComponent(spy);
      await comp.loadDetail();

      // No incomplete video → firstIncomplete is undefined → fallback to vids[0] = vid-1
      expect(comp.activeVideoId()).toBe('vid-1');
    });
  });
});
