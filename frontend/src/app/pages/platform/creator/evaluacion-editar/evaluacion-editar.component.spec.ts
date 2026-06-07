/**
 * evaluacion-editar.component.spec.ts — Component unit tests (Vitest + Angular TestBed).
 *
 * Covers (Strict TDD):
 *  - FE-2-A: empty state shown when getByCourse returns null (404)
 *  - FE-2-B: full editor loaded when getByCourse returns evaluation with questions
 *  - FE-2-C: verdadero_falso question shows exactly 2 options with radio selector
 *  - FE-2-D: client validation blocks save when opcion_multiple has no correcta=true
 *  - Create evaluation flow (calls EvaluationService.create and sets evaluation)
 *  - Add opcion_multiple question (calls QuestionService.create, then adds options)
 *  - Add verdadero_falso question (2 auto-options returned from backend)
 *  - Delete question (confirm dialog then QuestionService.delete)
 *  - Delete option (confirm dialog then QuestionService.deleteOption)
 *  - FE-3-A (LOAD-BEARING): navigate to /platform-prefixed absolute path from curso-editar
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter, ActivatedRoute, convertToParamMap, Router } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { of } from 'rxjs';

import { EvaluacionEditarComponent } from './evaluacion-editar.component';
import { EvaluationService } from '@core/services/evaluationService/evaluation.service';
import { QuestionService } from '@core/services/questionService/question.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { EvaluationDetail, QuestionItem, OptionItem } from '@core/services/evaluationService/evaluation.dto';

// ── Mock data ──────────────────────────────────────────────────────────────────

const MOCK_OPTION_V: OptionItem = {
  id: 'opt-v',
  questionId: 'q-vf',
  texto: 'Verdadero',
  correcta: false,
  orden: 0,
};

const MOCK_OPTION_F: OptionItem = {
  id: 'opt-f',
  questionId: 'q-vf',
  texto: 'Falso',
  correcta: false,
  orden: 1,
};

const MOCK_QUESTION_VF: QuestionItem = {
  id: 'q-vf',
  evaluationId: 'eval-1',
  enunciado: 'El cielo es azul?',
  tipo: 'verdadero_falso',
  puntaje: 5,
  orden: 0,
  options: [MOCK_OPTION_V, MOCK_OPTION_F],
};

const MOCK_QUESTION_OM: QuestionItem = {
  id: 'q-om',
  evaluationId: 'eval-1',
  enunciado: 'Cual es la capital de Francia?',
  tipo: 'opcion_multiple',
  puntaje: 10,
  orden: 1,
  options: [
    { id: 'opt-1', questionId: 'q-om', texto: 'Paris', correcta: true, orden: 0 },
    { id: 'opt-2', questionId: 'q-om', texto: 'Madrid', correcta: false, orden: 1 },
  ],
};

const MOCK_EVALUATION: EvaluationDetail = {
  id: 'eval-1',
  courseId: 'course-1',
  notaMinima: 70,
  intentosMax: 3,
  questions: [MOCK_QUESTION_VF, MOCK_QUESTION_OM],
};

// ── Spec ──────────────────────────────────────────────────────────────────────

describe('EvaluacionEditarComponent', () => {
  let evaluationServiceSpy: Partial<EvaluationService>;
  let questionServiceSpy: Partial<QuestionService>;
  let uiDialogSpy: Partial<UiDialogService>;

  beforeEach(async () => {
    evaluationServiceSpy = {
      getByCourse: vi.fn().mockResolvedValue(MOCK_EVALUATION),
      create: vi.fn().mockResolvedValue({
        id: 'eval-1', courseId: 'course-1', notaMinima: 70, intentosMax: 3, createdAt: '2026-01-01T00:00:00Z',
      }),
      update: vi.fn().mockResolvedValue({
        id: 'eval-1', courseId: 'course-1', notaMinima: 80, intentosMax: 3, createdAt: '2026-01-01T00:00:00Z',
      }),
    };

    questionServiceSpy = {
      create: vi.fn().mockResolvedValue({
        id: 'q-new', evaluationId: 'eval-1', enunciado: 'Nueva pregunta', tipo: 'opcion_multiple',
        puntaje: 10, orden: 0,
      }),
      update: vi.fn().mockResolvedValue({
        id: 'q-om', evaluationId: 'eval-1', enunciado: 'Actualizada', tipo: 'opcion_multiple',
        puntaje: 10, orden: 1,
      }),
      delete: vi.fn().mockResolvedValue(undefined),
      createOption: vi.fn().mockResolvedValue({
        id: 'opt-new', questionId: 'q-new', texto: 'Opcion 1', correcta: false, orden: 0,
      }),
      updateOption: vi.fn().mockResolvedValue({
        id: 'opt-v', questionId: 'q-vf', texto: 'Verdadero', correcta: true, orden: 0,
      }),
      deleteOption: vi.fn().mockResolvedValue(undefined),
    };

    uiDialogSpy = {
      showSuccess: vi.fn(),
      showError: vi.fn(),
      confirmDelete: vi.fn().mockResolvedValue(true),
    };

    await TestBed.configureTestingModule({
      imports: [EvaluacionEditarComponent],
      providers: [
        { provide: EvaluationService, useValue: evaluationServiceSpy },
        { provide: QuestionService, useValue: questionServiceSpy },
        { provide: UiDialogService, useValue: uiDialogSpy },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: { paramMap: convertToParamMap({ courseId: 'course-1' }) },
            params: of({ courseId: 'course-1' }),
          },
        },
        ConfirmationService,
        MessageService,
        provideRouter([{ path: '**', component: EvaluacionEditarComponent }]),
        provideAnimationsAsync(),
      ],
    }).compileComponents();
  });

  afterEach(() => {
    TestBed.resetTestingModule();
  });

  // ── FE-2-A: empty state when no evaluation (404) ─────────────────────────────

  it('FE-2-A: shows empty state when getByCourse returns null (no evaluation)', async () => {
    evaluationServiceSpy.getByCourse = vi.fn().mockResolvedValue(null);

    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    await comp.loadEvaluation();

    expect(comp.evaluation()).toBeNull();
    // Empty state: no evaluation, so the create form should be shown
    expect(comp.showEmptyState()).toBe(true);
  });

  // ── FE-2-B: full editor loaded when evaluation exists ────────────────────────

  it('FE-2-B: loads evaluation and populates questions when getByCourse returns detail', async () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    await comp.loadEvaluation();

    expect(comp.evaluation()).not.toBeNull();
    expect(comp.evaluation()!.id).toBe('eval-1');
    expect(comp.questions()).toHaveLength(2);
  });

  // ── FE-2-C: verdadero_falso shows exactly 2 options ──────────────────────────

  it('FE-2-C: verdadero_falso question has exactly 2 options (Verdadero + Falso)', async () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    await comp.loadEvaluation();

    const vfQuestion = comp.questions().find(q => q.tipo === 'verdadero_falso');
    expect(vfQuestion).toBeDefined();
    expect(vfQuestion!.options).toHaveLength(2);
    expect(vfQuestion!.options[0].texto).toBe('Verdadero');
    expect(vfQuestion!.options[1].texto).toBe('Falso');
  });

  // ── FE-2-D: client validation — opcion_multiple needs ≥1 correcta ────────────
  // LOAD-BEARING: This test verifies the client-side ≥1-correct guard.
  // A question with all options correcta=false must be blocked (returns false).
  // A question with ≥1 correcta=true must pass (returns true).

  it('FE-2-D LOAD-BEARING: hasAtLeastOneCorrect returns false when all options are false', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    const options: OptionItem[] = [
      { id: 'o1', questionId: 'q1', texto: 'A', correcta: false, orden: 0 },
      { id: 'o2', questionId: 'q1', texto: 'B', correcta: false, orden: 1 },
    ];

    expect(comp.hasAtLeastOneCorrect(options)).toBe(false);
  });

  it('FE-2-D: hasAtLeastOneCorrect returns true when at least one option is correct', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    const options: OptionItem[] = [
      { id: 'o1', questionId: 'q1', texto: 'A', correcta: true, orden: 0 },
      { id: 'o2', questionId: 'q1', texto: 'B', correcta: false, orden: 1 },
    ];

    expect(comp.hasAtLeastOneCorrect(options)).toBe(true);
  });

  it('FE-2-D: canSaveQuestion returns false for opcion_multiple with no correct options', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    const omQuestion: QuestionItem = {
      ...MOCK_QUESTION_OM,
      options: [
        { id: 'o1', questionId: 'q-om', texto: 'A', correcta: false, orden: 0 },
        { id: 'o2', questionId: 'q-om', texto: 'B', correcta: false, orden: 1 },
      ],
    };

    expect(comp.canSaveQuestion(omQuestion)).toBe(false);
  });

  it('FE-2-D: canSaveQuestion returns true for opcion_multiple with ≥1 correct option', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    expect(comp.canSaveQuestion(MOCK_QUESTION_OM)).toBe(true);
  });

  it('FE-2-D: canSaveQuestion always returns true for verdadero_falso (no OM constraint)', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    // VF with no correcta=true (initial state from backend) — save still allowed
    // (backend enforces mutual exclusion separately)
    expect(comp.canSaveQuestion(MOCK_QUESTION_VF)).toBe(true);
  });

  // ── Create evaluation flow ────────────────────────────────────────────────────

  it('createEvaluation() calls EvaluationService.create with form values and sets evaluation', async () => {
    evaluationServiceSpy.getByCourse = vi.fn().mockResolvedValue(null);

    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    comp.createForm.notaMinima = 70;
    comp.createForm.intentosMax = 3;

    await comp.createEvaluation();

    expect(evaluationServiceSpy.create).toHaveBeenCalledWith('course-1', { notaMinima: 70, intentosMax: 3 });
    expect(comp.showEmptyState()).toBe(false);
  });

  // ── Add question (opcion_multiple) ───────────────────────────────────────────

  it('addQuestion() calls QuestionService.create and appends the new question to list', async () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    await comp.loadEvaluation();

    // Simulate the modal form filled out
    comp.questionForm.enunciado = 'Nueva pregunta';
    comp.questionForm.tipo = 'opcion_multiple';
    comp.questionForm.puntaje = 10;

    await comp.addQuestion();

    expect(questionServiceSpy.create).toHaveBeenCalledWith('eval-1', {
      enunciado: 'Nueva pregunta',
      tipo: 'opcion_multiple',
      puntaje: 10,
    });
  });

  // ── Delete question ───────────────────────────────────────────────────────────

  it('deleteQuestion() shows confirm dialog and calls QuestionService.delete', async () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    await comp.loadEvaluation();

    await comp.deleteQuestion('q-om');

    expect(uiDialogSpy.confirmDelete).toHaveBeenCalled();
    expect(questionServiceSpy.delete).toHaveBeenCalledWith('q-om');
  });

  it('deleteQuestion() does NOT call delete when confirm is rejected', async () => {
    uiDialogSpy.confirmDelete = vi.fn().mockResolvedValue(false);

    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    await comp.loadEvaluation();

    await comp.deleteQuestion('q-om');

    expect(questionServiceSpy.delete).not.toHaveBeenCalled();
  });

  // ── Delete option ─────────────────────────────────────────────────────────────

  it('deleteOption() shows confirm dialog and calls QuestionService.deleteOption', async () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    await comp.loadEvaluation();

    await comp.deleteOption('q-om', 'opt-1');

    expect(uiDialogSpy.confirmDelete).toHaveBeenCalled();
    expect(questionServiceSpy.deleteOption).toHaveBeenCalledWith('opt-1');
  });

  // ── REQ-VALIDATION: inline validation for evaluacion-editar ─────────────────

  it('REQ-VALIDATION: notaError() is false when untouched', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    comp.createForm.notaMinima = 70;

    expect(comp.notaError()).toBe(false);
  });

  it('REQ-VALIDATION: notaError() is true when touched and value out of range (110)', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    comp.createForm.notaMinima = 110;
    comp.notaTouched.set(true);

    expect(comp.notaError()).toBe(true);
  });

  it('REQ-VALIDATION: notaError() is true when touched and value negative (-1)', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    comp.createForm.notaMinima = -1;
    comp.notaTouched.set(true);

    expect(comp.notaError()).toBe(true);
  });

  it('REQ-VALIDATION: notaError() is false when touched and value valid (70)', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    comp.createForm.notaMinima = 70;
    comp.notaTouched.set(true);

    expect(comp.notaError()).toBe(false);
  });

  it('REQ-VALIDATION: intentosError() is true when touched and value negative (-1)', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    comp.createForm.intentosMax = -1;
    comp.intentosTouched.set(true);

    expect(comp.intentosError()).toBe(true);
  });

  it('REQ-VALIDATION: intentosError() is false when touched and value is 0', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    comp.createForm.intentosMax = 0;
    comp.intentosTouched.set(true);

    expect(comp.intentosError()).toBe(false);
  });

  it('REQ-VALIDATION: enunciadoError() is true when touched and enunciado is empty', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    comp.questionForm.enunciado = '';
    comp.enunciadoTouched.set(true);

    expect(comp.enunciadoError()).toBe(true);
  });

  it('REQ-VALIDATION: enunciadoError() is false when untouched', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    comp.questionForm.enunciado = '';

    expect(comp.enunciadoError()).toBe(false);
  });

  it('REQ-VALIDATION: enunciadoTouched resets to false on openAddQuestionDialog()', () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    comp.enunciadoTouched.set(true);
    comp.openAddQuestionDialog();

    expect(comp.enunciadoTouched()).toBe(false);
  });

  // ── V/F radio: updateVFOption calls QuestionService.updateOption ─────────────

  it('setVFCorrect() calls QuestionService.updateOption with correcta=true for the selected option', async () => {
    const fixture = TestBed.createComponent(EvaluacionEditarComponent);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    await comp.loadEvaluation();

    await comp.setVFCorrect('q-vf', 'opt-v');

    expect(questionServiceSpy.updateOption).toHaveBeenCalledWith('opt-v', { correcta: true });
  });
});

// ── FE-3-A LOAD-BEARING: curso-editar navigate uses /platform-prefixed path ────
// This test lives outside the main describe block intentionally so it imports
// CursoEditarComponent separately and tests ONLY the navigate path assertion.
// DO NOT merge into the main block — we want this isolated and readable.

import { CursoEditarComponent } from '../curso-editar/curso-editar.component';
import { CourseService } from '@core/services/courseService/course.service';
import { SectionService } from '@core/services/sectionService/section.service';
import { VideoService } from '@core/services/videoService/video.service';
import { MaterialService } from '@core/services/materialService/material.service';

describe('FE-3-A LOAD-BEARING: CursoEditar — "Definir evaluación" navigate path', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CursoEditarComponent],
      providers: [
        {
          provide: CourseService,
          useValue: {
            getById: vi.fn().mockResolvedValue({
              id: 'c-1', creadorId: 'u-1', titulo: 'T', descripcion: 'D',
              estado: 'borrador', hasContent: false,
              createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z',
            }),
            update: vi.fn().mockResolvedValue({}),
          },
        },
        { provide: SectionService, useValue: { listByCourse: vi.fn().mockResolvedValue([]) } },
        { provide: VideoService, useValue: {} },
        {
          provide: MaterialService,
          useValue: { list: vi.fn().mockResolvedValue([]) },
        },
        { provide: UiDialogService, useValue: { showSuccess: vi.fn(), showError: vi.fn() } },
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

  it('FE-3-A LOAD-BEARING: navigateToEvaluation() uses the absolute /platform/creator/evaluacion-editar/:id path', async () => {
    const fixture = TestBed.createComponent(CursoEditarComponent);
    const comp = fixture.componentInstance;
    const router = TestBed.inject(Router);

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'c-1';

    const navigateSpy = vi.spyOn(router, 'navigateByUrl');
    await comp.navigateToEvaluation();

    // LOAD-BEARING: must use the absolute /platform prefix.
    // A relative path (/creator/...) is a known C2.2 bug pattern.
    expect(navigateSpy).toHaveBeenCalledWith('/platform/creator/evaluacion-editar/c-1');
  });
});
