/**
 * global-reports.component.ts — Global reports dashboard (C6.1).
 *
 * Displays system-wide metrics:
 *  - Metric cards: activeUsers, totalAttempts, certificatesIssued
 *  - coursesByEstado breakdown
 *  - Line chart: usersPerMonth (using PrimeNG ChartModule + chart.js/auto)
 *  - Bar chart: approvedCoursesPerMonth
 *  - Top-creators list
 *
 * Uses --page-* CSS tokens (NOT --color-*).
 * Import: ChartModule from primeng/chart + chart.js/auto.
 */
import 'chart.js/auto';
import {
  Component,
  inject,
  signal,
  OnInit,
} from '@angular/core';
import { ChartModule } from 'primeng/chart';
import { SkeletonModule } from 'primeng/skeleton';

import { ReportingService } from '@core/services/reportingService/reporting.service';
import type { GlobalReportResponse, CoursesByEstadoItem, TopCreatorItem, MonthCountItem } from '@core/services/reportingService/reporting.dto';

interface ChartData {
  labels: string[];
  datasets: {
    label: string;
    data: number[];
    borderColor?: string;
    backgroundColor?: string;
    tension?: number;
    fill?: boolean;
  }[];
}

interface ChartOptions {
  responsive: boolean;
  maintainAspectRatio: boolean;
  plugins: {
    legend: { display: boolean };
  };
}

@Component({
  selector: 'app-global-reports',
  standalone: true,
  imports: [
    ChartModule,
    SkeletonModule,
  ],
  templateUrl: './global-reports.component.html',
  styleUrl: './global-reports.component.sass',
})
export class GlobalReportsComponent implements OnInit {
  private readonly reportingService = inject(ReportingService);

  // ── State ──────────────────────────────────────────────────────────────────
  readonly report = signal<GlobalReportResponse | null>(null);
  readonly loading = signal<boolean>(false);

  // ── Chart Data ─────────────────────────────────────────────────────────────
  usersChartData: ChartData = { labels: [], datasets: [] };
  coursesChartData: ChartData = { labels: [], datasets: [] };

  readonly chartOptions: ChartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: { display: true },
    },
  };

  ngOnInit(): void {
    void this.loadReport();
  }

  async loadReport(): Promise<void> {
    this.loading.set(true);
    try {
      const data = await this.reportingService.getGlobal();
      this.report.set(data);
      this.buildCharts(data);
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    } finally {
      this.loading.set(false);
    }
  }

  private buildCharts(data: GlobalReportResponse): void {
    this.usersChartData = this.buildLineChart(
      data.usersPerMonth ?? [],
      'Usuarios registrados',
    );
    this.coursesChartData = this.buildBarChart(
      data.approvedCoursesPerMonth ?? [],
      'Cursos aprobados',
    );
  }

  private buildLineChart(items: MonthCountItem[], label: string): ChartData {
    return {
      labels: items.map(m => m.month),
      datasets: [
        {
          label,
          data: items.map(m => m.total),
          borderColor: '#0e7a96',
          tension: 0.3,
          fill: false,
        },
      ],
    };
  }

  private buildBarChart(items: MonthCountItem[], label: string): ChartData {
    return {
      labels: items.map(m => m.month),
      datasets: [
        {
          label,
          data: items.map(m => m.total),
          backgroundColor: '#0e7a96',
        },
      ],
    };
  }

  /** Expose for template. */
  get coursesByEstado(): CoursesByEstadoItem[] {
    return this.report()?.coursesByEstado ?? [];
  }

  get topCreators(): TopCreatorItem[] {
    return this.report()?.topCreators ?? [];
  }
}
