/**
 * course-reports.component.ts — Per-course report table (C6.1).
 *
 * Shows all courses (regardless of estado) with enrollment, attempt, and approval stats.
 * approvalRate is rendered as a percentage: (row.approvalRate * 100) | number:'1.0-1'
 *
 * Uses PrimeNG TableModule. Follows the my-courses/badges TableModule pattern.
 * Uses --page-* CSS tokens (NOT --color-*).
 */
import {
  Component,
  inject,
  signal,
  OnInit,
} from '@angular/core';
import { DecimalPipe } from '@angular/common';
import { TableModule } from 'primeng/table';
import { SkeletonModule } from 'primeng/skeleton';

import { ReportingService } from '@core/services/reportingService/reporting.service';
import type { CourseReportItem } from '@core/services/reportingService/reporting.dto';

@Component({
  selector: 'app-course-reports',
  standalone: true,
  imports: [
    DecimalPipe,
    TableModule,
    SkeletonModule,
  ],
  templateUrl: './course-reports.component.html',
  styleUrl: './course-reports.component.sass',
})
export class CourseReportsComponent implements OnInit {
  private readonly reportingService = inject(ReportingService);

  // ── State ──────────────────────────────────────────────────────────────────
  readonly courses = signal<CourseReportItem[]>([]);
  readonly loading = signal<boolean>(false);

  ngOnInit(): void {
    void this.loadCourses();
  }

  async loadCourses(): Promise<void> {
    this.loading.set(true);
    try {
      const items = await this.reportingService.getCourses();
      this.courses.set(items);
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    } finally {
      this.loading.set(false);
    }
  }
}
