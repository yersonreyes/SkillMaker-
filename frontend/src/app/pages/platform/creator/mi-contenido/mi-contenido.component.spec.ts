/**
 * mi-contenido.component.spec.ts — Component unit tests (Vitest + Angular TestBed).
 *
 * Covers:
 *  - Lazy load event → listByMe(page/size) called
 *  - Create flow: openCreateDialog → create → success shows dialog hidden / list refreshed
 *  - Row click → navigate to curso-editar/:id
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter, Router } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { MiContenidoComponent } from './mi-contenido.component';
import { CourseService } from '@core/services/courseService/course.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { Page, CourseListItem } from '@core/services/courseService/course.res.dto';

// ── Helpers ───────────────────────────────────────────────────────────────────

function buildPage(items: CourseListItem[]): Page<CourseListItem> {
  return { items, page: 1, size: 20, total: items.length, totalPages: 1 };
}

const MOCK_COURSES: CourseListItem[] = [
  { id: 'c-1', titulo: 'Go avanzado', estado: 'borrador',    createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' },
  { id: 'c-2', titulo: 'TypeScript', estado: 'en_revision', createdAt: '2026-01-02T00:00:00Z', updatedAt: '2026-01-02T00:00:00Z' },
];

// ── Spec ──────────────────────────────────────────────────────────────────────

describe('MiContenidoComponent', () => {
  let courseServiceSpy: Partial<CourseService>;
  let uiDialogSpy: Partial<UiDialogService>;

  beforeEach(async () => {
    courseServiceSpy = {
      listByMe: vi.fn().mockResolvedValue(buildPage(MOCK_COURSES)),
      create: vi.fn().mockResolvedValue({
        id: 'c-new',
        creadorId: 'user-1',
        titulo: 'Nuevo curso',
        descripcion: '',
        estado: 'borrador' as const,
        createdAt: '2026-01-03T00:00:00Z',
        updatedAt: '2026-01-03T00:00:00Z',
      }),
    };

    uiDialogSpy = {
      confirm: vi.fn().mockResolvedValue(true),
      showSuccess: vi.fn(),
      showError: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [MiContenidoComponent],
      providers: [
        { provide: CourseService, useValue: courseServiceSpy },
        { provide: UiDialogService, useValue: uiDialogSpy },
        ConfirmationService,
        MessageService,
        provideRouter([{ path: '**', component: MiContenidoComponent }]),
        provideAnimationsAsync(),
      ],
    }).compileComponents();
  });

  afterEach(() => {
    TestBed.resetTestingModule();
  });

  // ── F1-lazy: onLazyLoad maps first/rows → page/size ───────────────────────

  it('onLazyLoad with first=0, rows=20 calls listByMe with page=1, size=20', async () => {
    const fixture = TestBed.createComponent(MiContenidoComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    await comp.onLazyLoad({ first: 0, rows: 20 });

    expect(courseServiceSpy.listByMe).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1, size: 20 }),
    );
  });

  it('onLazyLoad with first=20, rows=10 calls listByMe with page=3, size=10', async () => {
    const fixture = TestBed.createComponent(MiContenidoComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    await comp.onLazyLoad({ first: 20, rows: 10 });

    expect(courseServiceSpy.listByMe).toHaveBeenCalledWith(
      expect.objectContaining({ page: 3, size: 10 }),
    );
  });

  // ── F2-create: create flow ─────────────────────────────────────────────────

  it('openCreateDialog sets dialogVisible to true', () => {
    const fixture = TestBed.createComponent(MiContenidoComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();

    comp.openCreateDialog();

    expect(comp.dialogVisible()).toBe(true);
  });

  it('onSaveDialog calls CourseService.create and closes dialog on success', async () => {
    const fixture = TestBed.createComponent(MiContenidoComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    comp.newTitulo.set('Nuevo curso');
    comp.newDescripcion.set('Descripcion del curso');
    comp.dialogVisible.set(true);

    await comp.onSaveDialog();

    expect(courseServiceSpy.create).toHaveBeenCalledWith({
      titulo: 'Nuevo curso',
      descripcion: 'Descripcion del curso',
    });
    expect(comp.dialogVisible()).toBe(false);
  });

  it('onSaveDialog does not call create when titulo is empty', async () => {
    const fixture = TestBed.createComponent(MiContenidoComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();

    comp.newTitulo.set('');
    await comp.onSaveDialog();

    expect(courseServiceSpy.create).not.toHaveBeenCalled();
  });

  // ── F3-navigate: row click → router navigate ──────────────────────────────

  it('navigateToCourse calls Router.navigate with curso-editar/:id', () => {
    const fixture = TestBed.createComponent(MiContenidoComponent);
    const comp = fixture.componentInstance;
    const router = TestBed.inject(Router);
    const navigateSpy = vi.spyOn(router, 'navigate');
    fixture.detectChanges();

    comp.navigateToCourse('c-1');

    // Must be the absolute /platform-prefixed path; '/creator/...' does not match
    // any route (platform routes are mounted under /platform) and bounces to catalog.
    expect(navigateSpy).toHaveBeenCalledWith(['/platform/creator/curso-editar', 'c-1']);
  });
});
