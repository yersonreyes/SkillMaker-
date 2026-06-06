/**
 * catalog.component.spec.ts — CatalogComponent unit tests (Strict TDD — RED → GREEN).
 *
 * Zoneless Angular 21 — uses vi.useFakeTimers() for debounce testing (NOT fakeAsync).
 *
 * Covers:
 *  - Loads catalog on init with default page=1, size=12
 *  - Renders CourseCard items after load
 *  - Shows empty state (.empty) when no courses returned
 *  - Debounce: typing then advancing 300ms fires one getCatalog call with q
 *  - Debounce: advancing < 300ms does NOT fire getCatalog
 *  - Paginator change reloads with new page number
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { CatalogComponent } from './catalog.component';
import { CourseCatalogService } from '@core/services/courseCatalogService/course-catalog.service';
import type { CatalogCourseCard, Page } from '@core/services/courseCatalogService/course-catalog.dto';

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

describe('CatalogComponent', () => {
  let catalogServiceSpy: Partial<CourseCatalogService>;

  beforeEach(async () => {
    catalogServiceSpy = {
      getCatalog: vi.fn().mockResolvedValue(MOCK_PAGE),
    };

    await TestBed.configureTestingModule({
      imports: [CatalogComponent],
      providers: [
        { provide: CourseCatalogService, useValue: catalogServiceSpy },
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

  it('calls getCatalog on init with default page=1, size=12', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(catalogServiceSpy.getCatalog).toHaveBeenCalledWith(1, 12, '');
  });

  it('renders CourseCard items after load', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const cards = fixture.nativeElement.querySelectorAll('app-course-card');
    expect(cards.length).toBe(1);
  });

  it('shows empty state (.empty) when no courses returned', async () => {
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
    // Let the initial load run synchronously (promise mock resolves immediately)
    await Promise.resolve();
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockClear();

    fixture.componentInstance.onSearchInput('angular');
    vi.advanceTimersByTime(300);
    await Promise.resolve(); // let the debounce callback and load promise resolve

    expect(catalogServiceSpy.getCatalog).toHaveBeenCalledWith(1, 12, 'angular');
  });

  it('debounce: advancing < 300ms does NOT fire getCatalog', async () => {
    vi.useFakeTimers();

    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await Promise.resolve();
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockClear();

    fixture.componentInstance.onSearchInput('ang');
    vi.advanceTimersByTime(200); // less than 300ms — debounce should NOT have fired

    expect(catalogServiceSpy.getCatalog).not.toHaveBeenCalled();

    // Cleanup: advance past debounce so the subscription settles
    vi.advanceTimersByTime(200);
    await Promise.resolve();
  });

  it('paginator change reloads with new page number', async () => {
    const fixture = TestBed.createComponent(CatalogComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    (catalogServiceSpy.getCatalog as ReturnType<typeof vi.fn>).mockClear();

    // PrimeNG Paginator emits 0-indexed page; page 1 (0-indexed) = page 2 (1-indexed)
    fixture.componentInstance.onPageChange({ page: 1, rows: 12 });
    await fixture.whenStable();

    expect(catalogServiceSpy.getCatalog).toHaveBeenCalledWith(2, 12, '');
  });
});
