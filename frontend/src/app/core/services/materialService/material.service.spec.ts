/**
 * material.service.spec.ts — MaterialService unit tests (Vitest + Angular TestBed).
 *
 * Re-keyed in course-structure-v2: materials now belong to videos (not courses).
 * Coverage targets:
 *  - presign()       → POST /api/videos/:videoId/materials/presign
 *  - confirm()       → POST /api/videos/:videoId/materials
 *  - list()          → GET  /api/videos/:videoId/materials
 *  - downloadUrl()   → GET  /api/materials/:materialId/download (flat; no courseId)
 *  - remove()        → DELETE /api/materials/:materialId
 *  - uploadToStorage → LOAD-BEARING: raw XHR, Content-Type set, NO Authorization header.
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { MaterialService } from './material.service';
import type {
  MaterialPresignRequest,
  PresignResponse,
  MaterialConfirmRequest,
  MaterialResponse,
  DownloadResponse,
} from './material.types';

const BASE_VIDEOS = 'http://localhost:3000/api/videos';
const BASE_MATERIALS = 'http://localhost:3000/api/materials';

const MOCK_PRESIGN_RESPONSE: PresignResponse = {
  uploadUrl: 'http://minio:9000/skillmaker-materials/courses/c-1/videos/v-1/materials/uuid-file.pdf?signature=abc',
  key: 'courses/c-1/videos/v-1/materials/uuid-file.pdf',
  expiresAt: '2026-06-03T16:00:00Z',
};

const MOCK_MATERIAL: MaterialResponse = {
  id: 'mat-1',
  nombre: 'slides.pdf',
  mimeType: 'application/pdf',
  tamanoBytes: 5_000_000,
  createdAt: '2026-06-03T15:00:00Z',
};

const MOCK_DOWNLOAD: DownloadResponse = {
  url: 'http://minio:9000/skillmaker-materials/courses/c-1/videos/v-1/materials/uuid-file.pdf?download=1',
  expiresAt: '2026-06-03T16:00:00Z',
};

describe('MaterialService — backend API methods (re-keyed to videoId)', () => {
  let service: MaterialService;
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
    service = TestBed.inject(MaterialService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── presign ───────────────────────────────────────────────────────────────────

  it('presign() sends POST /api/videos/:videoId/materials/presign with correct body', async () => {
    const req_body: MaterialPresignRequest = {
      nombre: 'slides.pdf',
      contentType: 'application/pdf',
      tamanoBytes: 5_000_000,
    };

    const promise = service.presign('v-1', req_body);

    const req = httpMock.expectOne(`${BASE_VIDEOS}/v-1/materials/presign`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(req_body);
    req.flush(MOCK_PRESIGN_RESPONSE, { status: 200, statusText: 'OK' });

    const result = await promise;
    expect(result.uploadUrl).toContain('minio');
    expect(result.key).toContain('videos/v-1/materials');
  });

  // ── confirm ───────────────────────────────────────────────────────────────────

  it('confirm() sends POST /api/videos/:videoId/materials with confirm body', async () => {
    const req_body: MaterialConfirmRequest = {
      key: 'courses/c-1/videos/v-1/materials/uuid-file.pdf',
      nombre: 'slides.pdf',
      contentType: 'application/pdf',
      tamanoBytes: 5_000_000,
    };

    const promise = service.confirm('v-1', req_body);

    const req = httpMock.expectOne(`${BASE_VIDEOS}/v-1/materials`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(req_body);
    req.flush(MOCK_MATERIAL, { status: 201, statusText: 'Created' });

    const result = await promise;
    expect(result.id).toBe('mat-1');
    expect(result.nombre).toBe('slides.pdf');
  });

  // ── list ──────────────────────────────────────────────────────────────────────

  it('list() sends GET /api/videos/:videoId/materials and returns array', async () => {
    const promise = service.list('v-1');

    const req = httpMock.expectOne(`${BASE_VIDEOS}/v-1/materials`);
    expect(req.request.method).toBe('GET');
    req.flush([MOCK_MATERIAL]);

    const result = await promise;
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('mat-1');
  });

  it('list() returns empty array when video has no materials', async () => {
    const promise = service.list('v-1');

    const req = httpMock.expectOne(`${BASE_VIDEOS}/v-1/materials`);
    req.flush([]);

    const result = await promise;
    expect(result).toHaveLength(0);
  });

  // ── downloadUrl ───────────────────────────────────────────────────────────────

  it('downloadUrl() sends GET /api/materials/:materialId/download (flat, no courseId)', async () => {
    const promise = service.downloadUrl('mat-1');

    const req = httpMock.expectOne(`${BASE_MATERIALS}/mat-1/download`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_DOWNLOAD);

    const result = await promise;
    expect(result.url).toContain('download=1');
    expect(result.expiresAt).toBe('2026-06-03T16:00:00Z');
  });

  // ── remove ────────────────────────────────────────────────────────────────────

  it('remove() sends DELETE /api/materials/:materialId', async () => {
    const promise = service.remove('mat-1');

    const req = httpMock.expectOne(`${BASE_MATERIALS}/mat-1`);
    expect(req.request.method).toBe('DELETE');
    req.flush(null, { status: 204, statusText: 'No Content' });

    await promise;
  });
});

// ── uploadToStorage — LOAD-BEARING XHR constraint ────────────────────────────
//
// CRITICAL (spec REQ-FE-UPLOADER + design D7):
//   The PUT to MinIO MUST use raw XMLHttpRequest, NOT HttpClient.
//   The app's JWT interceptor would inject "Authorization: Bearer <token>"
//   into a presigned PUT, breaking the MinIO signature verification.
//   This test suite asserts:
//     1. XHR is used (open/send called on the mock XHR instance).
//     2. Content-Type is set to file.type.
//     3. NO Authorization header is set — ever.
//     4. onprogress fires the callback with the computed percentage.
//     5. Resolve on status 200; reject on onerror.

describe('MaterialService.uploadToStorage — NO-JWT-PUT constraint (LOAD-BEARING)', () => {
  /**
   * Minimal fake XMLHttpRequest that records all calls.
   * We replace window.XMLHttpRequest with this class in each test.
   */
  class FakeXHR {
    static instances: FakeXHR[] = [];

    method = '';
    url = '';
    sentBody: unknown = undefined;
    readonly headers: Record<string, string> = {};
    upload = { onprogress: null as ((e: ProgressEvent) => void) | null };
    onload: (() => void) | null = null;
    onerror: (() => void) | null = null;
    status = 200;

    open(method: string, url: string): void {
      this.method = method;
      this.url = url;
    }

    setRequestHeader(name: string, value: string): void {
      this.headers[name.toLowerCase()] = value;
    }

    send(body: unknown): void {
      this.sentBody = body;
    }

    /** Simulate a successful upload response. */
    simulateLoad(status = 200): void {
      this.status = status;
      this.onload?.();
    }

    /** Simulate a network error. */
    simulateError(): void {
      this.onerror?.();
    }

    /** Simulate progress event. */
    simulateProgress(loaded: number, total: number): void {
      const event = { lengthComputable: true, loaded, total } as ProgressEvent;
      this.upload.onprogress?.(event);
    }

    constructor() {
      FakeXHR.instances.push(this);
    }
  }

  let service: MaterialService;
  let OriginalXHR: typeof XMLHttpRequest;

  beforeEach(() => {
    FakeXHR.instances = [];
    OriginalXHR = window.XMLHttpRequest;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).XMLHttpRequest = FakeXHR;

    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        ConfirmationService,
        MessageService,
      ],
    });
    service = TestBed.inject(MaterialService);
  });

  afterEach(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).XMLHttpRequest = OriginalXHR;
    TestBed.resetTestingModule();
  });

  it('LOAD-BEARING: uploadToStorage uses XHR.open("PUT", uploadUrl) — not HttpClient', async () => {
    const file = new File(['hello'], 'test.pdf', { type: 'application/pdf' });
    const promise = service.uploadToStorage('http://minio/presigned-url', file, vi.fn());

    const xhr = FakeXHR.instances[0];
    expect(xhr).toBeDefined();
    expect(xhr.method).toBe('PUT');
    expect(xhr.url).toBe('http://minio/presigned-url');

    xhr.simulateLoad(200);
    await promise;
  });

  it('LOAD-BEARING: uploadToStorage sets Content-Type to file.type', async () => {
    const file = new File(['hello'], 'test.pdf', { type: 'application/pdf' });
    const promise = service.uploadToStorage('http://minio/presigned-url', file, vi.fn());

    const xhr = FakeXHR.instances[0];
    expect(xhr.headers['content-type']).toBe('application/pdf');

    xhr.simulateLoad(200);
    await promise;
  });

  it('LOAD-BEARING (NO-JWT): uploadToStorage does NOT set Authorization header', async () => {
    // CRITICAL — if this test fails, the JWT interceptor is leaking into the
    // presigned PUT. MinIO will reject the request with 403 (signature mismatch).
    const file = new File(['hello'], 'test.pdf', { type: 'application/pdf' });
    const promise = service.uploadToStorage('http://minio/presigned-url', file, vi.fn());

    const xhr = FakeXHR.instances[0];
    // Authorization must be absent — raw XHR bypasses Angular's HTTP interceptors.
    expect(xhr.headers['authorization']).toBeUndefined();

    xhr.simulateLoad(200);
    await promise;
  });

  it('uploadToStorage sends the File object as XHR body', async () => {
    const file = new File(['hello'], 'test.pdf', { type: 'application/pdf' });
    const promise = service.uploadToStorage('http://minio/presigned-url', file, vi.fn());

    const xhr = FakeXHR.instances[0];
    expect(xhr.sentBody).toBe(file);

    xhr.simulateLoad(200);
    await promise;
  });

  it('uploadToStorage fires onProgress callback with computed percentage', async () => {
    const progressSpy = vi.fn();
    const file = new File(['hello'], 'test.pdf', { type: 'application/pdf' });
    const promise = service.uploadToStorage('http://minio/presigned-url', file, progressSpy);

    const xhr = FakeXHR.instances[0];
    xhr.simulateProgress(50, 100);
    expect(progressSpy).toHaveBeenCalledWith(50);

    xhr.simulateProgress(75, 100);
    expect(progressSpy).toHaveBeenCalledWith(75);

    xhr.simulateLoad(200);
    await promise;
  });

  it('uploadToStorage resolves when XHR status is 200', async () => {
    const file = new File(['hello'], 'test.pdf', { type: 'application/pdf' });
    const promise = service.uploadToStorage('http://minio/presigned-url', file, vi.fn());

    const xhr = FakeXHR.instances[0];
    xhr.simulateLoad(200);

    await expect(promise).resolves.toBeUndefined();
  });

  it('uploadToStorage rejects when XHR status is 4xx (upload failed)', async () => {
    const file = new File(['hello'], 'test.pdf', { type: 'application/pdf' });
    const promise = service.uploadToStorage('http://minio/presigned-url', file, vi.fn());

    const xhr = FakeXHR.instances[0];
    xhr.simulateLoad(403);

    await expect(promise).rejects.toThrow('upload failed 403');
  });

  it('uploadToStorage rejects on network error (onerror)', async () => {
    const file = new File(['hello'], 'test.pdf', { type: 'application/pdf' });
    const promise = service.uploadToStorage('http://minio/presigned-url', file, vi.fn());

    const xhr = FakeXHR.instances[0];
    xhr.simulateError();

    await expect(promise).rejects.toThrow('network error');
  });
});
