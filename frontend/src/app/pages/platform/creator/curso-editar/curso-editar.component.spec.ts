/**
 * curso-editar.component.spec.ts — Component unit tests (Vitest + Angular TestBed).
 *
 * Covers:
 *  - Component load → getById called → form populated (existing C2.1 tests preserved)
 *  - Save → update called with form values (C2.1)
 *  - "Enviar a revisión" button is disabled (C2.2: wired to !hasContent || true per D6)
 *  - FE-1-A: addSection() calls SectionService.create
 *  - FE-1-B: addVideo() calls VideoService.create
 *  - FE-1-D: deleteSection() shows confirm dialog then calls SectionService.delete
 *  - FE-2-A/B: submitDisabled is always true regardless of hasContent (D6)
 *  - CRITICAL (load-existing-content): loadSections() populates sections with nested videos
 *  - WARNING-2 (FE-1-C): onSectionsReorder() calls SectionService.reorder with courseId + ids
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter, ActivatedRoute, convertToParamMap } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { of } from 'rxjs';

import { CursoEditarComponent } from './curso-editar.component';
import { CourseService } from '@core/services/courseService/course.service';
import { SectionService } from '@core/services/sectionService/section.service';
import { VideoService } from '@core/services/videoService/video.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { CourseDetail } from '@core/services/courseService/course.res.dto';
import type { SectionItem, SectionWithVideos } from '@core/services/sectionService/section.res.dto';
import type { VideoItem } from '@core/services/videoService/video.res.dto';

// ── Helpers ───────────────────────────────────────────────────────────────────

const MOCK_COURSE_DETAIL: CourseDetail = {
  id: 'c-1',
  creadorId: 'user-1',
  titulo: 'Go avanzado',
  descripcion: 'Curso de Go para profesionales',
  estado: 'borrador',
  hasContent: false,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

const MOCK_SECTION: SectionItem = {
  id: 'sec-1',
  courseId: 'c-1',
  titulo: 'Introduccion',
  orden: 0,
  createdAt: '2026-01-01T00:00:00Z',
};

const MOCK_VIDEO: VideoItem = {
  id: 'vid-1',
  sectionId: 'sec-1',
  titulo: 'Intro a Go',
  url: 'https://www.youtube.com/watch?v=abc123',
  proveedor: 'youtube',
  duracionS: 300,
  orden: 0,
  createdAt: '2026-01-01T00:00:00Z',
};

// ── Spec ──────────────────────────────────────────────────────────────────────

describe('CursoEditarComponent', () => {
  let courseServiceSpy: Partial<CourseService>;
  let sectionServiceSpy: Partial<SectionService>;
  let videoServiceSpy: Partial<VideoService>;
  let uiDialogSpy: Partial<UiDialogService>;

  beforeEach(async () => {
    courseServiceSpy = {
      getById: vi.fn().mockResolvedValue(MOCK_COURSE_DETAIL),
      update: vi.fn().mockResolvedValue({ ...MOCK_COURSE_DETAIL, titulo: 'Actualizado' }),
    };

    sectionServiceSpy = {
      listByCourse: vi.fn().mockResolvedValue([{ ...MOCK_SECTION, videos: [] }]),
      create: vi.fn().mockResolvedValue(MOCK_SECTION),
      update: vi.fn().mockResolvedValue(MOCK_SECTION),
      delete: vi.fn().mockResolvedValue(undefined),
      reorder: vi.fn().mockResolvedValue([{ ...MOCK_SECTION, videos: [] }]),
    };

    videoServiceSpy = {
      create: vi.fn().mockResolvedValue(MOCK_VIDEO),
      update: vi.fn().mockResolvedValue(MOCK_VIDEO),
      delete: vi.fn().mockResolvedValue(undefined),
    };

    uiDialogSpy = {
      showSuccess: vi.fn(),
      showError: vi.fn(),
      confirmDelete: vi.fn().mockResolvedValue(true),
      confirm: vi.fn().mockResolvedValue(true),
    };

    await TestBed.configureTestingModule({
      imports: [CursoEditarComponent],
      providers: [
        { provide: CourseService, useValue: courseServiceSpy },
        { provide: SectionService, useValue: sectionServiceSpy },
        { provide: VideoService, useValue: videoServiceSpy },
        { provide: UiDialogService, useValue: uiDialogSpy },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: { paramMap: convertToParamMap({ id: 'c-1' }) },
            params: of({ id: 'c-1' }),
          },
        },
        ConfirmationService,
        MessageService,
        provideRouter([{ path: '**', component: CursoEditarComponent }]),
        provideAnimationsAsync(),
      ],
    }).compileComponents();
  });

  afterEach(() => {
    TestBed.resetTestingModule();
  });

  // ── F1-load: component loads course via loadCourse() ──────────────────────

  it('loadCourse() calls getById and populates titulo and descripcion', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    expect(courseServiceSpy.getById).toHaveBeenCalledWith('c-1');
    expect(comp.titulo()).toBe('Go avanzado');
    expect(comp.descripcion()).toBe('Curso de Go para profesionales');
  });

  it('loadCourse() also loads sections via SectionService.listByCourse', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    expect(sectionServiceSpy.listByCourse).toHaveBeenCalledWith('c-1');
    expect(comp.sections()).toHaveLength(1);
  });

  // ── F2-save: onSave calls update with form values ─────────────────────────

  it('onSave() calls CourseService.update with current form values', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    comp.titulo.set('Titulo actualizado');
    comp.descripcion.set('Nueva descripcion');

    await comp.onSave();

    expect(courseServiceSpy.update).toHaveBeenCalledWith('c-1', {
      titulo: 'Titulo actualizado',
      descripcion: 'Nueva descripcion',
    });
  });

  it('onSave() shows success toast on success', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    comp.titulo.set('Titulo');

    await comp.onSave();

    expect(uiDialogSpy.showSuccess).toHaveBeenCalled();
  });

  it('onSave() does nothing when courseId is empty', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;
    // courseId defaults to ''

    await comp.onSave();

    expect(courseServiceSpy.update).not.toHaveBeenCalled();
  });

  // ── FE-1-A: addSection calls SectionService.create ────────────────────────

  it('addSection() calls SectionService.create with titulo and adds section to list', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    comp.newSectionTitulo.set('Modulo 1');

    await comp.addSection();

    expect(sectionServiceSpy.create).toHaveBeenCalledWith('c-1', { titulo: 'Modulo 1' });
    const sections = comp.sections();
    expect(sections.some(s => s['id'] === 'sec-1')).toBe(true);
  });

  it('addSection() does nothing when titulo is empty', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    comp.newSectionTitulo.set('   ');

    await comp.addSection();

    expect(sectionServiceSpy.create).not.toHaveBeenCalled();
  });

  // ── FE-1-B: addVideo calls VideoService.create ────────────────────────────

  it('addVideo() calls VideoService.create with all required fields', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    comp.sections.set([{ ...MOCK_SECTION, videos: [] }]);
    comp.videoForm.set({
      titulo: 'Clase 1',
      url: 'https://www.youtube.com/watch?v=abc',
      proveedor: 'youtube',
      duracionS: undefined,
    });

    await comp.addVideo('sec-1');

    expect(videoServiceSpy.create).toHaveBeenCalledWith('sec-1', {
      titulo: 'Clase 1',
      url: 'https://www.youtube.com/watch?v=abc',
      proveedor: 'youtube',
    });
  });

  // ── FE-1-D: deleteSection calls SectionService.delete after confirm ────────

  it('deleteSection() shows confirm dialog then calls SectionService.delete', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    comp.sections.set([{ ...MOCK_SECTION, videos: [] }]);

    await comp.deleteSection('sec-1');

    expect(uiDialogSpy.confirmDelete).toHaveBeenCalled();
    expect(sectionServiceSpy.delete).toHaveBeenCalledWith('sec-1');
  });

  it('deleteSection() does NOT call delete when confirm is rejected', async () => {
    uiDialogSpy.confirmDelete = vi.fn().mockResolvedValue(false);

    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    comp.sections.set([{ ...MOCK_SECTION, videos: [] }]);

    await comp.deleteSection('sec-1');

    expect(sectionServiceSpy.delete).not.toHaveBeenCalled();
  });

  // ── FE-2-A/B: submitDisabled always true (D6) ─────────────────────────────

  it('FE-2-A: submitDisabled is true when hasContent is false', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    expect(comp.hasContent()).toBe(false);
    expect(comp.submitDisabled()).toBe(true);
  });

  it('FE-2-B: submitDisabled remains true even when hasContent is true (C4.1 pending)', async () => {
    courseServiceSpy.getById = vi.fn().mockResolvedValue({
      ...MOCK_COURSE_DETAIL,
      hasContent: true,
    });

    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    expect(comp.hasContent()).toBe(true);
    // D6: always disabled until C4.1
    expect(comp.submitDisabled()).toBe(true);
  });

  // ── F3-enviar-disabled: legacy test name preserved ────────────────────────

  it('submitDisabled is always true (C2.2 — wired as !hasContent || true per D6)', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();

    expect(comp.submitDisabled()).toBe(true);
  });

  // ── CRITICAL: load-existing-content regression ────────────────────────────
  // Regression test for the missing read path (verify obs #263).
  // Before the fix: loadSections() mapped `items.map(s => ({ ...s, videos: [] }))`
  // — this wiped any videos returned by the API, making existing sections/videos invisible on reload.
  // After the fix: videos from the API response are used directly (s.videos ?? []).

  it('CRITICAL: loadSections() preserves nested videos returned by the API (not videos: [])', async () => {
    // Arrange: mock returns 1 section with 2 nested videos.
    const sectionWithVideos: SectionWithVideos = {
      ...MOCK_SECTION,
      videos: [MOCK_VIDEO, { ...MOCK_VIDEO, id: 'vid-2', titulo: 'Clase 2', orden: 1 }],
    };
    sectionServiceSpy.listByCourse = vi.fn().mockResolvedValue([sectionWithVideos]);

    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadSections();

    // Assert: sections signal must have the section WITH its 2 videos (not videos: []).
    const sections = comp.sections();
    expect(sections).toHaveLength(1);
    expect(sections[0].videos).toHaveLength(2);
    expect(sections[0].videos[0].id).toBe('vid-1');
    expect(sections[0].videos[1].id).toBe('vid-2');
  });

  // ── WARNING-2: FE-1-C — onSectionsReorder calls SectionService.reorder ────
  // Asserts SectionService.reorder is called with the courseId and the reordered ids array.
  // Spec: FE-1-C — drag-to-reorder → SectionService.reorder called with updated ids.

  it('FE-1-C: onSectionsReorder() calls SectionService.reorder with courseId and reordered ids', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';

    const sec1: SectionWithVideos = { ...MOCK_SECTION, id: 'sec-1', orden: 0, videos: [] };
    const sec2: SectionWithVideos = {
      id: 'sec-2',
      courseId: 'c-1',
      titulo: 'Modulo 2',
      orden: 1,
      createdAt: '2026-01-01T00:00:00Z',
      videos: [],
    };
    // Simulate: user drags sec2 to first position → newOrder is [sec2, sec1].
    const newOrder = [sec2, sec1];

    await comp.onSectionsReorder(newOrder);

    expect(sectionServiceSpy.reorder).toHaveBeenCalledWith('c-1', ['sec-2', 'sec-1']);
  });
});
