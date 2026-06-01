/**
 * auth-service-harness.ts
 * Factory for configuring TestBed with all dependencies required by AuthService.
 * Uses the MODERN Angular HTTP testing API (provideHttpClient + provideHttpClientTesting),
 * NOT the deprecated HttpClientTestingModule.
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';

import { AuthService } from '@core/services/authService/auth.service';

export interface AuthServiceHarness {
  service: AuthService;
  httpMock: HttpTestingController;
}

/**
 * Creates a fully-configured TestBed environment for AuthService tests.
 * Call this inside a beforeEach block.
 *
 * Responsibilities:
 *  - Clears localStorage (prevents test state bleed)
 *  - Stubs the `google` global (AuthService declares it)
 *  - Configures TestBed with modern DI providers
 *  - Returns the service instance and the HttpTestingController
 */
export function createAuthServiceHarness(): AuthServiceHarness {
  localStorage.clear();

  vi.stubGlobal('google', {
    accounts: {
      id: {
        disableAutoSelect: vi.fn(),
      },
    },
  });

  TestBed.configureTestingModule({
    providers: [
      provideHttpClient(),
      provideHttpClientTesting(),
      provideRouter([]),
    ],
  });

  const service = TestBed.inject(AuthService);
  const httpMock = TestBed.inject(HttpTestingController);

  return { service, httpMock };
}

/**
 * Clears localStorage. Useful in beforeEach to guarantee a clean slate.
 */
export function clearStorage(): void {
  localStorage.clear();
}

/**
 * Tears down the harness after each test.
 * Verifies no unexpected HTTP requests remain, restores real timers,
 * and unstubs all global stubs set via vi.stubGlobal.
 */
export function teardown(httpMock: HttpTestingController): void {
  httpMock.verify();
  vi.useRealTimers();
  vi.unstubAllGlobals();
}
