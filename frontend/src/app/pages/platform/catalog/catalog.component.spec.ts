/**
 * catalog.component.spec.ts — CatalogComponent unit tests (Strict TDD — RED → GREEN).
 *
 * Zoneless Angular 21 — uses vi.useFakeTimers() for debounce testing (NOT fakeAsync).
 *
 * Covers (original):
 *  - Loads catalog on init with default page=1, size=12
 *  - Renders CourseCard items after load
 *  - Shows empty state (.empty) when no courses returned
 *  - Debounce: typing then advancing 300ms fires one getCatalog call with q
 *  - Debounce: advancing < 300ms does NOT fire getCatalog
 *  - Paginator change reloads with new page number
 *
 * Covers (Phase 7 — filter bar):
 *  - categorias loaded on init from CategoriaService
 *  - onFilterChange resets page to 1 and reloads immediately
 *  - filter change resets page even when on page 2
 *  - categoria filter: getCatalog receives categoriaIds array
 *  - nivel filter: getCatalog receives nivel string
 *  - sort filter: getCatalog receives sort string
 *  - hasActiveFilters is false when all defaults
 *  - hasActiveFilters is true when nivel set
 *  - hasActiveFilters is true when categoriaIds non-empty
 *  - hasActiveFilters is true when sort !== 'recientes'
 *  - hasActiveFilters is true when q non-empty
 *  - clearFilters resets all signals + reloads unfiltered
 *  - filtered-empty state shown when hasActiveFilters && courses empty
 *  - default empty state shown when !hasActiveFilters && courses empty
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { CatalogComponent } from './catalog.component';
import { CourseCatalogService } from '@core/services/courseCatalogService/course-catalog.service';
import { CategoriaService } from '@core/services/categoriaService/categoria.service';
import type { CatalogCourseCard, Page } from '@core/services/courseCatalogService/course-catalog.dto';
import type { CategoriaItem } from '@core/services/categoriaService/categoria.dto';

const MOCK_CARD: CatalogCourseCard = {
  id: 'course-1',
  titulo: 'Go Avanzado',
  descripcion: 'Aprende Go de verdad',
  creadorNombre: 'Yerson Reyes',
  createdAt: '2026-01-01T00:00:00Z',
  miniaturaUrl: '',
  nivel: null,
  categorias: [],
  cantidadClases: 0,
  horasVideo: 0,
  horasPractico: 0,
};

const MOCK_EMPTY_PAGE: Page<CatalogCourseCard> = {
  items: [],
  page: 1,
  size: 12,
  total: 0,
  totalPages: 0,
};

const MOCK_PAGE: Page<CatalogCourseCard> = {
  items: [MOCK_CARD],
  page: 1,
  size: 12,
  total: 1,
  totalPages: 1,
};

const MOCK_CATEGORIAS: CategoriaItem[] = [
  { id: 'cat-1', nombre: 'Backend', slug: 'backend' },
  { id: 'cat-2', nombre: 'Frontend', slug: 'frontend' },
];

describe('CatalogComponent', () => {
  let catalogServiceSpy: Partial<CourseCatalogService>;
  let categoriaServiceSpy: Partial<CategoriaService>;

  beforeEach(async () => {
    catalogServiceSpy = {
      getCatalog: vi.fn().mockResolvedValue(MOCK_PAGE),
    };
    categoriaServiceSpy = {
      getAll: vi.fn().mockResolvedValue(MOCK_CATEGORIAS),
    };

    await TestBed.configureTestingModule({
      imports: [CatalogComponent],
      providers: [
        { provide: CourseCatalogService, useValue: catalogServiceSpy },
        { provide: CategoriaService, useValue: categoriaServiceSpy },
        provideRouter([]),
        provideAnimationsAsync(),
        ConfirmationService,
        MessageService,
      ],
    }).compileComponents();
  });

  afterEach(() => {
    vi.useRealTimers();
    TestBed.resetTestingModule();
  });

  // ── Original tests (updated to match new getCatalog signature) ────────────────

  it('calls getCatalog on init with default page=1, size=12', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(catalogServiceSpy.getCatalog).toHaveBeenCalledWith(1, 12, '', undefined, [], 'recientes');
  });

  it('renders CourseCard items after load', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const cards = fixture.nativeElement.querySelectorAll('app-course-card');
    expect(cards.length).toBe(1);
  });

  it('shows default empty state when no courses and no active filters', async () => {
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockResolvedValue(MOCK_EMPTY_PAGE);

    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const emptyEl = fixture.nativeElement.querySelector('.empty');
    expect(emptyEl).not.toBeNull();
  });

  it('debounce: advancing 300ms after typing triggers getCatalog with q', async () => {
    vi.useFakeTimers();

    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await Promise.resolve();
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockClear();

    fixture.componentInstance.onSearchInput('angular');
    vi.advanceTimersByTime(300);
    await Promise.resolve();

    expect(catalogServiceSpy.getCatalog).toHaveBeenCalledWith(1, 12, 'angular', undefined, [], 'recientes');
  });

  it('debounce: advancing < 300ms does NOT fire getCatalog', async () => {
    vi.useFakeTimers();

    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await Promise.resolve();
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockClear();

    fixture.componentInstance.onSearchInput('ang');
    vi.advanceTimersByTime(200);

    expect(catalogServiceSpy.getCatalog).not.toHaveBeenCalled();

    vi.advanceTimersByTime(200);
    await Promise.resolve();
  });

  it('paginator change reloads with new page number', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockClear();

    fixture.componentInstance.onPageChange({ page: 1, rows: 12 });
    await fixture.whenStable();

    expect(catalogServiceSpy.getCatalog).toHaveBeenCalledWith(2, 12, '', undefined, [], 'recientes');
  });

  // ── Phase 7 — filter bar ──────────────────────────────────────────────────────

  it('loads categorias from CategoriaService on init', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(categoriaServiceSpy.getAll).toHaveBeenCalled();
    expect(fixture.componentInstance.categorias()).toEqual(MOCK_CATEGORIAS);
  });

  it('onFilterChange resets page to 1 and reloads immediately', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    // Simulate being on page 2
    fixture.componentInstance.page.set(2);
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockClear();

    fixture.componentInstance.nivel.set('basico');
    fixture.componentInstance.onFilterChange();
    await fixture.whenStable();

    expect(fixture.componentInstance.page()).toBe(1);
    expect(catalogServiceSpy.getCatalog).toHaveBeenCalledWith(1, 12, '', 'basico', [], 'recientes');
  });

  it('onFilterChange with categoriaIds sends repeated params to getCatalog', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockClear();

    fixture.componentInstance.categoriaIds.set(['cat-1', 'cat-2']);
    fixture.componentInstance.onFilterChange();
    await fixture.whenStable();

    expect(catalogServiceSpy.getCatalog).toHaveBeenCalledWith(1, 12, '', undefined, ['cat-1', 'cat-2'], 'recientes');
  });

  it('onFilterChange with sort sends sort param to getCatalog', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockClear();

    fixture.componentInstance.sort.set('titulo');
    fixture.componentInstance.onFilterChange();
    await fixture.whenStable();

    expect(catalogServiceSpy.getCatalog).toHaveBeenCalledWith(1, 12, '', undefined, [], 'titulo');
  });

  // ── hasActiveFilters ──────────────────────────────────────────────────────────

  it('hasActiveFilters is false when all defaults', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.componentInstance.hasActiveFilters()).toBe(false);
  });

  it('hasActiveFilters is true when nivel is set', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    fixture.componentInstance.nivel.set('basico');
    expect(fixture.componentInstance.hasActiveFilters()).toBe(true);
  });

  it('hasActiveFilters is true when categoriaIds is non-empty', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    fixture.componentInstance.categoriaIds.set(['cat-1']);
    expect(fixture.componentInstance.hasActiveFilters()).toBe(true);
  });

  it('hasActiveFilters is true when sort !== recientes', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    fixture.componentInstance.sort.set('titulo');
    expect(fixture.componentInstance.hasActiveFilters()).toBe(true);
  });

  it('hasActiveFilters is true when q is non-empty', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    fixture.componentInstance.q.set('react');
    expect(fixture.componentInstance.hasActiveFilters()).toBe(true);
  });

  // ── clearFilters ──────────────────────────────────────────────────────────────

  it('clearFilters resets all signals to defaults and reloads unfiltered', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    // Set active filters
    fixture.componentInstance.nivel.set('avanzado');
    fixture.componentInstance.categoriaIds.set(['cat-1']);
    fixture.componentInstance.sort.set('titulo');
    fixture.componentInstance.q.set('react');
    fixture.componentInstance.page.set(3);
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockClear();

    fixture.componentInstance.clearFilters();
    await fixture.whenStable();

    expect(fixture.componentInstance.nivel()).toBeNull();
    expect(fixture.componentInstance.categoriaIds()).toEqual([]);
    expect(fixture.componentInstance.sort()).toBe('recientes');
    expect(fixture.componentInstance.q()).toBe('');
    expect(fixture.componentInstance.page()).toBe(1);
    expect(catalogServiceSpy.getCatalog).toHaveBeenCalledWith(1, 12, '', undefined, [], 'recientes');
  });

  // ── Empty state differentiation ───────────────────────────────────────────────

  it('shows filtered empty state when filters active and courses empty', async () => {
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockResolvedValue(MOCK_EMPTY_PAGE);

    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    // Activate a filter then reload
    fixture.componentInstance.nivel.set('basico');
    fixture.componentInstance.onFilterChange();
    await fixture.whenStable();
    fixture.detectChanges();

    const emptyEl = fixture.nativeElement.querySelector('.empty--filtered');
    expect(emptyEl).not.toBeNull();
  });

  it('shows default empty state (not filtered) when no filters and no courses', async () => {
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockResolvedValue(MOCK_EMPTY_PAGE);

    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    // No filters active — should show default empty, not filtered empty
    const filteredEmptyEl = fixture.nativeElement.querySelector('.empty--filtered');
    expect(filteredEmptyEl).toBeNull();
    const defaultEmptyEl = fixture.nativeElement.querySelector('.empty');
    expect(defaultEmptyEl).not.toBeNull();
  });
});
