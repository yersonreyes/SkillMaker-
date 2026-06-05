/**
 * global-reports.component.spec.ts — GlobalReportsComponent unit tests (Strict TDD — RED → GREEN).
 *
 * Covers:
 *  - Calls getGlobal() on init
 *  - Renders metric card with activeUsers
 *  - Renders metric card with totalAttempts
 *  - Renders metric card with certificatesIssued
 *  - Renders 2 p-chart instances (line + bar)
 *  - Renders top-creators list
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { GlobalReportsComponent } from './global-reports.component';
import { ReportingService } from '@core/services/reportingService/reporting.service';
import type { GlobalReportResponse } from '@core/services/reportingService/reporting.dto';

const MOCK_GLOBAL: GlobalReportResponse = {
  activeUsers: 42,
  totalAttempts: 100,
  certificatesIssued: 15,
  coursesByEstado: [
    { estado: 'aprobado', total: 10 },
    { estado: 'borrador', total: 3 },
    { estado: 'en_revision', total: 2 },
    { estado: 'rechazado', total: 1 },
  ],
  topCreators: [
    { nombre: 'Ana Lopez', total: 5 },
    { nombre: 'Bob Martinez', total: 2 },
  ],
  usersPerMonth: [
    { month: '2026-01', total: 20 },
    { month: '2026-02', total: 22 },
  ],
  approvedCoursesPerMonth: [
    { month: '2026-01', total: 5 },
    { month: '2026-02', total: 3 },
  ],
};

describe('GlobalReportsComponent', () => {
  let reportingServiceSpy: Partial<ReportingService>;

  beforeEach(async () => {
    reportingServiceSpy = {
      getGlobal: vi.fn().mockResolvedValue(MOCK_GLOBAL),
    };

    await TestBed.configureTestingModule({
      imports: [GlobalReportsComponent],
      providers: [
        { provide: ReportingService, useValue: reportingServiceSpy },
        provideRouter([]),
        provideAnimationsAsync(),
        ConfirmationService,
        MessageService,
      ],
    }).compileComponents();
  });

  afterEach(() => {
    TestBed.resetTestingModule();
  });

  it('calls getGlobal() on init', async () => {
    const fixture = TestBed.createComponent(GlobalReportsComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(reportingServiceSpy.getGlobal).toHaveBeenCalledTimes(1);
  });

  it('renders activeUsers metric card', async () => {
    const fixture = TestBed.createComponent(GlobalReportsComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('42');
  });

  it('renders totalAttempts metric card', async () => {
    const fixture = TestBed.createComponent(GlobalReportsComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('100');
  });

  it('renders certificatesIssued metric card', async () => {
    const fixture = TestBed.createComponent(GlobalReportsComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('15');
  });

  it('renders 2 p-chart instances', async () => {
    const fixture = TestBed.createComponent(GlobalReportsComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    const charts = el.querySelectorAll('p-chart');
    expect(charts.length).toBe(2);
  });

  it('renders top-creators list with creator names', async () => {
    const fixture = TestBed.createComponent(GlobalReportsComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Ana Lopez');
    expect(el.textContent).toContain('Bob Martinez');
  });
});
