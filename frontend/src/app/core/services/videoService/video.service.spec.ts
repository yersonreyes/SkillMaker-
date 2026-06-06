/**
 * video.service.spec.ts — VideoService unit tests (Vitest + Angular TestBed).
 * Uses provideHttpClient + provideHttpClientTesting.
 * Covers URL shapes, HTTP methods, and request bodies.
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { VideoService } from './video.service';
import type { VideoItem } from './video.res.dto';

const SECTIONS_BASE = 'http://localhost:3000/api/sections';
const VIDEOS_BASE = 'http://localhost:3000/api/videos';

const MOCK_VIDEO: VideoItem = {
  id: 'vid-1',
  sectionId: 'sec-1',
  titulo: 'Introduccion a Go',
  url: 'https://www.youtube.com/watch?v=abc123',
  proveedor: 'youtube',
  duracionS: 300,
  orden: 0,
  // course-structure-v2 additions
  descripcion: '',
  materiales: [],
  createdAt: '2026-01-01T00:00:00Z',
};

describe('VideoService', () => {
  let service: VideoService;
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
    service = TestBed.inject(VideoService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── create ────────────────────────────────────────────────────────────────────

  it('create() sends POST /api/sections/:sectionId/videos with required fields', async () => {
    const promise = service.create('sec-1', {
      titulo: 'Introduccion a Go',
      url: 'https://www.youtube.com/watch?v=abc123',
      proveedor: 'youtube',
    });

    const req = httpMock.expectOne(`${SECTIONS_BASE}/sec-1/videos`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({
      titulo: 'Introduccion a Go',
      url: 'https://www.youtube.com/watch?v=abc123',
      proveedor: 'youtube',
    });
    req.flush(MOCK_VIDEO, { status: 201, statusText: 'Created' });

    const result = await promise;
    expect(result['id']).toBe('vid-1');
  });

  it('create() sends optional duracionS when provided', async () => {
    const promise = service.create('sec-1', {
      titulo: 'Clase 2',
      url: 'https://vimeo.com/123456',
      proveedor: 'vimeo',
      duracionS: 600,
    });

    const req = httpMock.expectOne(`${SECTIONS_BASE}/sec-1/videos`);
    expect(req.request.body).toEqual({
      titulo: 'Clase 2',
      url: 'https://vimeo.com/123456',
      proveedor: 'vimeo',
      duracionS: 600,
    });
    req.flush({ ...MOCK_VIDEO, duracionS: 600 });

    await promise;
  });

  // ── update ────────────────────────────────────────────────────────────────────

  it('update() sends PATCH /api/videos/:id with partial body', async () => {
    const promise = service.update('vid-1', { titulo: 'Nuevo titulo' });

    const req = httpMock.expectOne(`${VIDEOS_BASE}/vid-1`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ titulo: 'Nuevo titulo' });
    req.flush({ ...MOCK_VIDEO, titulo: 'Nuevo titulo' });

    const result = await promise;
    expect(result['titulo']).toBe('Nuevo titulo');
  });

  it('update() sends url and proveedor together for re-validation', async () => {
    const promise = service.update('vid-1', {
      url: 'https://vimeo.com/999',
      proveedor: 'vimeo',
    });

    const req = httpMock.expectOne(`${VIDEOS_BASE}/vid-1`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({
      url: 'https://vimeo.com/999',
      proveedor: 'vimeo',
    });
    req.flush({ ...MOCK_VIDEO, url: 'https://vimeo.com/999', proveedor: 'vimeo' });

    await promise;
  });

  // ── delete ────────────────────────────────────────────────────────────────────

  it('delete() sends DELETE /api/videos/:id', async () => {
    const promise = service.delete('vid-1');

    const req = httpMock.expectOne(`${VIDEOS_BASE}/vid-1`);
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });

    await promise;
  });
});
