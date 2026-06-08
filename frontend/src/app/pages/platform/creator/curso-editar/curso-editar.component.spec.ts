/**
 * curso-editar.component.spec.ts — Component unit tests (Vitest + Angular TestBed).
 *
 * Updated in course-structure-v2:
 * - Material service re-keyed to videoId (per-video materials)
 * - Course service gains nivel, categoriaIds, horasPractico
 * - CategoriaService added
 * - Video form gains descripcion
 * - Thumbnail upload added
 *
 * Covers:
 *  - Component load → getById called → form populated
 *  - Save → update called with form values (includes nivel, horasPractico, categoriaIds)
 *  - FE-1-A: addSection() calls SectionService.create
 *  - FE-1-B: addVideo() calls VideoService.create (with descripcion)
 *  - FE-1-D: deleteSection() shows confirm dialog then calls SectionService.delete
 *  - FE-2 (C4.1): submitDisabled reflects estado + hasContent
 *  - FE-2-submit: onSubmitToReview() calls ApprovalService.submitToReview then reloads course
 *  - CRITICAL (load-existing-content): loadSections() populates sections with nested videos
 *  - WARNING-2 (FE-1-C): onSectionsReorder() calls SectionService.reorder with courseId + ids
 *  - Per-video material: onVideoMaterialUploaded, deleteVideoMaterial, downloadVideoMaterial, loadAllVideoMaterials
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
import { MaterialService } from '@core/services/materialService/material.service';
import { CategoriaService } from '@core/services/categoriaService/categoria.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import { ApprovalService } from '@core/services/approvalService/approval.service';
import type { CourseDetail } from '@core/services/courseService/course.res.dto';
import type { SectionItem, SectionWithVideos } from '@core/services/sectionService/section.res.dto';
import type { VideoItem } from '@core/services/videoService/video.res.dto';
import type { MaterialResponse } from '@core/services/materialService/material.types';

// ── Helpers ───────────────────────────────────────────────────────────────────

const MOCK_COURSE_DETAIL: CourseDetail = {
  id: 'c-1',
  creadorId: 'user-1',
  titulo: 'Go avanzado',
  descripcion: 'Curso de Go para profesionales',
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
  descripcion: '',
  materiales: [],
  createdAt: '2026-01-01T00:00:00Z',
};

const MOCK_MATERIAL: MaterialResponse = {
  id: 'mat-1',
  nombre: 'slides.pdf',
  mimeType: 'application/pdf',
  tamanoBytes: 5_000_000,
  createdAt: '2026-06-03T15:00:00Z',
};

// ── Spec ──────────────────────────────────────────────────────────────────────

describe('CursoEditarComponent', () => {
  let courseServiceSpy: Partial<CourseService>;
  let sectionServiceSpy: Partial<SectionService>;
  let videoServiceSpy: Partial<VideoService>;
  let materialServiceSpy: Partial<MaterialService>;
  let categoriaServiceSpy: Partial<CategoriaService>;
  let uiDialogSpy: Partial<UiDialogService>;
  let approvalServiceSpy: Partial<ApprovalService>;

  beforeEach(async () => {
    courseServiceSpy = {
      getById: vi.fn().mockResolvedValue(MOCK_COURSE_DETAIL),
      update: vi.fn().mockResolvedValue({ ...MOCK_COURSE_DETAIL, titulo: 'Actualizado' }),
      presignThumbnail: vi.fn().mockResolvedValue({ uploadUrl: 'http://minio/thumb', key: 'thumb-key', expiresAt: '2026-06-04T00:00:00Z' }),
      confirmThumbnail: vi.fn().mockResolvedValue(undefined),
    };

    approvalServiceSpy = {
      submitToReview: vi.fn().mockResolvedValue({ courseId: 'c-1', estado: 'en_revision' }),
      history: vi.fn().mockResolvedValue([]),
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
      reorder: vi.fn().mockResolvedValue(undefined),
    };

    materialServiceSpy = {
      // Re-keyed to videoId (per-video materials)
      list: vi.fn().mockResolvedValue([]),
      downloadUrl: vi.fn().mockResolvedValue({ url: 'http://minio/download', expiresAt: '2026-06-04T00:00:00Z' }),
      remove: vi.fn().mockResolvedValue(undefined),
      presign: vi.fn().mockResolvedValue({ uploadUrl: 'http://minio/put', key: 'k', expiresAt: '2026-06-04T00:00:00Z' }),
      confirm: vi.fn().mockResolvedValue(MOCK_MATERIAL),
      uploadToStorage: vi.fn().mockResolvedValue(undefined),
    };

    categoriaServiceSpy = {
      getAll: vi.fn().mockResolvedValue([
        { id: 'cat-1', nombre: 'Frontend', slug: 'frontend' },
        { id: 'cat-2', nombre: 'Backend', slug: 'backend' },
      ]),
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
        { provide: MaterialService, useValue: materialServiceSpy },
        { provide: CategoriaService, useValue: categoriaServiceSpy },
        { provide: UiDialogService, useValue: uiDialogSpy },
        { provide: ApprovalService, useValue: approvalServiceSpy },
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

  it('loadCourse() populates nivel, horasPractico from course detail', async () => {
    courseServiceSpy.getById = vi.fn().mockResolvedValue({
      ...MOCK_COURSE_DETAIL,
      nivel: 'intermedio',
      horasPractico: 4.5,
    });

    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    expect(comp.nivel()).toBe('intermedio');
    expect(comp.horasPractico()).toBe(4.5);
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

  it('loadCategorias() calls CategoriaService.getAll and populates categorias', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    await comp.loadCategorias();

    expect(categoriaServiceSpy.getAll).toHaveBeenCalled();
    expect(comp.categorias()).toHaveLength(2);
    expect(comp.categorias()[0].id).toBe('cat-1');
  });

  // ── F2-save: onSave calls update with form values ─────────────────────────

  it('onSave() calls CourseService.update with current form values including nivel and categoriaIds', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    comp.titulo.set('Titulo actualizado');
    comp.descripcion.set('Nueva descripcion');
    comp.nivel.set('avanzado');
    comp.horasPractico.set(3.0);
    comp.selectedCategoriaIds.set(['cat-1', 'cat-2']);

    await comp.onSave();

    expect(courseServiceSpy.update).toHaveBeenCalledWith('c-1', {
      titulo: 'Titulo actualizado',
      descripcion: 'Nueva descripcion',
      nivel: 'avanzado',
      horasPractico: 3.0,
      categoriaIds: ['cat-1', 'cat-2'],
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

  // ── FE-1-B: saveVideo (create mode) calls VideoService.create ─────────────

  it('saveVideo() calls VideoService.create with all required fields (create mode)', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    comp.sections.set([{ ...MOCK_SECTION, videos: [] }]);
    comp.editingVideoId.set(null);
    comp.videoForm.set({
      titulo: 'Clase 1',
      url: 'https://www.youtube.com/watch?v=abc',
      proveedor: 'youtube',
      duracionS: undefined,
      descripcion: '',
    });

    await comp.saveVideo('sec-1');

    expect(videoServiceSpy.create).toHaveBeenCalledWith('sec-1', {
      titulo: 'Clase 1',
      url: 'https://www.youtube.com/watch?v=abc',
      proveedor: 'youtube',
    });
  });

  it('saveVideo() includes descripcion when provided (create mode)', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    comp.sections.set([{ ...MOCK_SECTION, videos: [] }]);
    comp.editingVideoId.set(null);
    comp.videoForm.set({
      titulo: 'Clase 1',
      url: 'https://www.youtube.com/watch?v=abc',
      proveedor: 'youtube',
      duracionS: undefined,
      descripcion: 'Intro a conceptos de Go',
    });

    await comp.saveVideo('sec-1');

    expect(videoServiceSpy.create).toHaveBeenCalledWith('sec-1', expect.objectContaining({
      descripcion: 'Intro a conceptos de Go',
    }));
  });

  // ── FE-1-B2: saveVideo (edit mode) calls VideoService.update ──────────────

  it('saveVideo() calls VideoService.update and replaces the video when editing', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    comp.sections.set([{ ...MOCK_SECTION, videos: [MOCK_VIDEO] }]);

    const updated = { ...MOCK_VIDEO, titulo: 'Intro a Go (editado)' };
    (videoServiceSpy.update as ReturnType<typeof vi.fn>).mockResolvedValueOnce(updated);

    comp.openEditVideoDialog('sec-1', MOCK_VIDEO);
    expect(comp.editingVideoId()).toBe('vid-1');
    expect(comp.videoForm().titulo).toBe('Intro a Go');

    comp.videoForm.update(f => ({ ...f, titulo: 'Intro a Go (editado)' }));
    await comp.saveVideo('sec-1');

    expect(videoServiceSpy.update).toHaveBeenCalledWith('vid-1', expect.objectContaining({
      titulo: 'Intro a Go (editado)',
    }));
    expect(videoServiceSpy.create).not.toHaveBeenCalled();
    expect(comp.sections()[0].videos[0].titulo).toBe('Intro a Go (editado)');
  });

  // ── moveVideo reorders within a section and persists via VideoService ─────

  it('moveVideo() swaps positions and calls VideoService.reorder with the new id order', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    const vA = { ...MOCK_VIDEO, id: 'vid-a', orden: 0 };
    const vB = { ...MOCK_VIDEO, id: 'vid-b', orden: 1 };
    const vC = { ...MOCK_VIDEO, id: 'vid-c', orden: 2 };
    comp.sections.set([{ ...MOCK_SECTION, videos: [vA, vB, vC] }]);

    // Move vA (index 0) down → expect order [B, A, C].
    await comp.moveVideo('sec-1', 0, 1);

    expect(videoServiceSpy.reorder).toHaveBeenCalledWith('sec-1', ['vid-b', 'vid-a', 'vid-c']);
    expect(comp.sections()[0].videos.map(v => v.id)).toEqual(['vid-b', 'vid-a', 'vid-c']);
  });

  it('moveVideo() is a no-op at the boundaries (first up / last down)', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    const vA = { ...MOCK_VIDEO, id: 'vid-a' };
    const vB = { ...MOCK_VIDEO, id: 'vid-b' };
    comp.sections.set([{ ...MOCK_SECTION, videos: [vA, vB] }]);

    await comp.moveVideo('sec-1', 0, -1); // first up → no-op
    await comp.moveVideo('sec-1', 1, 1);  // last down → no-op

    expect(videoServiceSpy.reorder).not.toHaveBeenCalled();
    expect(comp.sections()[0].videos.map(v => v.id)).toEqual(['vid-a', 'vid-b']);
  });

  // ── isCourseEditable mirrors backend estado gate ──────────────────────────

  it('isCourseEditable is true for borrador/rechazado, false for en_revision/aprobado', () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    comp.course.set({ ...MOCK_COURSE_DETAIL, estado: 'borrador' });
    expect(comp.isCourseEditable()).toBe(true);
    comp.course.set({ ...MOCK_COURSE_DETAIL, estado: 'rechazado' });
    expect(comp.isCourseEditable()).toBe(true);
    comp.course.set({ ...MOCK_COURSE_DETAIL, estado: 'en_revision' });
    expect(comp.isCourseEditable()).toBe(false);
    comp.course.set({ ...MOCK_COURSE_DETAIL, estado: 'aprobado' });
    expect(comp.isCourseEditable()).toBe(false);
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

  // ── FE-2 (C4.1): submitDisabled reflects estado + hasContent ─────────────────

  it('FE-2-A: submitDisabled is true when hasContent is false (no content)', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    expect(comp.hasContent()).toBe(false);
    expect(comp.submitDisabled()).toBe(true);
  });

  it('FE-2-B: submitDisabled is false (enabled) when estado=borrador AND hasContent=true', async () => {
    courseServiceSpy.getById = vi.fn().mockResolvedValue({
      ...MOCK_COURSE_DETAIL,
      estado: 'borrador',
      hasContent: true,
    });

    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    expect(comp.hasContent()).toBe(true);
    expect(comp.submitDisabled()).toBe(false);
  });

  it('FE-2-C: submitDisabled is false (enabled) when estado=rechazado AND hasContent=true', async () => {
    courseServiceSpy.getById = vi.fn().mockResolvedValue({
      ...MOCK_COURSE_DETAIL,
      estado: 'rechazado',
      hasContent: true,
    });

    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    expect(comp.hasContent()).toBe(true);
    expect(comp.submitDisabled()).toBe(false);
  });

  it('FE-2-D: submitDisabled is true when estado=en_revision (already in review)', async () => {
    courseServiceSpy.getById = vi.fn().mockResolvedValue({
      ...MOCK_COURSE_DETAIL,
      estado: 'en_revision',
      hasContent: true,
    });

    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    expect(comp.submitDisabled()).toBe(true);
  });

  it('FE-2-E: submitDisabled is true when estado=aprobado', async () => {
    courseServiceSpy.getById = vi.fn().mockResolvedValue({
      ...MOCK_COURSE_DETAIL,
      estado: 'aprobado',
      hasContent: true,
    });

    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    expect(comp.submitDisabled()).toBe(true);
  });

  // ── FE-2-submit: onSubmitToReview wiring ──────────────────────────────────

  it('FE-2-submit: onSubmitToReview() calls ApprovalService.submitToReview with courseId', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.onSubmitToReview();

    expect(approvalServiceSpy.submitToReview).toHaveBeenCalledWith('c-1');
  });

  it('FE-2-submit: onSubmitToReview() reloads the course on success', async () => {
    courseServiceSpy.getById = vi.fn().mockResolvedValue({
      ...MOCK_COURSE_DETAIL,
      estado: 'en_revision',
      hasContent: true,
    });

    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.onSubmitToReview();

    expect(courseServiceSpy.getById).toHaveBeenCalledWith('c-1');
    expect(comp.submitDisabled()).toBe(true);
  });

  // ── CRITICAL: load-existing-content regression ────────────────────────────

  it('CRITICAL: loadSections() preserves nested videos returned by the API (not videos: [])', async () => {
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

    const sections = comp.sections();
    expect(sections).toHaveLength(1);
    expect(sections[0].videos).toHaveLength(2);
    expect(sections[0].videos[0].id).toBe('vid-1');
    expect(sections[0].videos[1].id).toBe('vid-2');
  });

  // ── WARNING-2: FE-1-C — onSectionsReorder calls SectionService.reorder ────

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
    const newOrder = [sec2, sec1];

    await comp.onSectionsReorder(newOrder);

    expect(sectionServiceSpy.reorder).toHaveBeenCalledWith('c-1', ['sec-2', 'sec-1']);
  });

  // ── Per-video material tests ─────────────────────────────────────────────────

  it('MAT-PV-1: onVideoMaterialUploaded() appends material to the correct videoId entry', () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    comp.videoMaterials.set({ 'vid-1': [] });
    comp.onVideoMaterialUploaded('vid-1', MOCK_MATERIAL);

    expect(comp.getMaterialsForVideo('vid-1')).toHaveLength(1);
    expect(comp.getMaterialsForVideo('vid-1')[0].id).toBe('mat-1');
  });

  it('MAT-PV-2: deleteVideoMaterial() calls confirmDelete then MaterialService.remove', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    comp.videoMaterials.set({ 'vid-1': [MOCK_MATERIAL] });

    await comp.deleteVideoMaterial('vid-1', MOCK_MATERIAL);

    expect(uiDialogSpy.confirmDelete).toHaveBeenCalled();
    expect(materialServiceSpy.remove).toHaveBeenCalledWith('mat-1');
    expect(comp.getMaterialsForVideo('vid-1')).toHaveLength(0);
  });

  it('MAT-PV-3: downloadVideoMaterial() calls MaterialService.downloadUrl with materialId only', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    await comp.downloadVideoMaterial(MOCK_MATERIAL);

    // Re-keyed: only materialId, no courseId
    expect(materialServiceSpy.downloadUrl).toHaveBeenCalledWith('mat-1');
  });

  it('MAT-PV-4: loadAllVideoMaterials() calls MaterialService.list for each video', async () => {
    const sectionWithVideos: SectionWithVideos = {
      ...MOCK_SECTION,
      videos: [MOCK_VIDEO, { ...MOCK_VIDEO, id: 'vid-2', orden: 1 }],
    };

    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    await comp.loadAllVideoMaterials([sectionWithVideos]);

    expect(materialServiceSpy.list).toHaveBeenCalledWith('vid-1');
    expect(materialServiceSpy.list).toHaveBeenCalledWith('vid-2');
  });

  // ── Thumbnail ──────────────────────────────────────────────────────────────

  it('onThumbnailUploaded() sets miniaturaKey and shows success toast', () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    comp.onThumbnailUploaded('courses/c-1/thumbnail/uuid-cover.jpg');

    expect(comp.miniaturaKey()).toBe('courses/c-1/thumbnail/uuid-cover.jpg');
    expect(uiDialogSpy.showSuccess).toHaveBeenCalled();
  });

  it('MAT-7: loadCourse() loads sections via Promise.all', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    expect(sectionServiceSpy.listByCourse).toHaveBeenCalledWith('c-1');
  });

  // ── REQ-VALIDATION: inline validation for curso-editar ────────────────────

  it('REQ-VALIDATION: titulo .field__error is NOT visible when untouched', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    // No error visible when untouched (even though titulo might be empty initially)
    expect(comp.tituloTouched()).toBe(false);
    expect(comp.tituloInvalid()).toBe(false);
  });

  it('REQ-VALIDATION: titulo .field__error appears when touched with empty value', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    // Load course — titulo will be 'Go avanzado' initially
    await comp.loadCourse();

    // Now clear titulo and touch
    comp.titulo.set('');
    comp.tituloTouched.set(true);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(comp.tituloInvalid()).toBe(true);
    const el: HTMLElement = fixture.nativeElement;
    const error = el.querySelector('.field__error');
    expect(error).not.toBeNull();
  });

  it('REQ-VALIDATION: titulo .field__error is hidden when titulo has valid value and touched', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    comp.titulo.set('Go avanzado');
    comp.tituloTouched.set(true);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    expect(comp.tituloInvalid()).toBe(false);
  });

  it('REQ-VALIDATION: videoUrlInvalid is true when url is empty and touched', () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    comp.videoForm.set({ titulo: 'Test', url: '', proveedor: 'youtube', duracionS: undefined, descripcion: '' });
    comp.videoUrlTouched.set(true);

    expect(comp.videoUrlInvalid()).toBe(true);
  });

  it('REQ-VALIDATION: videoUrlInvalid is true when url is invalid (no http) and touched', () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    comp.videoForm.set({ titulo: 'Test', url: 'not-a-url', proveedor: 'youtube', duracionS: undefined, descripcion: '' });
    comp.videoUrlTouched.set(true);

    expect(comp.videoUrlInvalid()).toBe(true);
  });

  it('REQ-VALIDATION: videoUrlInvalid is false when url is valid http and touched', () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    comp.videoForm.set({ titulo: 'Test', url: 'https://youtube.com/watch?v=abc', proveedor: 'youtube', duracionS: undefined, descripcion: '' });
    comp.videoUrlTouched.set(true);

    expect(comp.videoUrlInvalid()).toBe(false);
  });

  it('REQ-VALIDATION: video touched signals reset on openAddVideoDialog()', () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;

    comp.videoTituloTouched.set(true);
    comp.videoUrlTouched.set(true);

    comp.openAddVideoDialog('sec-1');

    expect(comp.videoTituloTouched()).toBe(false);
    expect(comp.videoUrlTouched()).toBe(false);
  });
});
