/**
 * course-detail-eval.spec.ts — CourseDetailAlumnoComponent "Rendir evaluación" button tests.
 *
 * TDD Cycle: RED (this file written first) → GREEN.
 *
 * Covers:
 *  - "Rendir evaluación" button shown in enrolled branch when eval summary exists
 *  - "Rendir evaluación" button absent when summary is null (404)
 *  - Button click navigates to /platform/evaluations/:evaluationId (Router.navigate)
 *
 * Pattern: mirrors course-detail-cert.spec.ts (same component, new file for separation).
 * Key lesson: do NOT use provideRouter — it overrides ActivatedRoute snapshot.
 *             Provide Router as a stub directly.
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, afterEach, vi } from 'vitest';
import {
  ActivatedRoute,
  Router,
  convertToParamMap,
} from '@angular/router';
import { of } from 'rxjs';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { CourseDetailAlumnoComponent } from './course-detail.component';
import { CourseCatalogService } from '@core/services/courseCatalogService/course-catalog.service';
import { CertificateService } from '@core/services/certificateService/certificate.service';
import { EvaluationService } from '@core/services/evaluationService/evaluation.service';
import type { CourseDetailAlumnoResponse } from '@core/services/courseCatalogService/course-catalog.dto';
import type { EvaluationSummary } from '@core/services/evaluationService/evaluation.dto';

// ── Fixtures ──────────────────────────────────────────────────────────────────

const MOCK_ENROLLED: CourseDetailAlumnoResponse = {
  enrolled: true,
  id: 'course-1',
  titulo: 'Go Avanzado',
  descripcion: 'Aprende Go de verdad',
  creadorNombre: 'Yerson Reyes',
  secciones: [],
  materiales: [],
};

const MOCK_SUMMARY: EvaluationSummary = {
  evaluationId: 'eval-1',
  notaMinima: 70,
  intentosMax: 3,
};

const noCertSpy: Partial<CertificateService> = {
  getMyCertificates: vi.fn().mockResolvedValue([]),
  getDownloadUrl: vi.fn().mockResolvedValue({ url: '', expiresAt: '' }),
};

// ── Helper ────────────────────────────────────────────────────────────────────

async function createComponent(
  catalogSpy: Partial<CourseCatalogService>,
  evalSpy: Partial<EvaluationService>,
  routerStub?: { navigate: ReturnType<typeof vi.fn> },
) {
  const router = routerStub ?? { navigate: vi.fn().mockResolvedValue(true) };

  await TestBed.configureTestingModule({
    imports: [CourseDetailAlumnoComponent],
    providers: [
      { provide: CourseCatalogService, useValue: catalogSpy },
      { provide: CertificateService, useValue: noCertSpy },
      { provide: EvaluationService, useValue: evalSpy },
      // Do NOT use provideRouter — it overrides ActivatedRoute snapshot.
      // Provide Router as a stub instead (C5.1 lesson from course-detail-cert.spec.ts).
      { provide: Router, useValue: router },
      {
        provide: ActivatedRoute,
        useValue: {
          snapshot: { paramMap: convertToParamMap({ id: 'course-1' }) },
          params: of({ id: 'course-1' }),
        },
      },
      provideAnimationsAsync(),
      ConfirmationService,
      MessageService,
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(CourseDetailAlumnoComponent);
  const comp = fixture.componentInstance;
  return { fixture, comp, router };
}

// ── Specs ─────────────────────────────────────────────────────────────────────

describe('CourseDetailAlumnoComponent — Rendir evaluación', () => {
  afterEach(() => {
    TestBed.resetTestingModule();
  });

  it('"Rendir evaluación" button visible when enrolled and eval summary exists', async () => {
    const catalogSpy: Partial<CourseCatalogService> = {
      getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED),
      enroll: vi.fn(),
    };
    const evalSpy: Partial<EvaluationService> = {
      getCourseEvaluationSummary: vi.fn().mockResolvedValue(MOCK_SUMMARY),
    };

    const { fixture, comp } = await createComponent(catalogSpy, evalSpy);

    // Pattern from course-detail-cert.spec.ts (C5.1 lesson):
    // Set courseId + courseIdSignal before detectChanges so ngOnInit uses correct id.
    // Then call loadDetail explicitly AFTER detectChanges (which triggers ngOnInit's async loadDetail)
    // and call it AGAIN to settle the ngOnInit-triggered async load.
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseIdSignal'].set('course-1');
    await comp.loadDetail();
    fixture.detectChanges();
    await comp.loadDetail();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    const rendirBtn = Array.from(el.querySelectorAll('button')).find(b =>
      b.textContent?.trim().includes('Rendir evaluación'),
    );
    expect(rendirBtn).toBeDefined();
  });

  it('"Rendir evaluación" button absent when summary is null (no evaluation)', async () => {
    const catalogSpy: Partial<CourseCatalogService> = {
      getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED),
      enroll: vi.fn(),
    };
    const evalSpy: Partial<EvaluationService> = {
      getCourseEvaluationSummary: vi.fn().mockResolvedValue(null),
    };

    const { fixture, comp } = await createComponent(catalogSpy, evalSpy);

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseIdSignal'].set('course-1');
    await comp.loadDetail();
    fixture.detectChanges();
    await comp.loadDetail();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    const rendirBtn = Array.from(el.querySelectorAll('button')).find(b =>
      b.textContent?.trim().includes('Rendir evaluación'),
    );
    expect(rendirBtn).toBeUndefined();
  });

  it('clicking "Rendir evaluación" navigates to /platform/evaluations/:evaluationId', async () => {
    const catalogSpy: Partial<CourseCatalogService> = {
      getDetail: vi.fn().mockResolvedValue(MOCK_ENROLLED),
      enroll: vi.fn(),
    };
    const evalSpy: Partial<EvaluationService> = {
      getCourseEvaluationSummary: vi.fn().mockResolvedValue(MOCK_SUMMARY),
    };
    const routerStub = { navigate: vi.fn().mockResolvedValue(true) };

    const { fixture, comp } = await createComponent(catalogSpy, evalSpy, routerStub);

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseIdSignal'].set('course-1');
    await comp.loadDetail();
    fixture.detectChanges();
    await comp.loadDetail();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    const rendirBtn = Array.from(el.querySelectorAll('button')).find(b =>
      b.textContent?.trim().includes('Rendir evaluación'),
    ) as HTMLButtonElement | undefined;

    expect(rendirBtn).toBeDefined();
    rendirBtn!.click();

    expect(routerStub.navigate).toHaveBeenCalledWith(['/platform/evaluations', 'eval-1']);
  });
});
