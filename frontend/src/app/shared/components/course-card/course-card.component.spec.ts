/**
 * course-card.component.spec.ts — CourseCardComponent unit tests (Strict TDD — RED → GREEN).
 *
 * Covers:
 *  - Renders titulo and creadorNombre from @Input() card
 *  - Renders descripcion (truncated via CSS — component renders full text)
 *  - Emits open event when "Ver detalle" button is clicked
 *  - Does NOT emit when nothing is clicked
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { ConfirmationService, MessageService } from 'primeng/api';

import { CourseCardComponent } from './course-card.component';
import type { CatalogCourseCard } from '@core/services/courseCatalogService/course-catalog.dto';

const MOCK_CARD: CatalogCourseCard = {
  id: 'course-1',
  titulo: 'Go Avanzado',
  descripcion: 'Aprende Go de verdad con ejemplos practicos',
  creadorNombre: 'Yerson Reyes',
  createdAt: '2026-01-01T00:00:00Z',
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

  it('renders titulo from @Input() card', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Go Avanzado');
  });

  it('renders creadorNombre from @Input() card', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Yerson Reyes');
  });

  it('renders descripcion from @Input() card', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD;
    fixture.detectChanges();
    await fixture.whenStable();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Aprende Go de verdad');
  });

  it('emits open event when "Ver detalle" button is clicked', async () => {
    const fixture = TestBed.createComponent(CourseCardComponent);
    fixture.componentInstance.card = MOCK_CARD;
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
    fixture.componentInstance.card = MOCK_CARD;
    fixture.detectChanges();
    await fixture.whenStable();

    const openSpy = vi.fn();
    fixture.componentInstance.open.subscribe(openSpy);

    expect(openSpy).not.toHaveBeenCalled();
  });
});
