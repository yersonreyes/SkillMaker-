/**
 * platform-layout.component.spec.ts — PlatformLayoutComponent bell & notification panel tests.
 *
 * Zoneless Angular 21 — uses vi.useFakeTimers() for poll interval testing.
 *
 * Covers:
 *  - Renders unread count badge when unreadCount > 0
 *  - Badge is hidden when unreadCount === 0
 *  - ngOnInit loads unread count and notification list
 *  - onNotificationClick calls markRead + navigates for curso_aprobado
 *  - onNotificationClick calls markRead + navigates for certificado_emitido
 *  - onNotificationClick does NOT call markRead for already-read notification
 *  - markAll calls markAllRead and resets unreadCount to 0
 *  - Empty state shown when notifications list is empty
 *  - onBellOpen re-calls getMine (panel-open reload)
 *  - Empty state DOM: .notif-panel__empty / "Sin notificaciones" renders when list is empty
 *  - Poll: setInterval is called on init (via fake timers)
 *  - ngOnDestroy clears the poll interval
 *  - REQ-CRUMB: pageTitle() defaults to "" when no titled route is active
 *  - REQ-CRUMB: .bar__crumb is NOT rendered when pageTitle is empty
 *  - REQ-CRUMB (W-2 closure): NavigationEnd to a child route with data.title sets pageTitle + renders .bar__crumb
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { Component } from '@angular/core';
import { provideRouter, Router } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { signal } from '@angular/core';
import { By } from '@angular/platform-browser';
import { Popover } from 'primeng/popover';

/** Minimal stub component used as a navigation target to provide a real ActivatedRoute child. */
@Component({ selector: 'app-stub-page', standalone: true, template: '<p>stub</p>' })
class StubPageComponent {}

import { PlatformLayoutComponent } from './platform-layout.component';
import { NotificationService } from '@core/services/notificationService/notification.service';
import { AuthService } from '@core/services/authService/auth.service';
import type { NotificationItem } from '@core/services/notificationService/notification.dto';
import type { UserPublic } from '@core/services/authService/auth.res.dto';

const MOCK_NOTIF_UNREAD: NotificationItem = {
  id: 'notif-1',
  tipo: 'curso_aprobado',
  titulo: 'Curso aprobado',
  cuerpo: 'Tu curso fue aprobado',
  leida: false,
  refId: 'course-1',
  creadoEn: '2026-06-01T10:00:00Z',
};

const MOCK_NOTIF_READ: NotificationItem = {
  id: 'notif-2',
  tipo: 'certificado_emitido',
  titulo: 'Certificado emitido',
  cuerpo: 'Tu certificado fue emitido',
  leida: true,
  refId: 'cert-1',
  creadoEn: '2026-06-01T09:00:00Z',
};

const MOCK_NOTIF_RECHAZADO: NotificationItem = {
  id: 'notif-3',
  tipo: 'curso_rechazado',
  titulo: 'Curso rechazado',
  cuerpo: 'Falta bibliografía',
  leida: false,
  refId: 'course-2',
  creadoEn: '2026-06-01T08:00:00Z',
};

