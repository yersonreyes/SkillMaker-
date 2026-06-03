/**
 * curso-editar.component.spec.ts — Component unit tests (Vitest + Angular TestBed).
 *
 * Covers:
 *  - Component load → getById called → form populated
 *  - Save → update called with form values
 *  - "Enviar a revisión" button is always disabled in C2.1
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter, ActivatedRoute, convertToParamMap } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { of } from 'rxjs';

import { CursoEditarComponent } from './curso-editar.component';
import { CourseService } from '@core/services/courseService/course.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { CourseDetail } from '@core/services/courseService/course.res.dto';

// ── Helpers ───────────────────────────────────────────────────────────────────

const MOCK_COURSE_DETAIL: CourseDetail = {
  id: 'c-1',
  creadorId: 'user-1',
  titulo: 'Go avanzado',
  descripcion: 'Curso de Go para profesionales',
  estado: 'borrador',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

// ── Spec ──────────────────────────────────────────────────────────────────────

describe('CursoEditarComponent', () => {
  let courseServiceSpy: Partial<CourseService>;
  let uiDialogSpy: Partial<UiDialogService>;

  beforeEach(async () => {
    courseServiceSpy = {
      getById: vi.fn().mockResolvedValue(MOCK_COURSE_DETAIL),
      update: vi.fn().mockResolvedValue({ ...MOCK_COURSE_DETAIL, titulo: 'Actualizado' }),
    };

    uiDialogSpy = {
      showSuccess: vi.fn(),
      showError: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [CursoEditarComponent],
      providers: [
        { provide: CourseService, useValue: courseServiceSpy },
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

    // Set courseId directly and call loadCourse — tests the load logic independent of routing
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';
    await comp.loadCourse();

    expect(courseServiceSpy.getById).toHaveBeenCalledWith('c-1');
    expect(comp.titulo()).toBe('Go avanzado');
    expect(comp.descripcion()).toBe('Curso de Go para profesionales');
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
    // courseId defaults to '' — do NOT set it

    await comp.onSave();

    expect(courseServiceSpy.update).not.toHaveBeenCalled();
  });

  // ── F3-enviar-disabled: "Enviar a revisión" button is disabled ────────────

  it('submitDisabled is always true (C2.1 — wired in C2.2)', () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();

    expect(comp.submitDisabled).toBe(true);
  });
});
