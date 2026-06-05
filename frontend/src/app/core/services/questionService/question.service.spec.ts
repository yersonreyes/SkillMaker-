/**
 * question.service.spec.ts — QuestionService unit tests (Vitest + Angular TestBed).
 * Verifies correct URLs, HTTP verbs, and request bodies.
 * Covers: question create/update/delete + option create/update/delete (all folded in).
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { QuestionService } from './question.service';
import type { QuestionResponse, OptionResponse } from '../evaluationService/evaluation.dto';

const EVALUATIONS_BASE = 'http://localhost:3000/api/evaluations';
const QUESTIONS_BASE   = 'http://localhost:3000/api/questions';
const OPTIONS_BASE     = 'http://localhost:3000/api/options';

const MOCK_QUESTION: QuestionResponse = {
  id: 'q-1',
  evaluationId: 'eval-1',
  enunciado: 'Cual es la capital de Francia?',
  tipo: 'opcion_multiple',
  puntaje: 10,
  orden: 0,
};

const MOCK_OPTION: OptionResponse = {
  id: 'opt-1',
  questionId: 'q-1',
  texto: 'Paris',
  correcta: true,
  orden: 0,
};

describe('QuestionService', () => {
  let service: QuestionService;
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
    service = TestBed.inject(QuestionService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── Questions ─────────────────────────────────────────────────────────────────

  it('create() sends POST /api/evaluations/:evalId/questions with body', async () => {
    const body = { enunciado: 'Pregunta 1', tipo: 'opcion_multiple' as const, puntaje: 10 };
    const promise = service.create('eval-1', body);

    const req = httpMock.expectOne(`${EVALUATIONS_BASE}/eval-1/questions`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(body);
    req.flush(MOCK_QUESTION, { status: 201, statusText: 'Created' });

    const result = await promise;
    expect(result.id).toBe('q-1');
    expect(result.tipo).toBe('opcion_multiple');
  });

  it('create() sends verdadero_falso tipo', async () => {
    const body = { enunciado: 'El cielo es azul?', tipo: 'verdadero_falso' as const, puntaje: 5 };
    const promise = service.create('eval-1', body);

    const req = httpMock.expectOne(`${EVALUATIONS_BASE}/eval-1/questions`);
    expect(req.request.body).toEqual(body);
    req.flush({ ...MOCK_QUESTION, tipo: 'verdadero_falso', id: 'q-2' }, { status: 201, statusText: 'Created' });

    const result = await promise;
    expect(result.tipo).toBe('verdadero_falso');
  });

  it('update() sends PATCH /api/questions/:id with partial body', async () => {
    const body = { puntaje: 20 };
    const promise = service.update('q-1', body);

    const req = httpMock.expectOne(`${QUESTIONS_BASE}/q-1`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual(body);
    req.flush({ ...MOCK_QUESTION, puntaje: 20 });

    const result = await promise;
    expect(result.puntaje).toBe(20);
  });

  it('delete() sends DELETE /api/questions/:id and resolves void', async () => {
    const promise = service.delete('q-1');

    const req = httpMock.expectOne(`${QUESTIONS_BASE}/q-1`);
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });

    await promise;
  });

  // ── Options ───────────────────────────────────────────────────────────────────

  it('createOption() sends POST /api/questions/:questionId/options with body', async () => {
    const body = { texto: 'Paris', correcta: true };
    const promise = service.createOption('q-1', body);

    const req = httpMock.expectOne(`${QUESTIONS_BASE}/q-1/options`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(body);
    req.flush(MOCK_OPTION, { status: 201, statusText: 'Created' });

    const result = await promise;
    expect(result.id).toBe('opt-1');
    expect(result.correcta).toBe(true);
  });

  it('createOption() sends correcta=false option', async () => {
    const body = { texto: 'Madrid', correcta: false };
    const promise = service.createOption('q-1', body);

    const req = httpMock.expectOne(`${QUESTIONS_BASE}/q-1/options`);
    expect(req.request.body).toEqual(body);
    req.flush({ ...MOCK_OPTION, id: 'opt-2', texto: 'Madrid', correcta: false }, { status: 201, statusText: 'Created' });

    const result = await promise;
    expect(result.correcta).toBe(false);
  });

  it('updateOption() sends PATCH /api/options/:optionId with partial body', async () => {
    const body = { correcta: true };
    const promise = service.updateOption('opt-1', body);

    const req = httpMock.expectOne(`${OPTIONS_BASE}/opt-1`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual(body);
    req.flush({ ...MOCK_OPTION, correcta: true });

    const result = await promise;
    expect(result.correcta).toBe(true);
  });

  it('updateOption() sends texto-only patch', async () => {
    const body = { texto: 'Roma' };
    const promise = service.updateOption('opt-1', body);

    const req = httpMock.expectOne(`${OPTIONS_BASE}/opt-1`);
    expect(req.request.body).toEqual(body);
    req.flush({ ...MOCK_OPTION, texto: 'Roma' });

    const result = await promise;
    expect(result.texto).toBe('Roma');
  });

  it('deleteOption() sends DELETE /api/options/:optionId and resolves void', async () => {
    const promise = service.deleteOption('opt-1');

    const req = httpMock.expectOne(`${OPTIONS_BASE}/opt-1`);
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });

    await promise;
  });
});
