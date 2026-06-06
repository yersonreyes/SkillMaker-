/**
 * profile.component.spec.ts — ProfileComponent unit tests (Strict TDD — RED → GREEN).
 *
 * C8.1: "Sesiones activas" section — T2.3 (RED) → T2.4 (GREEN).
 *
 * Covers:
 *  - On init calls getMySessions() and stores result in sessions signal
 *  - Empty state when getMySessions() returns []
 *  - onRevoke: confirm → true → calls revokeSession(id) → reloads sessions + shows toast
 *  - onRevoke: confirm → false → revokeSession NOT called
 */
import { TestBed } from '@angular/core/testing';
import { signal } from '@angular/core';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { ProfileComponent } from './profile.component';
import { AuthService } from '@core/services/authService/auth.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { SessionResponse } from '@core/services/authService/auth.res.dto';
import type { UserPublic } from '@core/services/authService/auth.res.dto';

// ── Fixtures ──────────────────────────────────────────────────────────────────

const MOCK_SESSIONS: SessionResponse[] = [
  {
    id: 'sess-001',
    ip: '10.0.0.1',
    userAgent: 'Chrome/120',
    createdAt: '2026-06-01T10:00:00Z',
    expiresAt: '2026-06-08T10:00:00Z',
  },
  {
    id: 'sess-002',
    ip: '192.168.1.5',
    userAgent: 'Firefox/115',
    createdAt: '2026-06-02T08:00:00Z',
    expiresAt: '2026-06-09T08:00:00Z',
  },
];

const MOCK_USER: UserPublic = {
  id: 'user-abc-123',
  email: 'test@example.com',
  nombre: 'Test User',
  roles: ['alumno'],
};

// ── Specs ─────────────────────────────────────────────────────────────────────

describe('ProfileComponent — sessions section (C8.1)', () => {
  let authServiceSpy: Partial<AuthService>;
  let uiDialogSpy: Partial<UiDialogService>;

  beforeEach(async () => {
    authServiceSpy = {
      getMySessions: vi.fn().mockResolvedValue(MOCK_SESSIONS),
      revokeSession: vi.fn().mockResolvedValue(undefined),
      user: signal<UserPublic | null>(MOCK_USER),
    };

    uiDialogSpy = {
      confirm: vi.fn().mockResolvedValue(true),
      showSuccess: vi.fn(),
      showError: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [ProfileComponent],
      providers: [
        { provide: AuthService, useValue: authServiceSpy },
        { provide: UiDialogService, useValue: uiDialogSpy },
        ConfirmationService,
        MessageService,
        provideAnimationsAsync(),
      ],
    }).compileComponents();
  });

  afterEach(() => {
    TestBed.resetTestingModule();
  });

  // ── Init: loads sessions ──────────────────────────────────────────────────

  it('calls getMySessions() on init and stores result in sessions signal', async () => {
    const fixture = TestBed.createComponent(ProfileComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();

    expect(authServiceSpy.getMySessions).toHaveBeenCalled();
    expect(comp.sessions()).toHaveLength(2);
    expect(comp.sessions()[0].id).toBe('sess-001');
  });

  it('loadingSessions is false after sessions load', async () => {
    const fixture = TestBed.createComponent(ProfileComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();

    expect(comp.loadingSessions()).toBe(false);
  });

  // ── Empty state ───────────────────────────────────────────────────────────

  it('sessions signal is empty array when getMySessions() returns []', async () => {
    (authServiceSpy.getMySessions as ReturnType<typeof vi.fn>).mockResolvedValue([]);

    const fixture = TestBed.createComponent(ProfileComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();

    expect(comp.sessions()).toHaveLength(0);
  });

  // ── onRevoke: confirm → true ─────────────────────────────────────────────

  it('onRevoke(): confirm → true → calls revokeSession(id)', async () => {
    const fixture = TestBed.createComponent(ProfileComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();
    await comp.onRevoke(MOCK_SESSIONS[0]);

    expect(uiDialogSpy.confirm).toHaveBeenCalled();
    expect(authServiceSpy.revokeSession).toHaveBeenCalledWith('sess-001');
  });

  it('onRevoke(): confirm → true → shows success toast', async () => {
    const fixture = TestBed.createComponent(ProfileComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();
    await comp.onRevoke(MOCK_SESSIONS[0]);

    expect(uiDialogSpy.showSuccess).toHaveBeenCalled();
  });

  it('onRevoke(): confirm → true → reloads sessions (getMySessions called twice)', async () => {
    (authServiceSpy.getMySessions as ReturnType<typeof vi.fn>)
      .mockResolvedValueOnce(MOCK_SESSIONS)
      .mockResolvedValueOnce([MOCK_SESSIONS[1]]);

    const fixture = TestBed.createComponent(ProfileComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();
    await comp.onRevoke(MOCK_SESSIONS[0]);

    expect(authServiceSpy.getMySessions).toHaveBeenCalledTimes(2);
    expect(comp.sessions()).toHaveLength(1);
  });

  // ── onRevoke: confirm → false ─────────────────────────────────────────────

  it('onRevoke(): confirm → false → revokeSession is NOT called', async () => {
    (uiDialogSpy.confirm as ReturnType<typeof vi.fn>).mockResolvedValue(false);

    const fixture = TestBed.createComponent(ProfileComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();
    await comp.onRevoke(MOCK_SESSIONS[0]);

    expect(authServiceSpy.revokeSession).not.toHaveBeenCalled();
  });
});
