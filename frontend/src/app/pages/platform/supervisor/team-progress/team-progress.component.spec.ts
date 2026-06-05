/**
 * team-progress.component.spec.ts — TeamProgressComponent unit tests (Strict TDD — RED → GREEN).
 *
 * Covers:
 *  - Calls getTeam() on init
 *  - Renders p-table with empleadoNombre column
 *  - Renders p-table with enrolledCount column
 *  - Renders p-table with completedCount column
 *  - Renders lastAttemptDate when present
 *  - Shows '—' when lastAttemptDate is null
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { TeamProgressComponent } from './team-progress.component';
import { ReportingService } from '@core/services/reportingService/reporting.service';
import type { TeamReportItem } from '@core/services/reportingService/reporting.dto';

const MOCK_TEAM: TeamReportItem[] = [
  {
    empleadoId: 'user-1',
    empleadoNombre: 'Ana Lopez',
    enrolledCount: 5,
    completedCount: 3,
    lastAttemptDate: '2026-05-15',
  },
  {
    empleadoId: 'user-2',
    empleadoNombre: 'Bob Martinez',
    enrolledCount: 2,
    completedCount: 0,
    lastAttemptDate: null,
  },
];

describe('TeamProgressComponent', () => {
  let reportingServiceSpy: Partial<ReportingService>;

  beforeEach(async () => {
    reportingServiceSpy = {
      getTeam: vi.fn().mockResolvedValue(MOCK_TEAM),
    };

    await TestBed.configureTestingModule({
      imports: [TeamProgressComponent],
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

  it('calls getTeam() on init', async () => {
    const fixture = TestBed.createComponent(TeamProgressComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(reportingServiceSpy.getTeam).toHaveBeenCalledTimes(1);
  });

  it('renders p-table with empleadoNombre', async () => {
    const fixture = TestBed.createComponent(TeamProgressComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Ana Lopez');
    expect(el.textContent).toContain('Bob Martinez');
  });

  it('renders enrolledCount column', async () => {
    const fixture = TestBed.createComponent(TeamProgressComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('5');
  });

  it('renders completedCount column', async () => {
    const fixture = TestBed.createComponent(TeamProgressComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('3');
  });

  it('renders lastAttemptDate when present', async () => {
    const fixture = TestBed.createComponent(TeamProgressComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('2026-05-15');
  });

  it('shows em-dash when lastAttemptDate is null', async () => {
    const fixture = TestBed.createComponent(TeamProgressComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('—');
  });
});