describe('PlatformLayoutComponent — bell & notifications', () => {
  let notifServiceSpy: Partial<NotificationService>;
  let authServiceSpy: Partial<AuthService>;

  beforeEach(async () => {
    notifServiceSpy = {
      getMine: vi.fn().mockResolvedValue([MOCK_NOTIF_UNREAD]),
      getUnreadCount: vi.fn().mockResolvedValue(1),
      markRead: vi.fn().mockResolvedValue(undefined),
      markAllRead: vi.fn().mockResolvedValue(undefined),
    };

    authServiceSpy = {
      user: signal<UserPublic | null>({ id: 'u-1', nombre: 'Test User', email: 'test@test.com', roles: [] }),
      userRoles: signal([]),
      logout: vi.fn().mockResolvedValue(undefined),
    };

    await TestBed.configureTestingModule({
      imports: [PlatformLayoutComponent],
      providers: [
        { provide: NotificationService, useValue: notifServiceSpy },
        { provide: AuthService, useValue: authServiceSpy },
        provideRouter([]),
        provideAnimationsAsync(),
        ConfirmationService,
        MessageService,
      ],
    }).compileComponents();
  });

  afterEach(() => {
    vi.useRealTimers();
    TestBed.resetTestingModule();
  });

  // ── Badge visibility ──────────────────────────────────────────────────────────

  it('renders unread count badge when unreadCount > 0', async () => {
    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const badge = fixture.nativeElement.querySelector('.bar__badge');
    expect(badge).not.toBeNull();
    expect(badge.textContent.trim()).toBe('1');
  });

  it('badge is hidden when unreadCount is 0', async () => {
    (notifServiceSpy.getUnreadCount as ReturnType<typeof vi.fn>).mockResolvedValue(0);

    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const badge = fixture.nativeElement.querySelector('.bar__badge');
    expect(badge).toBeNull();
  });

  // ── Init loads ────────────────────────────────────────────────────────────────

  it('ngOnInit loads unread count and notification list', async () => {
    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(notifServiceSpy.getUnreadCount).toHaveBeenCalledTimes(1);
    expect(notifServiceSpy.getMine).toHaveBeenCalledTimes(1);
  });

  // ── onNotificationClick ───────────────────────────────────────────────────────

  it('onNotificationClick calls markRead and navigates for curso_aprobado', async () => {
    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    const router = TestBed.inject(Router);
    const navigateSpy = vi.spyOn(router, 'navigate').mockResolvedValue(true);

    fixture.detectChanges();
    await fixture.whenStable();

    await fixture.componentInstance.onNotificationClick(MOCK_NOTIF_UNREAD);

    expect(notifServiceSpy.markRead).toHaveBeenCalledWith('notif-1');
    expect(navigateSpy).toHaveBeenCalledWith(['/platform/courses', 'course-1']);
  });

  it('onNotificationClick navigates to certificates for certificado_emitido', async () => {
    (notifServiceSpy.getMine as ReturnType<typeof vi.fn>).mockResolvedValue([MOCK_NOTIF_READ]);
    (notifServiceSpy.getUnreadCount as ReturnType<typeof vi.fn>).mockResolvedValue(0);

    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    const router = TestBed.inject(Router);
    const navigateSpy = vi.spyOn(router, 'navigate').mockResolvedValue(true);

    fixture.detectChanges();
    await fixture.whenStable();

    // Already read — markRead NOT called, but still navigates
    await fixture.componentInstance.onNotificationClick(MOCK_NOTIF_READ);

    expect(notifServiceSpy.markRead).not.toHaveBeenCalled();
    expect(navigateSpy).toHaveBeenCalledWith(['/platform/certificates']);
  });

  it('onNotificationClick calls markRead for curso_rechazado and navigates to courses', async () => {
    (notifServiceSpy.getMine as ReturnType<typeof vi.fn>).mockResolvedValue([MOCK_NOTIF_RECHAZADO]);
    (notifServiceSpy.getUnreadCount as ReturnType<typeof vi.fn>).mockResolvedValue(1);

    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    const router = TestBed.inject(Router);
    const navigateSpy = vi.spyOn(router, 'navigate').mockResolvedValue(true);

    fixture.detectChanges();
    await fixture.whenStable();

    await fixture.componentInstance.onNotificationClick(MOCK_NOTIF_RECHAZADO);

    expect(notifServiceSpy.markRead).toHaveBeenCalledWith('notif-3');
    expect(navigateSpy).toHaveBeenCalledWith(['/platform/creator/curso-editar', 'course-2']);
  });

  // ── markAll ───────────────────────────────────────────────────────────────────

  it('markAll calls markAllRead and resets unreadCount to 0', async () => {
    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    await fixture.componentInstance.markAll();

    expect(notifServiceSpy.markAllRead).toHaveBeenCalledTimes(1);
    expect(fixture.componentInstance.unreadCount()).toBe(0);
  });

  // ── Empty state ───────────────────────────────────────────────────────────────

  it('empty state: notifications signal is empty array when getMine returns []', async () => {
    (notifServiceSpy.getMine as ReturnType<typeof vi.fn>).mockResolvedValue([]);
    (notifServiceSpy.getUnreadCount as ReturnType<typeof vi.fn>).mockResolvedValue(0);

    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.componentInstance.notifications()).toHaveLength(0);
  });

  // ── onBellOpen reload (W-1) ───────────────────────────────────────────────────

  it('onBellOpen re-calls getMine to refresh the notification list', async () => {
    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    // Reset spy counts so we only count calls triggered by onBellOpen
    (notifServiceSpy.getMine as ReturnType<typeof vi.fn>).mockClear();

    fixture.componentInstance.onBellOpen();
    await fixture.whenStable();

    expect(notifServiceSpy.getMine).toHaveBeenCalledTimes(1);
  });

  // ── Empty-state DOM (W-2) ─────────────────────────────────────────────────────

  it('empty-state DOM: .notif-panel__empty with "Sin notificaciones" renders when list is empty', async () => {
    (notifServiceSpy.getMine as ReturnType<typeof vi.fn>).mockResolvedValue([]);
    (notifServiceSpy.getUnreadCount as ReturnType<typeof vi.fn>).mockResolvedValue(0);

    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    // Open the bell popover programmatically so the panel DOM is rendered
    const popoverDe = fixture.debugElement.query(By.directive(Popover));
    const popoverInstance = popoverDe.componentInstance as Popover;
    // show() requires a mock event with a currentTarget for positioning
    const bellBtn = fixture.nativeElement.querySelector('.bar__bell') as HTMLElement;
    popoverInstance.show({ currentTarget: bellBtn, target: bellBtn } as unknown as MouseEvent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    // Popover appends its container to document.body
    const emptyEl = document.body.querySelector('.notif-panel__empty');
    expect(emptyEl).not.toBeNull();
    expect(emptyEl!.textContent?.trim()).toBe('Sin notificaciones');

    popoverInstance.hide();
  });

  // ── Poll ──────────────────────────────────────────────────────────────────────

  it('poll: setInterval is set on init and reloads count after 30s', async () => {
    vi.useFakeTimers();

    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    fixture.detectChanges();
    await Promise.resolve();

    (notifServiceSpy.getUnreadCount as ReturnType<typeof vi.fn>).mockClear();

    vi.advanceTimersByTime(30000);
    await Promise.resolve();

    expect(notifServiceSpy.getUnreadCount).toHaveBeenCalledTimes(1);
  });

  it('ngOnDestroy clears the poll interval', async () => {
    vi.useFakeTimers();

    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    fixture.detectChanges();
    await Promise.resolve();

    fixture.destroy();

    (notifServiceSpy.getUnreadCount as ReturnType<typeof vi.fn>).mockClear();
    vi.advanceTimersByTime(30000);
    await Promise.resolve();

    // After destroy, no more poll calls
    expect(notifServiceSpy.getUnreadCount).not.toHaveBeenCalled();
  });

  // ── REQ-CRUMB: pageTitle breadcrumb ──────────────────────────────────────────

  it('REQ-CRUMB: pageTitle() defaults to "" on init with provideRouter([])', async () => {
    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.componentInstance.pageTitle()).toBe('');
  });

  it('REQ-CRUMB: .bar__crumb is NOT rendered when pageTitle is empty', async () => {
    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const crumb = fixture.nativeElement.querySelector('.bar__crumb');
    expect(crumb).toBeNull();
  });
});

