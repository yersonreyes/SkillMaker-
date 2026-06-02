/**
 * user-management.component.spec.ts — Component unit tests (Vitest + Angular TestBed).
 *
 * Covers:
 *  - Role delta computation (selected vs. original → {add, remove})
 *  - Confirm dialog appears when the `administrador` role is touched
 *  - onLazyLoad event → correct page/size calculation passed to service
 *  - Deactivate action triggers confirm dialog
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { UserManagementComponent } from './user-management.component';
import { UserService } from '@core/services/userService/user.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { Page, UserListItem } from '@core/services/userService/user.res.dto';

// ── Helpers ──────────────────────────────────────────────────────────────────

function buildPage(items: UserListItem[]): Page<UserListItem> {
  return { items, page: 1, size: 20, total: items.length, totalPages: 1 };
}

const MOCK_USERS: UserListItem[] = [
  { id: 'user-1', email: 'alice@x.com', nombre: 'Alice', activo: true, roles: ['administrador'] },
  { id: 'user-2', email: 'bob@x.com',   nombre: 'Bob',   activo: true, roles: ['alumno'] },
];

// ── Spec ──────────────────────────────────────────────────────────────────────

describe('UserManagementComponent', () => {
  let userServiceSpy: Partial<UserService>;
  let uiDialogSpy: Partial<UiDialogService>;

  beforeEach(async () => {
    userServiceSpy = {
      getAll: vi.fn().mockResolvedValue(buildPage(MOCK_USERS)),
      updateRoles: vi.fn().mockResolvedValue(MOCK_USERS[0]),
      setActive: vi.fn().mockResolvedValue({ ...MOCK_USERS[1], activo: false }),
    };

    uiDialogSpy = {
      confirm: vi.fn().mockResolvedValue(true),
      confirmDelete: vi.fn().mockResolvedValue(true),
      showSuccess: vi.fn(),
      showError: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [UserManagementComponent],
      providers: [
        { provide: UserService, useValue: userServiceSpy },
        { provide: UiDialogService, useValue: uiDialogSpy },
        ConfirmationService,
        MessageService,
        provideRouter([]),
        provideAnimationsAsync(),
      ],
    }).compileComponents();
  });

  afterEach(() => {
    TestBed.resetTestingModule();
  });

  // ── F1-lazy: onLazyLoad maps first/rows → page/size ────────────────────────

  it('onLazyLoad with first=20, rows=20 calls getAll with page=2, size=20', async () => {
    const fixture = TestBed.createComponent(UserManagementComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    // Simulate table lazy load event: first=20, rows=20 → page 2
    await comp.onLazyLoad({ first: 20, rows: 20 });

    expect(userServiceSpy.getAll).toHaveBeenCalledWith(
      expect.objectContaining({ page: 2, size: 20 }),
    );
  });

  it('onLazyLoad with first=0, rows=10 calls getAll with page=1, size=10', async () => {
    const fixture = TestBed.createComponent(UserManagementComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    await comp.onLazyLoad({ first: 0, rows: 10 });

    expect(userServiceSpy.getAll).toHaveBeenCalledWith(
      expect.objectContaining({ page: 1, size: 10 }),
    );
  });

  // ── Role delta: computeRoleDelta ───────────────────────────────────────────

  it('computeRoleDelta returns empty add/remove when roles unchanged', () => {
    const fixture = TestBed.createComponent(UserManagementComponent);
    const comp = fixture.componentInstance;

    const delta = comp.computeRoleDelta(['alumno', 'supervisor'], ['alumno', 'supervisor']);
    expect(delta.add).toHaveLength(0);
    expect(delta.remove).toHaveLength(0);
  });

  it('computeRoleDelta returns add=[supervisor] when supervisor is newly selected', () => {
    const fixture = TestBed.createComponent(UserManagementComponent);
    const comp = fixture.componentInstance;

    const delta = comp.computeRoleDelta(['alumno'], ['alumno', 'supervisor']);
    expect(delta.add).toContain('supervisor');
    expect(delta.remove).toHaveLength(0);
  });

  it('computeRoleDelta returns remove=[creador] when creador is deselected', () => {
    const fixture = TestBed.createComponent(UserManagementComponent);
    const comp = fixture.componentInstance;

    const delta = comp.computeRoleDelta(['alumno', 'creador'], ['alumno']);
    expect(delta.remove).toContain('creador');
    expect(delta.add).toHaveLength(0);
  });

  // ── F1-confirm-admin: confirm dialog when administrador role is touched ─────

  it('saveRoles shows confirm dialog when administrador role is in the delta', async () => {
    const fixture = TestBed.createComponent(UserManagementComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    // Open dialog for user-1 who IS an administrador
    await comp.openEditDialog(MOCK_USERS[0]);
    // Now remove administrador from selected roles
    await comp.saveRoles('user-1', ['administrador'], []);

    expect(uiDialogSpy.confirm).toHaveBeenCalled();
  });

  it('saveRoles shows confirm dialog when administrador is being added', async () => {
    const fixture = TestBed.createComponent(UserManagementComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    // user-2 does not have administrador; we add it
    await comp.saveRoles('user-2', [], ['administrador']);

    expect(uiDialogSpy.confirm).toHaveBeenCalled();
  });

  it('saveRoles skips confirm dialog when administrador role is not touched', async () => {
    const fixture = TestBed.createComponent(UserManagementComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    // Only changing alumno → supervisor, no administrador
    await comp.saveRoles('user-2', ['supervisor'], []);

    expect(uiDialogSpy.confirm).not.toHaveBeenCalled();
    expect(userServiceSpy.updateRoles).toHaveBeenCalled();
  });

  it('saveRoles does NOT call updateRoles when user rejects the confirm dialog', async () => {
    uiDialogSpy.confirm = vi.fn().mockResolvedValue(false);

    const fixture = TestBed.createComponent(UserManagementComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    await comp.saveRoles('user-1', ['administrador'], []);

    expect(userServiceSpy.updateRoles).not.toHaveBeenCalled();
  });

  // ── Deactivate action ──────────────────────────────────────────────────────

  it('toggleActive shows confirm dialog and calls setActive on confirm', async () => {
    const fixture = TestBed.createComponent(UserManagementComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    await comp.toggleActive(MOCK_USERS[1]);

    expect(uiDialogSpy.confirm).toHaveBeenCalled();
    expect(userServiceSpy.setActive).toHaveBeenCalledWith('user-2', false);
  });

  it('toggleActive does NOT call setActive when user rejects confirm', async () => {
    uiDialogSpy.confirm = vi.fn().mockResolvedValue(false);

    const fixture = TestBed.createComponent(UserManagementComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    await comp.toggleActive(MOCK_USERS[1]);

    expect(userServiceSpy.setActive).not.toHaveBeenCalled();
  });
});
