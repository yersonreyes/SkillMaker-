/**
 * notification.service.spec.ts — NotificationService unit tests (Strict TDD — RED → GREEN).
 *
 * Strategy: HttpTestingController intercepts real HTTP calls
 * (same pattern as certificate.service.spec.ts).
 *
 * Covers:
 *  - getMine(page, size) sends GET /api/notifications/me?page=&size=; returns items array
 *  - getUnreadCount() sends GET /api/notifications/me/unread-count; returns number
 *  - markRead(id) sends PATCH /api/notifications/:id/read
 *  - markAllRead() sends PATCH /api/notifications/me/read-all
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { NotificationService } from './notification.service';
import type { NotificationItem, NotificationListResponse, UnreadCountResponse } from './notification.dto';

const BASE = 'http://localhost:3000/api/notifications';

const MOCK_NOTIF: NotificationItem = {
  id: 'notif-1',
  tipo: 'curso_aprobado',
  titulo: 'Curso aprobado',
  cuerpo: 'Tu curso fue aprobado',
  leida: false,
  refId: 'course-1',
  creadoEn: '2026-06-01T10:00:00Z',
};

const MOCK_LIST_RESPONSE: NotificationListResponse = {
  items: [MOCK_NOTIF],
  page: 1,
  size: 10,
  total: 1,
  totalPages: 1,
};

const MOCK_UNREAD: UnreadCountResponse = {
  unread: 3,
};

describe('NotificationService', () => {
  let service: NotificationService;
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
    service = TestBed.inject(NotificationService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── getMine ──────────────────────────────────────────────────────────────────

  it('getMine() sends GET /api/notifications/me with page and size params', async () => {
    const promise = service.getMine(1, 10);

    const req = httpMock.expectOne(r => r.url === `${BASE}/me`);
    expect(req.request.method).toBe('GET');
    expect(req.request.params.get('page')).toBe('1');
    expect(req.request.params.get('size')).toBe('10');
    req.flush(MOCK_LIST_RESPONSE);

    const result = await promise;
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('notif-1');
    expect(result[0].tipo).toBe('curso_aprobado');
  });

  it('getMine() returns empty array when backend returns null items', async () => {
    const promise = service.getMine(1, 10);

    const req = httpMock.expectOne(r => r.url === `${BASE}/me`);
    req.flush({ items: null, page: 1, size: 10, total: 0, totalPages: 0 });

    const result = await promise;
    expect(result).toHaveLength(0);
  });

  it('getMine() uses default page=1 and size=20 when called without arguments', async () => {
    const promise = service.getMine();

    const req = httpMock.expectOne(r => r.url === `${BASE}/me`);
    expect(req.request.params.get('page')).toBe('1');
    expect(req.request.params.get('size')).toBe('20');
    req.flush(MOCK_LIST_RESPONSE);

    await promise;
  });

  // ── getUnreadCount ────────────────────────────────────────────────────────────

  it('getUnreadCount() sends GET /api/notifications/me/unread-count', async () => {
    const promise = service.getUnreadCount();

    const req = httpMock.expectOne(`${BASE}/me/unread-count`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_UNREAD);

    const result = await promise;
    expect(result).toBe(3);
  });

  it('getUnreadCount() returns 0 when backend returns null/missing unread', async () => {
    const promise = service.getUnreadCount();

    const req = httpMock.expectOne(`${BASE}/me/unread-count`);
    req.flush({});

    const result = await promise;
    expect(result).toBe(0);
  });

  // ── markRead ─────────────────────────────────────────────────────────────────

  it('markRead(id) sends PATCH /api/notifications/:id/read', async () => {
    const promise = service.markRead('notif-1');

    const req = httpMock.expectOne(`${BASE}/notif-1/read`);
    expect(req.request.method).toBe('PATCH');
    req.flush({ ok: true });

    await promise;
  });

  // ── markAllRead ───────────────────────────────────────────────────────────────

  it('markAllRead() sends PATCH /api/notifications/me/read-all', async () => {
    const promise = service.markAllRead();

    const req = httpMock.expectOne(`${BASE}/me/read-all`);
    expect(req.request.method).toBe('PATCH');
    req.flush({ ok: true });

    await promise;
  });
});
