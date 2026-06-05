/**
 * auth.service.spec.ts — AuthService unit tests.
 * Uses the shared harness from src/testing/auth-service-harness.ts.
 *
 * Test bed: Angular TestBed + provideHttpClient/provideHttpClientTesting (modern API).
 * Fake timers via vi.useFakeTimers().
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';

import {
  createAuthServiceHarness,
  teardown,
  type AuthServiceHarness,
} from '../../../../testing/auth-service-harness';

// ── Helpers ──────────────────────────────────────────────────────────────────

/** Builds a minimal JWT with a given exp unix timestamp (no real signature). */
function buildJwt(exp: number): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = btoa(
    JSON.stringify({
      sub: 'user-123',
      email: 'test@example.com',
      nombre: 'Test User',
      roles: ['alumno'],
      exp,
      iat: Math.floor(Date.now() / 1000),
    }),
  );
  return `${header}.${payload}.fake-signature`;
}

const REFRESH_URL = 'http://localhost:3000/api/auth/refresh';

const MOCK_LOGIN_RESPONSE = {
  access_token: buildJwt(Math.floor(Date.now() / 1000) + 3600),
  refresh_token: 'rt-opaque-plain',
  expires_at: new Date(Date.now() + 3600 * 1000).toISOString(),
  user: {
    id: 'user-123',
    email: 'test@example.com',
    nombre: 'Test User',
    roles: ['alumno' as const],
  },
};

// ── Specs ─────────────────────────────────────────────────────────────────────

describe('AuthService', () => {
  let harness: AuthServiceHarness;

  beforeEach(() => {
    harness = createAuthServiceHarness();
    // Flush any HTTP request triggered by restoreSession() during construction.
    harness.httpMock.match(() => true).forEach(r => r.flush({}, { status: 404, statusText: 'Not Found' }));
  });

  afterEach(() => {
    teardown(harness.httpMock);
    TestBed.resetTestingModule();
  });

  // ── AC3 Scenario 1: token storage and expiry detection ──────────────────

  it('isTokenExpired returns false before exp and true after exp has passed', () => {
    vi.useFakeTimers();

    const nowSeconds = Math.floor(Date.now() / 1000);
    // JWT that expires in 2 seconds from now.
    const expInTwoSeconds = nowSeconds + 2;
    const jwt = buildJwt(expInTwoSeconds);

    localStorage.setItem('auth_token', jwt);

    // Before expiry: should NOT be expired.
    expect(harness.service.isTokenExpired()).toBe(false);

    // Advance time by 3 seconds — token should now be expired.
    vi.advanceTimersByTime(3000);
    expect(harness.service.isTokenExpired()).toBe(true);
  });

  // ── AC3 Scenario 2: refresh clears expired state ────────────────────────

  it('refresh() posts to /auth/refresh and clears sessionExpired signal', async () => {
    // Arrange: manually set a refresh token so _doRefresh finds it.
    localStorage.setItem('auth_refresh_token', 'rt-opaque-plain');

    // Mark session as expired before calling refresh.
    harness.service.sessionExpired.set(true);
    expect(harness.service.sessionExpired()).toBe(true);

    // Act: call refresh() and flush the HTTP mock.
    const refreshPromise = harness.service.refresh();
    const req = harness.httpMock.expectOne(REFRESH_URL);

    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({ refreshToken: 'rt-opaque-plain' });

    req.flush(MOCK_LOGIN_RESPONSE);
    await refreshPromise;

    // Assert: sessionExpired is cleared after a successful refresh.
    expect(harness.service.sessionExpired()).toBe(false);
    expect(harness.service.getToken()).toBe(MOCK_LOGIN_RESPONSE.access_token);
  });
});
