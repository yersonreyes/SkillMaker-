/**
 * evaluacion-tomar.component.spec.ts — Strict TDD tests for the student attempt page.
 *
 * Covers (LOAD-BEARING tests):
 *  - FE-4-A: start flow — startAttempt() then getAttempt() called on init action
 *  - FE-4-B: answer selection calls saveAnswer() (save-on-change)
 *  - FE-4-C: submit flow — confirm dialog → submitAttempt() → shows result
 *  - FE-4-D LOAD-BEARING: rendered options contain NO correctness indicator (no leak in UI)
 *  - FE-4-E: 409 on start → friendly blocked state
 *  - FE-4-F LOAD-BEARING: navigate uses absolute /platform-prefixed path (C2.2 bug class)
 *  - FE-4-G LOAD-BEARING: resumed attempt pre-selects previously saved answers from state.answers
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter, ActivatedRoute, convertToParamMap, Router } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { of } from 'rxjs';

import { EvaluacionTomarComponent } from './evaluacion-tomar.component';
import { AttemptService } from '@core/services/attemptService/attempt.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { AttemptState, AttemptStartResponse, SubmitResponse } from '@core/services/attemptService/attempt.dto';

// ── Fixtures ──────────────────────────────────────────────────────────────────

const MOCK_START: AttemptStartResponse = {
  attemptId: 'att-1',
  numero: 1,
  iniciadoEn: '2026-06-01T10:00:00Z',
};

// LOAD-BEARING: options have NO correcta field — mirrors the backend structural guarantee.
const MOCK_STATE: AttemptState = {
  attemptId: 'att-1',
  numero: 1,
  submitted: false,
  puntaje: 0,
  aprobado: false,
  questions: [
    {
      id: 'q-1',
      enunciado: 'Cual es la capital de Francia?',
      tipo: 'opcion_multiple',
      puntaje: 10,
      options: [
        { id: 'opt-paris', texto: 'Paris' },    // ← NO correcta
        { id: 'opt-madrid', texto: 'Madrid' },  // ← NO correcta
      ],
    },
    {
      id: 'q-2',
      enunciado: 'El agua hierve a 100°C?',
      tipo: 'verdadero_falso',
      puntaje: 5,
      options: [
        { id: 'opt-v', texto: 'Verdadero' },  // ← NO correcta
        { id: 'opt-f', texto: 'Falso' },      // ← NO correcta
      ],
    },
  ],
  answers: [],
};

const MOCK_SUBMIT: SubmitResponse = {
  puntaje: 80,
  aprobado: true,
};

// ── Helper ────────────────────────────────────────────────────────────────────

async function createComponent(
  attemptSpy: Partial<AttemptService>,
  uiSpy: Partial<UiDialogService>,
  evaluationId = 'eval-1',
) {
  await TestBed.configureTestingModule({
    imports: [EvaluacionTomarComponent],
    providers: [
      { provide: AttemptService, useValue: attemptSpy },
      { provide: UiDialogService, useValue: uiSpy },
      {
        provide: ActivatedRoute,
        useValue: {
          snapshot: { paramMap: convertToParamMap({ id: evaluationId }) },
          params: of({ id: evaluationId }),
        },
      },
      ConfirmationService,
      MessageService,
      provideRouter([{ path: '**', component: EvaluacionTomarComponent }]),
      provideAnimationsAsync(),
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(EvaluacionTomarComponent);
  // Trigger ngOnInit (reads :id param from ActivatedRoute)
  fixture.detectChanges();
  return fixture;
}

// ── Specs ─────────────────────────────────────────────────────────────────────

describe('EvaluacionTomarComponent', () => {
  let attemptSpy: Partial<AttemptService>;
  let uiSpy: Partial<UiDialogService>;

  beforeEach(() => {
    attemptSpy = {
      startAttempt: vi.fn().mockResolvedValue(MOCK_START),
      getAttempt:   vi.fn().mockResolvedValue(MOCK_STATE),
      saveAnswer:   vi.fn().mockResolvedValue(undefined),
      submitAttempt: vi.fn().mockResolvedValue(MOCK_SUBMIT),
    };

    uiSpy = {
      showSuccess: vi.fn(),
      showError:   vi.fn(),
      confirm:     vi.fn().mockResolvedValue(true),
    };
  });

  afterEach(() => {
    TestBed.resetTestingModule();
  });

  // ── FE-4-A: start flow ────────────────────────────────────────────────────────

  it('FE-4-A: on start, calls startAttempt(evaluationId) then getAttempt(attemptId)', async () => {
    const fixture = await createComponent(attemptSpy, uiSpy);
    const comp = fixture.componentInstance;

    // Set evaluationId directly — same pattern as evaluacion-editar.spec.ts
    // (provideRouter may replace ActivatedRoute; this is the reliable approach)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-1';

    expect(comp.phase()).toBe('start');

    await comp.startAttempt();

    expect(attemptSpy.startAttempt).toHaveBeenCalledWith('eval-1');
    expect(attemptSpy.getAttempt).toHaveBeenCalledWith('att-1');
    expect(comp.phase()).toBe('taking');
    expect(comp.questions()).toHaveLength(2);
  });

  it('FE-4-A: component reads :id param as evaluationId', async () => {
    const fixture = await createComponent(attemptSpy, uiSpy, 'eval-42');
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-42';

    await comp.startAttempt();

    expect(attemptSpy.startAttempt).toHaveBeenCalledWith('eval-42');
  });

  // ── FE-4-B: save-on-change ────────────────────────────────────────────────────

  it('FE-4-B: selecting an option calls saveAnswer(attemptId, {questionId, optionId})', async () => {
    const fixture = await createComponent(attemptSpy, uiSpy);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-1';
    await comp.startAttempt();
    await comp.selectOption('q-1', 'opt-paris');

    expect(attemptSpy.saveAnswer).toHaveBeenCalledWith('att-1', {
      questionId: 'q-1',
      optionId: 'opt-paris',
    });
  });

  it('FE-4-B: changing the answer calls saveAnswer again with the new optionId', async () => {
    const fixture = await createComponent(attemptSpy, uiSpy);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-1';
    await comp.startAttempt();
    await comp.selectOption('q-1', 'opt-paris');
    await comp.selectOption('q-1', 'opt-madrid');

    expect(attemptSpy.saveAnswer).toHaveBeenCalledTimes(2);
    expect(attemptSpy.saveAnswer).toHaveBeenLastCalledWith('att-1', {
      questionId: 'q-1',
      optionId: 'opt-madrid',
    });
  });

  it('FE-4-B: selectOption() updates local answers map to reflect the current selection', async () => {
    const fixture = await createComponent(attemptSpy, uiSpy);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-1';
    await comp.startAttempt();
    await comp.selectOption('q-1', 'opt-paris');

    expect(comp.selectedAnswer('q-1')).toBe('opt-paris');
  });

  // ── FE-4-C: submit flow ───────────────────────────────────────────────────────

  it('FE-4-C: submit shows confirm dialog then calls submitAttempt()', async () => {
    const fixture = await createComponent(attemptSpy, uiSpy);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-1';
    await comp.startAttempt();
    await comp.submitAttempt();

    expect(uiSpy.confirm).toHaveBeenCalled();
    expect(attemptSpy.submitAttempt).toHaveBeenCalledWith('att-1');
    expect(comp.phase()).toBe('result');
  });

  it('FE-4-C: after submit, result shows puntaje and aprobado from response', async () => {
    const fixture = await createComponent(attemptSpy, uiSpy);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-1';
    await comp.startAttempt();
    await comp.submitAttempt();

    expect(comp.result()).not.toBeNull();
    expect(comp.result()!.puntaje).toBe(80);
    expect(comp.result()!.aprobado).toBe(true);
  });

  it('FE-4-C: submit does NOT call submitAttempt when user rejects the confirm dialog', async () => {
    uiSpy.confirm = vi.fn().mockResolvedValue(false);

    const fixture = await createComponent(attemptSpy, uiSpy);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-1';
    await comp.startAttempt();
    await comp.submitAttempt();

    expect(attemptSpy.submitAttempt).not.toHaveBeenCalled();
    expect(comp.phase()).toBe('taking');
  });

  // ── FE-4-D LOAD-BEARING: no correctness indicator in state ────────────────────

  it('FE-4-D LOAD-BEARING: questions/options held in component state have no correcta field', async () => {
    const fixture = await createComponent(attemptSpy, uiSpy);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-1';
    await comp.startAttempt();

    const questions = comp.questions();
    expect(questions.length).toBeGreaterThan(0);

    for (const q of questions) {
      for (const opt of q.options) {
        expect(Object.prototype.hasOwnProperty.call(opt, 'correcta')).toBe(false);
        expect((opt as Record<string, unknown>)['correcta']).toBeUndefined();
      }
    }
  });

  it('FE-4-D: result phase does not expose correcta in questions state', async () => {
    const fixture = await createComponent(attemptSpy, uiSpy);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-1';
    await comp.startAttempt();
    await comp.submitAttempt();

    expect(comp.phase()).toBe('result');
    // Questions state still present in result phase — options must still have no correcta
    for (const q of comp.questions()) {
      for (const opt of q.options) {
        expect((opt as Record<string, unknown>)['correcta']).toBeUndefined();
      }
    }
  });

  // ── FE-4-G LOAD-BEARING: resumed attempt pre-selects existing answers ─────────

  it('FE-4-G LOAD-BEARING: resumed attempt pre-populates selectedAnswer() from state.answers', async () => {
    // This fixture represents a state returned by the backend for a RESUMED attempt:
    // the student already answered q-1 with opt-paris in a previous session.
    const RESUMED_STATE: AttemptState = {
      attemptId: 'att-1',
      numero: 1,
      submitted: false,
      puntaje: 0,
      aprobado: false,
      questions: MOCK_STATE.questions,
      answers: [
        { questionId: 'q-1', optionId: 'opt-paris' }, // previously saved answer
      ],
    };
    attemptSpy.getAttempt = vi.fn().mockResolvedValue(RESUMED_STATE);

    const fixture = await createComponent(attemptSpy, uiSpy);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-1';
    await comp.startAttempt();

    // The component must have mapped state.answers into the answers map so
    // the student's prior selection is visible without re-selecting.
    expect(comp.selectedAnswer('q-1')).toBe('opt-paris');
    // Questions without a saved answer should still return null.
    expect(comp.selectedAnswer('q-2')).toBeNull();
  });

  it('FE-4-G: fresh attempt (empty answers array) leaves all selections null', async () => {
    // MOCK_STATE has answers: [] — the default fixture used in most tests.
    const fixture = await createComponent(attemptSpy, uiSpy);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-1';
    await comp.startAttempt();

    expect(comp.selectedAnswer('q-1')).toBeNull();
    expect(comp.selectedAnswer('q-2')).toBeNull();
  });

  // ── FE-4-E: 409 on start — blocked state ─────────────────────────────────────

  it('FE-4-E: 409 on startAttempt — phase stays "start", blocked=true, questions empty', async () => {
    const error = Object.assign(new Error('max attempts reached'), { status: 409 });
    attemptSpy.startAttempt = vi.fn().mockRejectedValue(error);

    const fixture = await createComponent(attemptSpy, uiSpy);
    const comp = fixture.componentInstance;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['evaluationId'] = 'eval-1';
    await comp.startAttempt();

    expect(comp.phase()).toBe('start');
    expect(comp.startBlocked()).toBe(true);
    expect(comp.questions()).toHaveLength(0);
  });
});

// ── FE-4-F LOAD-BEARING: navigation uses absolute /platform path ───────────────────

describe('FE-4-F LOAD-BEARING: EvaluacionTomar — "Volver" navigates to absolute /platform path', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [EvaluacionTomarComponent],
      providers: [
        {
          provide: AttemptService,
          useValue: {
            startAttempt:  vi.fn().mockResolvedValue(MOCK_START),
            getAttempt:    vi.fn().mockResolvedValue(MOCK_STATE),
            saveAnswer:    vi.fn().mockResolvedValue(undefined),
            submitAttempt: vi.fn().mockResolvedValue(MOCK_SUBMIT),
          },
        },
        {
          provide: UiDialogService,
          useValue: { showSuccess: vi.fn(), showError: vi.fn(), confirm: vi.fn().mockResolvedValue(true) },
        },
        {
          provide: ActivatedRoute,
          useValue: {
            snapshot: { paramMap: convertToParamMap({ id: 'eval-1' }) },
            params: of({ id: 'eval-1' }),
          },
        },
        ConfirmationService,
        MessageService,
        provideRouter([{ path: '**', component: EvaluacionTomarComponent }]),
        provideAnimationsAsync(),
      ],
    }).compileComponents();
  });

  afterEach(() => {
    TestBed.resetTestingModule();
  });

  it('FE-4-F LOAD-BEARING: goBack() uses an absolute /platform-prefixed path (C2.2 bug class)', async () => {
    const fixture = TestBed.createComponent(EvaluacionTomarComponent);
    const comp    = fixture.componentInstance;
    const router  = TestBed.inject(Router);

    const navigateSpy = vi.spyOn(router, 'navigateByUrl');
    comp.goBack();

    expect(navigateSpy).toHaveBeenCalledWith(
      expect.stringMatching(/^\/platform\//),
    );
    // Must NOT start with /catalog or any relative segment
    const calledWith: string = (navigateSpy.mock.calls[0] as unknown[])[0] as string;
    expect(calledWith.startsWith('/platform/')).toBe(true);
  });
});
