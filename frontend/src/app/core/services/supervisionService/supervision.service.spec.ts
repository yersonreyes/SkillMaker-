/**
 * supervision.service.spec.ts — SupervisionService unit tests (Vitest + Angular TestBed).
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { SupervisionService } from './supervision.service';
import type { SupervisionItem } from '../userService/user.res.dto';

const BASE = 'http://localhost:3000/api/supervisions';

const MOCK_SUPERVISION: SupervisionItem = {
  id: 'sup-1',
  supervisorId: 'user-10',
  supervisorName: 'Carlos Supervisor',
  empleadoId: 'user-20',
  empleadoName: 'Maria Empleada',
  creadoEn: '2026-01-01T00:00:00Z',
};

describe('SupervisionService', () => {
  let service: SupervisionService;
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
    service = TestBed.inject(SupervisionService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── getAll ──────────────────────────────────────────────────────────────────

  it('getAll() sends GET /api/supervisions', async () => {
    const promise = service.getAll();

    const req = httpMock.expectOne(BASE);
    expect(req.request.method).toBe('GET');
    req.flush([MOCK_SUPERVISION]);

    const result = await promise;
    expect(result).toHaveLength(1);
    expect(result[0]['id']).toBe('sup-1');
  });

  it('getAll() returns empty array when no supervisions exist', async () => {
    const promise = service.getAll();

    const req = httpMock.expectOne(BASE);
    req.flush([]);

    const result = await promise;
    expect(result).toHaveLength(0);
  });

  // ── create ─────────────────────────────────────────────────────────────────

  it('create() sends POST /api/supervisions with supervisorId and empleadoId', async () => {
    const promise = service.create({ supervisorId: 'user-10', empleadoId: 'user-20' });

    const req = httpMock.expectOne(BASE);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ supervisorId: 'user-10', empleadoId: 'user-20' });
    req.flush(MOCK_SUPERVISION, { status: 201, statusText: 'Created' });

    const result = await promise;
    expect(result['supervisorId']).toBe('user-10');
    expect(result['empleadoId']).toBe('user-20');
  });

  // ── delete ─────────────────────────────────────────────────────────────────

  it('delete() sends DELETE /api/supervisions/:id', async () => {
    const promise = service.delete('sup-1');

    const req = httpMock.expectOne(`${BASE}/sup-1`);
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });

    await promise;
    // resolves without error
  });
});
