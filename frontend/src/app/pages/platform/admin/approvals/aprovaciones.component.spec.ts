/**
 * aprovaciones.component.spec.ts — AprovacionesComponent unit tests (Vitest + Angular TestBed).
 *
 * Strict TDD — specs written FIRST (RED), then component implemented (GREEN).
 *
 * Covers:
 *  - Renders pending list from ApprovalService.listPending()
 *  - Empty state when no pending courses (uses .empty primitive)
 *  - Approve flow: calls ApprovalService.approve then refreshes list
 *  - Reject flow: reject-requires-comment (empty comment → no call)
 *  - Reject flow: with valid comment → calls ApprovalService.reject then refreshes list
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { AprovacionesComponent } from './aprovaciones.component';
import { ApprovalService } from '@core/services/approvalService/approval.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { PendingItem } from '@core/services/approvalService/approval.dto';

// ── Fixtures ──────────────────────────────────────────────────────────────────

const MOCK_PENDING: PendingItem[] = [
  {
    id: 'course-1',
    titulo: 'Go Avanzado',
    creadorId: 'user-1',
    estado: 'en_revision',
    fechaEnvio: '2026-06-01T10:00:00Z',
  },
  {
    id: 'course-2',
    titulo: 'Angular Signals',
    creadorId: 'user-2',
    estado: 'en_revision',
    fechaEnvio: '2026-06-02T08:00:00Z',
  },
];

// ── Specs ─────────────────────────────────────────────────────────────────────

describe('AprovacionesComponent', () => {
  let approvalServiceSpy: Partial<ApprovalService>;
  let uiDialogSpy: Partial<UiDialogService>;

  beforeEach(async () => {
    approvalServiceSpy = {
      listPending: vi.fn().mockResolvedValue(MOCK_PENDING),
      approve: vi.fn().mockResolvedValue(undefined),
      reject: vi.fn().mockResolvedValue(undefined),
    };

    uiDialogSpy = {
      showSuccess: vi.fn(),
      showError: vi.fn(),
      confirm: vi.fn().mockResolvedValue(true),
      confirmDelete: vi.fn().mockResolvedValue(true),
    };

    await TestBed.configureTestingModule({
      imports: [AprovacionesComponent],
      providers: [
        { provide: ApprovalService, useValue: approvalServiceSpy },
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

  // ── Pending list render ──────────────────────────────────────────────────────

  it('loads pending list on init via ApprovalService.listPending()', async () => {
    const fixture = TestBed.createComponent(AprovacionesComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();

    expect(approvalServiceSpy.listPending).toHaveBeenCalled();
    expect(comp.pending()).toHaveLength(2);
    expect(comp.pending()[0].id).toBe('course-1');
    expect(comp.pending()[1].titulo).toBe('Angular Signals');
  });

  it('empty state: pending signal is empty array when no courses are pending', async () => {
    approvalServiceSpy.listPending = vi.fn().mockResolvedValue([]);

    const fixture = TestBed.createComponent(AprovacionesComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();

    expect(comp.pending()).toHaveLength(0);
  });

  // ── Approve flow ─────────────────────────────────────────────────────────────

  it('approve() calls ApprovalService.approve with courseId', async () => {
    const fixture = TestBed.createComponent(AprovacionesComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();
    await comp.approve('course-1');

    expect(approvalServiceSpy.approve).toHaveBeenCalledWith('course-1', undefined);
  });

  it('approve() refreshes the pending list on success', async () => {
    // After approve, listPending is called again to refresh
    approvalServiceSpy.listPending = vi.fn()
      .mockResolvedValueOnce(MOCK_PENDING)  // initial load
      .mockResolvedValueOnce([MOCK_PENDING[1]]); // after approve removes course-1

    const fixture = TestBed.createComponent(AprovacionesComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();
    expect(comp.pending()).toHaveLength(2);

    await comp.approve('course-1');

    // listPending called twice (init + refresh)
    expect(approvalServiceSpy.listPending).toHaveBeenCalledTimes(2);
    expect(comp.pending()).toHaveLength(1);
  });

  it('approve() shows success toast on success', async () => {
    const fixture = TestBed.createComponent(AprovacionesComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();
    await comp.approve('course-1');

    expect(uiDialogSpy.showSuccess).toHaveBeenCalled();
  });

  // ── Reject flow — requires comment ─────────────────────────────────────────

  it('reject() does NOT call ApprovalService.reject when comment is empty', async () => {
    const fixture = TestBed.createComponent(AprovacionesComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();

    // Simulate: dialog confirm called but resolved with empty/no comment → rejected
    await comp.reject('course-1');

    // If no non-empty comment was gathered, reject should not be called
    expect(approvalServiceSpy.reject).not.toHaveBeenCalled();
  });

  it('reject() calls ApprovalService.reject with courseId and comment when comment provided', async () => {
    const fixture = TestBed.createComponent(AprovacionesComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();

    // Simulate: provide the comment via the reject form signal directly before calling
    comp.rejectComentario.set('Falta evaluacion completa');
    await comp.reject('course-1');

    expect(approvalServiceSpy.reject).toHaveBeenCalledWith('course-1', 'Falta evaluacion completa');
  });

  it('reject() refreshes the pending list on success', async () => {
    approvalServiceSpy.listPending = vi.fn()
      .mockResolvedValueOnce(MOCK_PENDING)
      .mockResolvedValueOnce([MOCK_PENDING[1]]);

    const fixture = TestBed.createComponent(AprovacionesComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();

    comp.rejectComentario.set('Contenido insuficiente');
    await comp.reject('course-1');

    expect(approvalServiceSpy.listPending).toHaveBeenCalledTimes(2);
    expect(comp.pending()).toHaveLength(1);
  });

  it('reject() clears the rejection comment after success', async () => {
    const fixture = TestBed.createComponent(AprovacionesComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();

    comp.rejectComentario.set('Motivo valido');
    await comp.reject('course-1');

    expect(comp.rejectComentario()).toBe('');
  });

  it('reject() shows success toast on success', async () => {
    const fixture = TestBed.createComponent(AprovacionesComponent);
    const comp = fixture.componentInstance;

    await fixture.whenStable();

    comp.rejectComentario.set('Motivo valido');
    await comp.reject('course-1');

    expect(uiDialogSpy.showSuccess).toHaveBeenCalled();
  });
});