// ── REQ-CRUMB W-2 closure: NavigationEnd walker exercises deepestTitle() ─────
// This describe block uses its OWN TestBed configuration with a titled route so
// it does NOT share state with the bell/notification suite above.
//
// WHY a separate describe: provideRouter config differs (adds a titled child route)
// and TestBed.resetTestingModule() in the outer afterEach would interfere.
//
// This test makes the `while (r.firstChild)` walk LOAD-BEARING:
// removing the walker (while r.firstChild loop) causes this test to fail because
// the root ActivatedRoute has no data.title — only its firstChild does.

describe('PlatformLayoutComponent — REQ-CRUMB navigation walker (W-2)', () => {
  let notifServiceSpy: Partial<NotificationService>;
  let authServiceSpy: Partial<AuthService>;

  beforeEach(async () => {
    notifServiceSpy = {
      getMine: vi.fn().mockResolvedValue([]),
      getUnreadCount: vi.fn().mockResolvedValue(0),
      markRead: vi.fn().mockResolvedValue(undefined),
      markAllRead: vi.fn().mockResolvedValue(undefined),
    };

    authServiceSpy = {
      user: signal<UserPublic | null>({ id: 'u-1', nombre: 'Test User', email: 'test@test.com', roles: [] }),
      userRoles: signal([]),
      logout: vi.fn().mockResolvedValue(undefined),
    };

    await TestBed.configureTestingModule({
      imports: [PlatformLayoutComponent, StubPageComponent],
      providers: [
        { provide: NotificationService, useValue: notifServiceSpy },
        { provide: AuthService, useValue: authServiceSpy },
        // Route carries data.title — this is the titled child the walker must find.
        provideRouter([{ path: 'x', component: StubPageComponent, data: { title: 'Test Page' } }]),
        provideAnimationsAsync(),
        ConfirmationService,
        MessageService,
      ],
    }).compileComponents();
  });

  afterEach(() => {
    vi.useRealTimers();
    TestBed.resetTestingModule();
  });

  it('REQ-CRUMB (W-2): NavigationEnd to a titled child route sets pageTitle() and renders .bar__crumb', async () => {
    const fixture = TestBed.createComponent(PlatformLayoutComponent);
    const router  = TestBed.inject(Router);

    // Initial render — no navigation yet, pageTitle should be empty.
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    // Navigate to the titled route.  navigateByUrl returns a Promise that
    // resolves after the navigation completes (NavigationEnd has fired).
    await router.navigateByUrl('/x');
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    // Assert: pageTitle signal was updated by the NavigationEnd handler.
    expect(fixture.componentInstance.pageTitle()).toBe('Test Page');

    // Assert: .bar__crumb is rendered with the page title text.
    const crumb: HTMLElement | null = fixture.nativeElement.querySelector('.bar__crumb');
    expect(crumb).not.toBeNull();
    expect(crumb!.textContent).toContain('Test Page');
  });
});
