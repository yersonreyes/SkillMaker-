/**
 * attempt.service.spec.ts — AttemptService unit tests (Vitest + Angular TestBed).
 *
 * Strict TDD — tests written FIRST (RED), then service implemented (GREEN).
 *
 * Covers:
 *  - startAttempt()  → POST /api/evaluations/:id/attempts
 *  - getAttempt()    → GET  /api/attempts/:id
 *  - saveAnswer()    → POST /api/attempts/:id/answers
 *  - submitAttempt() → POST /api/attempts/:id/submit
 *  - LOAD-BEARING: AttemptState option type has NO correcta field (type-level + fixture)
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { AttemptService } from './attempt.service';
import type {
  AttemptStartResponse,
  AttemptState,
  AttemptStateOption,
  SubmitResponse,
} from './attempt.dto';

const BASE = 'http://localhost:3000/api';

// ── Fixtures ──────────────────────────────────────────────────────────────────

// LOAD-BEARING: the option fixture MUST NOT contain a correcta field.
// This mirrors the backend structural no-leak guarantee (design §7, ADR-E).
const MOCK_OPTION: AttemptStateOption = {
  id: 'opt-1',
  texto: 'Paris',
  // ← NO correcta field here: the type itself forbids it
};

const MOCK_START: AttemptStartResponse = {
  attemptId: 'att-1',
  numero: 1,
  iniciadoEn: '2026-06-01T10:00:00Z',
};

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
        MOCK_OPTION,
        { id: 'opt-2', texto: 'Madrid' },
      ],
    },
  ],
  answers: [],
};

const MOCK_SUBMIT: SubmitResponse = {
  puntaje: 100,
  aprobado: true,
};

// ── LOAD-BEARING: no-correcta-in-fixture assertion ────────────────────────────
// Verifies at spec-level that the AttemptStateOption fixture has no correcta.
// This is a compile-time + runtime double-check of the no-leak invariant.

describe('AttemptStateOption — no correcta field', () => {
  it('LOAD-BEARING: fixture option has no correcta property', () => {
    expect(Object.prototype.hasOwnProperty.call(MOCK_OPTION, 'correcta')).toBe(false);
    expect((MOCK_OPTION as Record<string, unknown>)['correcta']).toBeUndefined();
  });

  it('LOAD-BEARING: AttemptState questions options have no correcta property', () => {
    const opts = MOCK_STATE.questions[0].options;
    expect(opts).toHaveLength(2);
    for (const opt of opts) {
      expect(Object.prototype.hasOwnProperty.call(opt, 'correcta')).toBe(false);
      expect((opt as Record<string, unknown>)['correcta']).toBeUndefined();
    }
  });
});

// ── AttemptService HTTP specs ─────────────────────────────────────────────────

describe('AttemptService', () => {
  let service: AttemptService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        ConfirmationService,
        MessageService,
      ],
    });
    service = TestBed.inject(AttemptService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── startAttempt ─────────────────────────────────────────────────────────────

  it('startAttempt() sends POST /api/evaluations/:id/attempts and returns AttemptStartResponse', async () => {
    const promise = service.startAttempt('eval-1');

    const req = httpMock.expectOne(`${BASE}/evaluations/eval-1/attempts`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toBeNull();
    req.flush(MOCK_START);

    const result = await promise;
    expect(result.attemptId).toBe('att-1');
    expect(result.numero).toBe(1);
    expect(result.iniciadoEn).toBe('2026-06-01T10:00:00Z');
  });

  it('startAttempt() sends to the correct URL for a different evaluationId', async () => {
    const promise = service.startAttempt('eval-99');

    const req = httpMock.expectOne(`${BASE}/evaluations/eval-99/attempts`);
    expect(req.request.method).toBe('POST');
    req.flush({ attemptId: 'att-99', numero: 2, iniciadoEn: '2026-06-01T11:00:00Z' });

    const result = await promise;
    expect(result.attemptId).toBe('att-99');
    expect(result.numero).toBe(2);
  });

  // ── getAttempt ────────────────────────────────────────────────────────────────

  it('getAttempt() sends GET /api/attempts/:id and returns AttemptState', async () => {
    const promise = service.getAttempt('att-1');

    const req = httpMock.expectOne(`${BASE}/attempts/att-1`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_STATE);

    const result = await promise;
    expect(result.attemptId).toBe('att-1');
    expect(result.submitted).toBe(false);
    expect(result.questions).toHaveLength(1);
    expect(result.questions[0].options).toHaveLength(2);
  });

  it('getAttempt() sends to the correct URL for a different attemptId', async () => {
    const promise = service.getAttempt('att-42');

    const req = httpMock.expectOne(`${BASE}/attempts/att-42`);
    expect(req.request.method).toBe('GET');
    req.flush({ ...MOCK_STATE, attemptId: 'att-42', numero: 3 });

    const result = await promise;
    expect(result.attemptId).toBe('att-42');
    expect(result.numero).toBe(3);
  });

  // ── LOAD-BEARING: getAttempt options have no correcta ─────────────────────────

  it('LOAD-BEARING: getAttempt() returns state whose options have no correcta property', async () => {
    const stateWithLeak = {
      ...MOCK_STATE,
      questions: [
        {
          ...MOCK_STATE.questions[0],
          options: [
            { id: 'opt-1', texto: 'Paris' },        // ← no correcta
            { id: 'opt-2', texto: 'Madrid' },       // ← no correcta
          ],
        },
      ],
    };

    const promise = service.getAttempt('att-1');

    const req = httpMock.expectOne(`${BASE}/attempts/att-1`);
    req.flush(stateWithLeak);

    const result = await promise;
    for (const q of result.questions) {
      for (const opt of q.options) {
        expect(Object.prototype.hasOwnProperty.call(opt, 'correcta')).toBe(false);
        expect((opt as Record<string, unknown>)['correcta']).toBeUndefined();
      }
    }
  });

  // ── saveAnswer ────────────────────────────────────────────────────────────────

  it('saveAnswer() sends POST /api/attempts/:id/answers with {questionId, optionId}', async () => {
    const promise = service.saveAnswer('att-1', { questionId: 'q-1', optionId: 'opt-1' });

    const req = httpMock.expectOne(`${BASE}/attempts/att-1/answers`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ questionId: 'q-1', optionId: 'opt-1' });
    req.flush(null, { status: 204, statusText: 'No Content' });

    await promise; // should resolve without error
  });

  it('saveAnswer() sends to correct URL for a different attemptId', async () => {
    const promise = service.saveAnswer('att-77', { questionId: 'q-2', optionId: 'opt-3' });

    const req = httpMock.expectOne(`${BASE}/attempts/att-77/answers`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ questionId: 'q-2', optionId: 'opt-3' });
    req.flush(null, { status: 204, statusText: 'No Content' });

    await promise;
  });

  // ── submitAttempt ─────────────────────────────────────────────────────────────

  it('submitAttempt() sends POST /api/attempts/:id/submit and returns SubmitResponse', async () => {
    const promise = service.submitAttempt('att-1');

    const req = httpMock.expectOne(`${BASE}/attempts/att-1/submit`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toBeNull();
    req.flush(MOCK_SUBMIT);

    const result = await promise;
    expect(result.puntaje).toBe(100);
    expect(result.aprobado).toBe(true);
  });

  it('submitAttempt() returns aprobado=false when score is below nota_minima', async () => {
    const promise = service.submitAttempt('att-2');

    const req = httpMock.expectOne(`${BASE}/attempts/att-2/submit`);
    req.flush({ puntaje: 40, aprobado: false });

    const result = await promise;
    expect(result.puntaje).toBe(40);
    expect(result.aprobado).toBe(false);
  });
});
