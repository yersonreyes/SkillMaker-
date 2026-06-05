/**
 * my-courses.component.spec.ts — MyCoursesComponent unit tests (Strict TDD — RED → GREEN).
 *
 * Covers:
 *  - Renders course rows from getMyCourses()
 *  - Shows "Completado" badge (.tag) when completado=true
 *  - "Continuar" button navigates with ABSOLUTE /platform/courses/:id path (C2.2 anti-pattern)
 *  - Empty state when no courses returned
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter, Router } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { MyCoursesComponent } from './my-courses.component';
import { CourseCatalogService } from '@core/services/courseCatalogService/course-catalog.service';
import type { MyCourseItem } from '@core/services/courseCatalogService/course-catalog.dto';

const MOCK_COMPLETED: MyCourseItem = {
  courseId: 'course-1',
  titulo: 'Go Avanzado',
  creadorNombre: 'Yerson Reyes',
  completado: true,
  inscritoEn: '2026-01-01T00:00:00Z',
};

const MOCK_IN_PROGRESS: MyCourseItem = {
  courseId: 'course-2',
  titulo: 'Angular Signals',
  creadorNombre: 'Otro Autor',
  completado: false,
  inscritoEn: '2026-02-01T00:00:00Z',
};

describe('MyCoursesComponent', () => {
  let catalogServiceSpy: Partial<CourseCatalogService>;

  beforeEach(async () => {
    catalogServiceSpy = {
      getMyCourses: vi.fn().mockResolvedValue([MOCK_COMPLETED, MOCK_IN_PROGRESS]),
    };

    await TestBed.configureTestingModule({
      imports: [MyCoursesComponent],
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
    TestBed.resetTestingModule();
  });

  it('calls getMyCourses on init', async () => {
    const fixture = TestBed.createComponent(MyCoursesComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(catalogServiceSpy.getMyCourses).toHaveBeenCalledTimes(1);
  });

  it('renders course rows after load', async () => {
    const fixture = TestBed.createComponent(MyCoursesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Go Avanzado');
    expect(el.textContent).toContain('Angular Signals');
  });

  it('shows "Completado" badge (.tag) when completado=true', async () => {
    const fixture = TestBed.createComponent(MyCoursesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    // The .tag element should exist and show Completado for course-1
    const tags = el.querySelectorAll('.tag');
    expect(tags.length).toBeGreaterThan(0);
    const completadoTag = Array.from(tags).find(t => t.textContent?.trim() === 'Completado');
    expect(completadoTag).toBeDefined();
  });

  it('shows empty state (.empty) when no courses returned', async () => {
    (catalogServiceSpy.getMyCourses as ReturnType<typeof vi.fn>).mockResolvedValue([]);

    const fixture = TestBed.createComponent(MyCoursesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    const emptyEl = el.querySelector('.empty');
    expect(emptyEl).not.toBeNull();
  });

  it('"Continuar" navigates to absolute /platform/courses/:id path', async () => {
    const fixture = TestBed.createComponent(MyCoursesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const router = TestBed.inject(Router);
    const navigateSpy = vi.spyOn(router, 'navigate').mockResolvedValue(true);

    fixture.componentInstance.goToCourse(MOCK_IN_PROGRESS);

    expect(navigateSpy).toHaveBeenCalledWith(['/platform/courses', 'course-2']);
  });
});
