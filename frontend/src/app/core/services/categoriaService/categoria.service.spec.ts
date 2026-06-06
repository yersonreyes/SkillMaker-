/**
 * categoria.service.spec.ts — CategoriaService unit tests (Vitest + Angular TestBed).
 *
 * Coverage targets:
 *  - getAll() → GET /api/categorias → CategoriaItem[]
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { CategoriaService } from './categoria.service';
import type { CategoriaItem } from './categoria.dto';

const BASE = 'http://localhost:3000/api/categorias';

const MOCK_CATEGORIAS: CategoriaItem[] = [
  { id: 'cat-1', nombre: 'Frontend', slug: 'frontend' },
  { id: 'cat-2', nombre: 'Backend', slug: 'backend' },
  { id: 'cat-3', nombre: 'DevOps', slug: 'devops' },
];

describe('CategoriaService', () => {
  let service: CategoriaService;
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
    service = TestBed.inject(CategoriaService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  it('getAll() sends GET /api/categorias and returns CategoriaItem array', async () => {
    const promise = service.getAll();

    const req = httpMock.expectOne(BASE);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_CATEGORIAS, { status: 200, statusText: 'OK' });

    const result = await promise;
    expect(result).toHaveLength(3);
    expect(result[0].id).toBe('cat-1');
    expect(result[0].nombre).toBe('Frontend');
    expect(result[0].slug).toBe('frontend');
  });

  it('getAll() returns empty array when no categorias exist', async () => {
    const promise = service.getAll();

    const req = httpMock.expectOne(BASE);
    req.flush([], { status: 200, statusText: 'OK' });

    const result = await promise;
    expect(result).toHaveLength(0);
  });
});
