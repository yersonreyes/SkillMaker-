/**
 * certificate.service.spec.ts — CertificateService unit tests (Strict TDD — RED → GREEN).
 *
 * Strategy: HttpTestingController intercepts real HTTP calls
 * (same pattern as course-catalog.service.spec.ts).
 *
 * Covers:
 *  - getMyCertificates() sends GET /api/certificates/me; returns array
 *  - getDetail(id) sends GET /api/certificates/:id
 *  - getDownloadUrl(id) sends GET /api/certificates/:id/download; returns url
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { CertificateService } from './certificate.service';
import type {
  CertificateListItem,
  CertificateResponse,
  DownloadURLResponse,
} from './certificate.dto';

const BASE = 'http://localhost:3000/api/certificates';

const MOCK_LIST_ITEM: CertificateListItem = {
  id: 'cert-1',
  courseId: 'course-1',
  courseTitulo: 'Go Avanzado',
  codigo: 'ABCDEFGHIJKLM',
  emitidoEn: '2026-01-01T00:00:00Z',
};

const MOCK_CERT_RESPONSE: CertificateResponse = {
  id: 'cert-1',
  courseId: 'course-1',
  courseTitulo: 'Go Avanzado',
  codigo: 'ABCDEFGHIJKLM',
  emitidoEn: '2026-01-01T00:00:00Z',
};

const MOCK_DOWNLOAD: DownloadURLResponse = {
  url: 'https://minio/certificates/cert-1.pdf?sig=abc',
  expiresAt: '2026-01-01T00:15:00Z',
};

describe('CertificateService', () => {
  let service: CertificateService;
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
    service = TestBed.inject(CertificateService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── getMyCertificates ────────────────────────────────────────────────────────

  it('getMyCertificates() sends GET /api/certificates/me', async () => {
    const promise = service.getMyCertificates();

    const req = httpMock.expectOne(`${BASE}/me`);
    expect(req.request.method).toBe('GET');
    req.flush({ certificates: [MOCK_LIST_ITEM] });

    const result = await promise;
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('cert-1');
    expect(result[0].courseTitulo).toBe('Go Avanzado');
  });

  it('getMyCertificates() returns empty array when backend returns null certificates', async () => {
    const promise = service.getMyCertificates();

    const req = httpMock.expectOne(`${BASE}/me`);
    req.flush({ certificates: null });

    const result = await promise;
    expect(result).toHaveLength(0);
  });

  // ── getDetail ────────────────────────────────────────────────────────────────

  it('getDetail(id) sends GET /api/certificates/:id', async () => {
    const promise = service.getDetail('cert-1');

    const req = httpMock.expectOne(`${BASE}/cert-1`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_CERT_RESPONSE);

    const result = await promise;
    expect(result.id).toBe('cert-1');
    expect(result.codigo).toBe('ABCDEFGHIJKLM');
  });

  // ── getDownloadUrl ────────────────────────────────────────────────────────────

  it('getDownloadUrl(id) sends GET /api/certificates/:id/download', async () => {
    const promise = service.getDownloadUrl('cert-1');

    const req = httpMock.expectOne(`${BASE}/cert-1/download`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_DOWNLOAD);

    const result = await promise;
    expect(result.url).toContain('minio');
    expect(result.expiresAt).toBeDefined();
  });
});
