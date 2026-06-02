/**
 * supervision.component.spec.ts — Component unit tests (Vitest + Angular TestBed).
 *
 * Covers:
 *  - F2-assign: create supervision calls POST with correct ids
 *  - F2-conflict-message: 409 error is surfaced as a human-readable message
 *  - Remove supervision triggers confirm dialog before DELETE
 *  - Self-supervision validation (supervisorId === empleadoId) prevented client-side
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { HttpErrorResponse } from '@angular/common/http';

import { SupervisionComponent } from './supervision.component';
import { SupervisionService } from '@core/services/supervisionService/supervision.service';
import { UserService } from '@core/services/userService/user.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { SupervisionItem } from '@core/services/userService/user.res.dto';

const MOCK_SUPERVISIONS: SupervisionItem[] = [
  {
    id: 'sup-1',
    supervisorId: 'user-10',
    supervisorName: 'Carlos Supervisor',
    empleadoId: 'user-20',
    empleadoName: 'Maria Empleada',
    creadoEn: '2026-01-01T00:00:00Z',
  },
];

const MOCK_CREATED: SupervisionItem = {
  id: 'sup-2',
  supervisorId: 'user-11',
  supervisorName: 'Nuevo Supervisor',
  empleadoId: 'user-21',
  empleadoName: 'Nuevo Empleado',
  creadoEn: '2026-06-01T00:00:00Z',
};

describe('SupervisionComponent', () => {
  let supervisionServiceSpy: Partial<SupervisionService>;
  let userServiceSpy: Partial<UserService>;
  let uiDialogSpy: Partial<UiDialogService>;

  beforeEach(async () => {
    supervisionServiceSpy = {
      getAll: vi.fn().mockResolvedValue(MOCK_SUPERVISIONS),
      create: vi.fn().mockResolvedValue(MOCK_CREATED),
      delete: vi.fn().mockResolvedValue(undefined),
    };

    userServiceSpy = {
      getAll: vi.fn().mockResolvedValue({
        items: [
          { id: 'user-10', email: 'carlos@x.com', nombre: 'Carlos Supervisor', activo: true, roles: ['supervisor'] },
          { id: 'user-20', email: 'maria@x.com',  nombre: 'Maria Empleada',    activo: true, roles: ['alumno'] },
          { id: 'user-11', email: 'nuevo@x.com',  nombre: 'Nuevo Supervisor',  activo: true, roles: ['supervisor'] },
        ],
        page: 1,
        size: 100,
        total: 3,
        totalPages: 1,
      }),
    };

    uiDialogSpy = {
      confirm: vi.fn().mockResolvedValue(true),
      confirmDelete: vi.fn().mockResolvedValue(true),
      showSuccess: vi.fn(),
      showError: vi.fn(),
      showWarn: vi.fn(),
    };

    await TestBed.configureTestingModule({
      imports: [SupervisionComponent],
      providers: [
        { provide: SupervisionService, useValue: supervisionServiceSpy },
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

  // ── Initial load ────────────────────────────────────────────────────────────

  it('loads supervision list on init', async () => {
    const fixture = TestBed.createComponent(SupervisionComponent);
    fixture.detectChanges();
    // Flush all pending microtasks (mocked Promises resolve in the microtask queue)
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(supervisionServiceSpy.getAll).toHaveBeenCalled();
    expect(fixture.componentInstance.supervisions()).toHaveLength(1);
  });

  // ── F2-assign: create relation ──────────────────────────────────────────────

  it('createSupervision calls service.create with correct ids', async () => {
    const fixture = TestBed.createComponent(SupervisionComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await new Promise(resolve => setTimeout(resolve, 0));

    comp.selectedSupervisorId.set('user-11');
    comp.selectedEmpleadoId.set('user-21');
    await comp.createSupervision();

    expect(supervisionServiceSpy.create).toHaveBeenCalledWith({
      supervisorId: 'user-11',
      empleadoId: 'user-21',
    });
  });

  it('createSupervision adds new relation to the list on success', async () => {
    const fixture = TestBed.createComponent(SupervisionComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await new Promise(resolve => setTimeout(resolve, 0));

    const initialCount = comp.supervisions().length;
    comp.selectedSupervisorId.set('user-11');
    comp.selectedEmpleadoId.set('user-21');
    await comp.createSupervision();

    expect(comp.supervisions().length).toBe(initialCount + 1);
  });

  // ── F2-conflict-message: 409 shown as user-visible message ─────────────────

  it('createSupervision shows error message when API returns 409', async () => {
    const conflictErr = new HttpErrorResponse({
      error: { code: 'SUPERVISION_EXISTS', message: 'El empleado ya tiene un supervisor' },
      status: 409,
    });
    supervisionServiceSpy.create = vi.fn().mockRejectedValue(conflictErr);

    const fixture = TestBed.createComponent(SupervisionComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    comp.selectedSupervisorId.set('user-10');
    comp.selectedEmpleadoId.set('user-20');
    await comp.createSupervision();

    expect(uiDialogSpy.showError).toHaveBeenCalledWith(
      expect.any(String),
      expect.stringContaining('supervisor'),
    );
  });

  // ── Client-side self-supervision guard ─────────────────────────────────────

  it('createSupervision shows warning when supervisorId equals empleadoId', async () => {
    const fixture = TestBed.createComponent(SupervisionComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();

    comp.selectedSupervisorId.set('user-10');
    comp.selectedEmpleadoId.set('user-10');
    await comp.createSupervision();

    expect(supervisionServiceSpy.create).not.toHaveBeenCalled();
    expect(uiDialogSpy.showWarn).toHaveBeenCalled();
  });

  // ── Remove relation ─────────────────────────────────────────────────────────

  it('removeSupervision calls confirmDelete and then service.delete', async () => {
    const fixture = TestBed.createComponent(SupervisionComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    // Seed supervisions directly so we don't depend on async init timing
    comp.supervisions.set([...MOCK_SUPERVISIONS]);

    await comp.removeSupervision(MOCK_SUPERVISIONS[0]);

    expect(uiDialogSpy.confirmDelete).toHaveBeenCalled();
    expect(supervisionServiceSpy.delete).toHaveBeenCalledWith('sup-1');
  });

  it('removeSupervision does NOT call delete when user rejects confirm', async () => {
    uiDialogSpy.confirmDelete = vi.fn().mockResolvedValue(false);

    const fixture = TestBed.createComponent(SupervisionComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    comp.supervisions.set([...MOCK_SUPERVISIONS]);

    await comp.removeSupervision(MOCK_SUPERVISIONS[0]);

    expect(supervisionServiceSpy.delete).not.toHaveBeenCalled();
  });

  it('removeSupervision removes the relation from the list on success', async () => {
    const fixture = TestBed.createComponent(SupervisionComponent);
    const comp = fixture.componentInstance;
    fixture.detectChanges();
    comp.supervisions.set([...MOCK_SUPERVISIONS]);

    const before = comp.supervisions().length;
    await comp.removeSupervision(MOCK_SUPERVISIONS[0]);

    expect(comp.supervisions().length).toBe(before - 1);
  });
});
