/**
 * user.service.spec.ts — UserService unit tests (Vitest + Angular TestBed).
 * Uses provideHttpClient + provideHttpClientTesting (modern API).
 * Mock strategy: HttpTestingController intercepts real HTTP calls.
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { UserService } from './user.service';
import type { UserListItem, UserDetail, Page } from './user.res.dto';

const BASE = 'http://localhost:3000/api/users';

const MOCK_USER_LIST_ITEM: UserListItem = {
  id: 'user-1',
  email: 'alice@example.com',
  nombre: 'Alice',
  activo: true,
  roles: ['alumno'],
};

const MOCK_USER_DETAIL: UserDetail = {
  id: 'user-1',
  email: 'alice@example.com',
  nombre: 'Alice',
  activo: true,
  roles: ['alumno'],
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

const MOCK_PAGE: Page<UserListItem> = {
  items: [MOCK_USER_LIST_ITEM],
  page: 1,
  size: 20,
  total: 1,
  totalPages: 1,
};

describe('UserService', () => {
  let service: UserService;
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
    service = TestBed.inject(UserService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── getAll ──────────────────────────────────────────────────────────────────

  it('getAll() sends GET /api/users with default params', async () => {
    const promise = service.getAll({});

    const req = httpMock.expectOne(r => r.url === BASE && r.method === 'GET');
    expect(req.request.params.has('q')).toBe(false);
    req.flush(MOCK_PAGE);

    const result = await promise;
    expect(result.items).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('getAll() sends page and size query params', async () => {
    const promise = service.getAll({ page: 2, size: 10 });

    const req = httpMock.expectOne(r => r.url === BASE && r.method === 'GET');
    expect(req.request.params.get('page')).toBe('2');
    expect(req.request.params.get('size')).toBe('10');
    req.flush({ ...MOCK_PAGE, page: 2, size: 10 });

    await promise;
  });

  it('getAll() sends q filter param when provided', async () => {
    const promise = service.getAll({ q: 'alice' });

    const req = httpMock.expectOne(r => r.url === BASE && r.method === 'GET');
    expect(req.request.params.get('q')).toBe('alice');
    req.flush(MOCK_PAGE);

    await promise;
  });

  it('getAll() sends role filter param when provided', async () => {
    const promise = service.getAll({ role: 'administrador' });

    const req = httpMock.expectOne(r => r.url === BASE && r.method === 'GET');
    expect(req.request.params.get('role')).toBe('administrador');
    req.flush(MOCK_PAGE);

    await promise;
  });

  it('getAll() sends active filter param when provided', async () => {
    const promise = service.getAll({ active: false });

    const req = httpMock.expectOne(r => r.url === BASE && r.method === 'GET');
    expect(req.request.params.get('active')).toBe('false');
    req.flush(MOCK_PAGE);

    await promise;
  });

  it('getAll() omits empty/undefined params', async () => {
    const promise = service.getAll({ q: '', role: '' });

    const req = httpMock.expectOne(r => r.url === BASE && r.method === 'GET');
    expect(req.request.params.has('q')).toBe(false);
    expect(req.request.params.has('role')).toBe(false);
    req.flush(MOCK_PAGE);

    await promise;
  });

  // ── getById ─────────────────────────────────────────────────────────────────

  it('getById() sends GET /api/users/:id', async () => {
    const promise = service.getById('user-1');

    const req = httpMock.expectOne(`${BASE}/user-1`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_USER_DETAIL);

    const result = await promise;
    expect(result.id).toBe('user-1');
  });

  // ── getMe ──────────────────────────────────────────────────────────────────

  it('getMe() sends GET /api/users/me', async () => {
    const promise = service.getMe();

    const req = httpMock.expectOne(`${BASE}/me`);
    expect(req.request.method).toBe('GET');
    req.flush(MOCK_USER_DETAIL);

    const result = await promise;
    expect(result.email).toBe('alice@example.com');
  });

  // ── updateRoles ─────────────────────────────────────────────────────────────

  it('updateRoles() sends PATCH /api/users/:id/roles with add/remove body', async () => {
    const promise = service.updateRoles('user-1', { add: ['supervisor'], remove: ['alumno'] });

    const req = httpMock.expectOne(`${BASE}/user-1/roles`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ add: ['supervisor'], remove: ['alumno'] });
    req.flush({ ...MOCK_USER_DETAIL, roles: ['supervisor'] });

    const result = await promise;
    expect(result.roles).toContain('supervisor');
  });

  // ── setActive ──────────────────────────────────────────────────────────────

  it('setActive() sends PATCH /api/users/:id/active with active body', async () => {
    const promise = service.setActive('user-1', false);

    const req = httpMock.expectOne(`${BASE}/user-1/active`);
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ active: false });
    req.flush({ ...MOCK_USER_DETAIL, activo: false });

    const result = await promise;
    expect(result.activo).toBe(false);
  });
});
