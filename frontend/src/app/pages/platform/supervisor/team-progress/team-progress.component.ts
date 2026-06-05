/**
 * team-progress.component.ts — Supervisor team progress dashboard (C6.1).
 *
 * Replaces the PendingViewComponent stub at /platform/supervisor/team-progress.
 * Shows the authenticated supervisor's supervised employees and their course progress.
 *
 * Uses PrimeNG TableModule. lastAttemptDate shows '—' when null.
 * Uses --page-* CSS tokens (NOT --color-*).
 */
import {
  Component,
  inject,
  signal,
  OnInit,
} from '@angular/core';
import { TableModule } from 'primeng/table';
import { SkeletonModule } from 'primeng/skeleton';

import { ReportingService } from '@core/services/reportingService/reporting.service';
import type { TeamReportItem } from '@core/services/reportingService/reporting.dto';

@Component({
  selector: 'app-team-progress',
  standalone: true,
  imports: [
    TableModule,
    SkeletonModule,
  ],
  templateUrl: './team-progress.component.html',
  styleUrl: './team-progress.component.sass',
})
export class TeamProgressComponent implements OnInit {
  private readonly reportingService = inject(ReportingService);

  // ── State ──────────────────────────────────────────────────────────────────
  readonly team = signal<TeamReportItem[]>([]);
  readonly loading = signal<boolean>(false);

  ngOnInit(): void {
    void this.loadTeam();
  }

  async loadTeam(): Promise<void> {
    this.loading.set(true);
    try {
      const items = await this.reportingService.getTeam();
      this.team.set(items);
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    } finally {
      this.loading.set(false);
    }
  }

  /** Format lastAttemptDate: return the date string or '—' when null/undefined. */
  formatDate(date: string | null | undefined): string {
    return date ?? '—';
  }
}
