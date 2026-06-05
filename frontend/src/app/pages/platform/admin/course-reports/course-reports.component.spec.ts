/**
 * course-reports.component.spec.ts — CourseReportsComponent unit tests (Strict TDD — RED → GREEN).
 *
 * Covers:
 *  - Calls getCourses() on init
 *  - Renders p-table with titulo column
 *  - Renders p-table with estado column
 *  - Renders p-table with enrollments column
 *  - Renders p-table with attempts column
 *  - Renders approvalRate as percentage
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { CourseReportsComponent } from './course-reports.component';
import { ReportingService } from '@core/services/reportingService/reporting.service';
import type { CourseReportItem } from '@core/services/reportingService/reporting.dto';

const MOCK_COURSES: CourseReportItem[] = [
  {
    courseId: 'course-1',
    titulo: 'Go Avanzado',
    estado: 'aprobado',
    enrollments: 10,
    attempts: 8,
    approvalRate: 0.75,
  },
  {
    courseId: 'course-2',
    titulo: 'Angular Basico',
    estado: 'borrador',
    enrollments: 0,
    attempts: 0,
    approvalRate: 0,
  },
];

describe('CourseReportsComponent', () => {
  let reportingServiceSpy: Partial<ReportingService>;

  beforeEach(async () => {
    reportingServiceSpy = {
      getCourses: vi.fn().mockResolvedValue(MOCK_COURSES),
    };

    await TestBed.configureTestingModule({
      imports: [CourseReportsComponent],
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

  it('calls getCourses() on init', async () => {
    const fixture = TestBed.createComponent(CourseReportsComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(reportingServiceSpy.getCourses).toHaveBeenCalledTimes(1);
  });

  it('renders p-table with course titulo', async () => {
    const fixture = TestBed.createComponent(CourseReportsComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Go Avanzado');
  });

  it('renders p-table with estado column', async () => {
    const fixture = TestBed.createComponent(CourseReportsComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('aprobado');
  });

  it('renders p-table with enrollments count', async () => {
    const fixture = TestBed.createComponent(CourseReportsComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('10');
  });

  it('renders p-table with attempts count', async () => {
    const fixture = TestBed.createComponent(CourseReportsComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('8');
  });

  it('renders approvalRate as percentage', async () => {
    const fixture = TestBed.createComponent(CourseReportsComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    // 0.75 * 100 = 75.0 → renders as "75.0%"
    expect(el.textContent).toContain('75');
    expect(el.textContent).toContain('%');
  });
});
