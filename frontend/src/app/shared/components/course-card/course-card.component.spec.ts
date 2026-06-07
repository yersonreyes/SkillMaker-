/**
 * course-card.component.spec.ts — CourseCardComponent unit tests (Strict TDD).
 *
 * Updated in course-structure-v2:
 * - CatalogCourseCard gains metadata fields (miniaturaUrl, nivel, categorias, stats)
 * - Tests added for: miniatura/placeholder, nivel tag, categorias chips, stats
 * - Existing tests preserved (titulo, creadorNombre, descripcion, open event)
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { ConfirmationService, MessageService } from 'primeng/api';

import { CourseCardComponent } from './course-card.component';
import type { CatalogCourseCard } from '@core/services/courseCatalogService/course-catalog.dto';

const MOCK_CARD_MINIMAL: CatalogCourseCard = {
  id: 'course-1',
  titulo: 'Go Avanzado',
  descripcion: 'Aprende Go de verdad con ejemplos practicos',
  creadorNombre: 'Yerson Reyes',
  createdAt: '2026-01-01T00:00:00Z',
  miniaturaUrl: '',
  nivel: null,
  categorias: [],
  cantidadClases: 0,
  horasVideo: 0,
  horasPractico: 0,
};

const MOCK_CARD_FULL: CatalogCourseCard = {
  id: 'course-2',
  titulo: 'TypeScript Pro',
  descripcion: 'Curso completo de TypeScript',
  creadorNombre: 'Ana Lopez',
  createdAt: '2026-02-01T00:00:00Z',
  miniaturaUrl: 'http://minio/thumbnail/cover.jpg',
  nivel: 'intermedio',
  categorias: [
    { id: 'cat-1', nombre: 'Frontend' },
    { id: 'cat-2', nombre: 'Backend' },
  ],
  cantidadClases: 5,
  horasVideo: 2.5,
  horasPractico: 1.5,
};

describe('CourseCardComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CourseCardComponent],
      providers: [
        provideAnimationsAsync(),
        ConfirmationService,
        MessageService,
      ],
    }).compileComponents();
  });

  afterEach(() => {
    TestBed.resetTestingModule();
  });

  // ── Existing tests preserved ──────────────────────────────────────────────

  it('renders titulo from @Input() card', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_MINIMAL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Go Avanzado');
  });

  it('renders creadorNombre from @Input() card', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_MINIMAL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Yerson Reyes');
  });

  it('renders descripcion from @Input() card', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_MINIMAL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Aprende Go de verdad');
  });

  it('emits open event when "Ver detalle" button is clicked', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_MINIMAL;
    fixture.detectChanges();
    await fixture.whenStable();

    const openSpy = vi.fn();
    fixture.componentInstance.open.subscribe(openSpy);

    const btn: HTMLButtonElement = fixture.nativeElement.querySelector('button');
    expect(btn).not.toBeNull();
    btn.click();
    fixture.detectChanges();

    expect(openSpy).toHaveBeenCalledTimes(1);
  });

  it('does NOT emit open before interaction', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_MINIMAL;
    fixture.detectChanges();
    await fixture.whenStable();

    const openSpy = vi.fn();
    fixture.componentInstance.open.subscribe(openSpy);

    expect(openSpy).not.toHaveBeenCalled();
  });

  // ── Miniatura / placeholder (course-structure-v2) ─────────────────────────

  it('shows miniatura img when miniaturaUrl is set', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_FULL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    const img = el.querySelector('.course-card__cover-img') as HTMLImageElement;
    expect(img).not.toBeNull();
    expect(img.src).toContain('thumbnail/cover.jpg');
  });

  it('shows placeholder when miniaturaUrl is empty string', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_MINIMAL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    const img = el.querySelector('.course-card__cover-img');
    const placeholder = el.querySelector('.course-card__cover-placeholder');
    expect(img).toBeNull();
    expect(placeholder).not.toBeNull();
  });

  // ── Nivel tag ─────────────────────────────────────────────────────────────

  it('renders nivel tag when nivel is set', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_FULL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    const nivelTag = el.querySelector('.course-card__nivel');
    expect(nivelTag).not.toBeNull();
    expect(nivelTag?.textContent).toContain('intermedio');
  });

  it('does NOT render nivel tag when nivel is null', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_MINIMAL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    const nivelTag = el.querySelector('.course-card__nivel');
    expect(nivelTag).toBeNull();
  });

  // ── Categorias chips ──────────────────────────────────────────────────────

  it('renders categorias chips when categorias are set', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_FULL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    const categorias = el.querySelector('.course-card__categorias');
    expect(categorias).not.toBeNull();
    expect(categorias?.textContent).toContain('Frontend');
    expect(categorias?.textContent).toContain('Backend');
  });

  it('does NOT render categorias section when categorias is empty', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_MINIMAL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    const categorias = el.querySelector('.course-card__categorias');
    expect(categorias).toBeNull();
  });

  // ── Stats ─────────────────────────────────────────────────────────────────

  it('renders cantidadClases and horasVideo in stats when non-zero', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_FULL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    const stats = el.querySelector('.course-card__stats');
    expect(stats).not.toBeNull();
    expect(stats?.textContent).toContain('5');     // cantidadClases
    expect(stats?.textContent).toContain('2.5');   // horasVideo
    expect(stats?.textContent).toContain('1.5');   // horasPractico
  });

  it('does NOT render stats when all are zero', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_MINIMAL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    const statItems = el.querySelectorAll('.course-card__stat');
    expect(statItems.length).toBe(0);
  });

  // ── REQ-LAZY: lazy loading attributes ─────────────────────────────────────

  it('REQ-LAZY: img has loading="lazy" and decoding="async" when miniaturaUrl is non-null', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_FULL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    const img = el.querySelector('.course-card__cover-img') as HTMLImageElement;
    expect(img).not.toBeNull();
    expect(img.getAttribute('loading')).toBe('lazy');
    expect(img.getAttribute('decoding')).toBe('async');
  });

  it('REQ-LAZY: placeholder renders (no lazy img) when miniaturaUrl is empty string', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD_MINIMAL;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    const img = el.querySelector('[loading="lazy"]');
    const placeholder = el.querySelector('.course-card__cover-placeholder');
    expect(img).toBeNull();
    expect(placeholder).not.toBeNull();
  });
});
