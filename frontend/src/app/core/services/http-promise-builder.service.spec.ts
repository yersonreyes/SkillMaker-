/**
 * http-promise-builder.service.spec.ts — HttpPromiseBuilderService unit tests.
 *
 * Strategy: HttpTestingController intercepts real HTTP calls.
 *
 * Covers queryParamArray (Phase 5 — repeated params via .append):
 *  - emits repeated params when values array has multiple entries
 *  - emits single param when values array has one entry
 *  - omits param entirely when values array is empty
 *  - skips falsy/empty string values within the array
 *  - does NOT overwrite earlier values (unlike queryParam which uses .set)
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { HttpPromiseBuilderService } from './http-promise-builder.service';

const TEST_URL = 'http://localhost:3000/api/test';

describe('HttpPromiseBuilderService.queryParamArray', () => {
  let service: HttpPromiseBuilderService;
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
    service = TestBed.inject(HttpPromiseBuilderService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  it('emits repeated params for multiple values (append, not set)', async () => {
    const CAT_A = '11111111-1111-1111-1111-111111111111';
    const CAT_B = '22222222-2222-2222-2222-222222222222';
    const promise = service
      .request()
      .get()
      .url(TEST_URL)
      .queryParamArray('categoria', [CAT_A, CAT_B])
      .send();

    const req = httpMock.expectOne(r => r.url === TEST_URL && r.method === 'GET');
    // HttpParams.getAll returns all values for a repeated key
    const vals = req.request.params.getAll('categoria');
    expect(vals).toEqual([CAT_A, CAT_B]);
    req.flush({});

    await promise;
  });

  it('emits a single param when values array has one entry', async () => {
    const CAT_A = '11111111-1111-1111-1111-111111111111';
    const promise = service
      .request()
      .get()
      .url(TEST_URL)
      .queryParamArray('categoria', [CAT_A])
      .send();

    const req = httpMock.expectOne(r => r.url === TEST_URL && r.method === 'GET');
    expect(req.request.params.getAll('categoria')).toEqual([CAT_A]);
    req.flush({});

    await promise;
  });

  it('omits param entirely when values array is empty', async () => {
    const promise = service
      .request()
      .get()
      .url(TEST_URL)
      .queryParamArray('categoria', [])
      .send();

    const req = httpMock.expectOne(r => r.url === TEST_URL && r.method === 'GET');
    expect(req.request.params.has('categoria')).toBe(false);
    req.flush({});

    await promise;
  });

  it('does NOT overwrite earlier values — each call appends', async () => {
    const CAT_A = '11111111-1111-1111-1111-111111111111';
    const CAT_B = '22222222-2222-2222-2222-222222222222';
    const CAT_C = '33333333-3333-3333-3333-333333333333';
    const promise = service
      .request()
      .get()
      .url(TEST_URL)
      // Two separate calls to queryParamArray — all values must survive
      .queryParamArray('categoria', [CAT_A, CAT_B])
      .queryParamArray('categoria', [CAT_C])
      .send();

    const req = httpMock.expectOne(r => r.url === TEST_URL && r.method === 'GET');
    const vals = req.request.params.getAll('categoria') ?? [];
    expect(vals).toContain(CAT_A);
    expect(vals).toContain(CAT_B);
    expect(vals).toContain(CAT_C);
    req.flush({});

    await promise;
  });
});
