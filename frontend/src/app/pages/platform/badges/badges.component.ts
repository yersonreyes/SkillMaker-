/**
 * badges.component.ts — Insignias page (C5.1).
 *
 * Replaces the PendingViewComponent stub for the /platform/badges route.
 * Shows earned badge cards + a PrimeNG ranking table (top-10 by cert count).
 */
import {
  Component,
  inject,
  signal,
  OnInit,
} from '@angular/core';
import { DatePipe } from '@angular/common';
import { TableModule } from 'primeng/table';
import { SkeletonModule } from 'primeng/skeleton';

import { BadgeService } from '@core/services/badgeService/badge.service';
import type { BadgeResponse, RankingItem } from '@core/services/badgeService/badge.dto';

@Component({
  selector: 'app-badges',
  standalone: true,
  imports: [DatePipe, TableModule, SkeletonModule],
  templateUrl: './badges.component.html',
  styleUrl: './badges.component.sass',
})
export class BadgesComponent implements OnInit {
  private readonly badgeService = inject(BadgeService);

  // ── State ──────────────────────────────────────────────────────────────────
  readonly badges = signal<BadgeResponse[]>([]);
  readonly ranking = signal<RankingItem[]>([]);
  readonly loading = signal<boolean>(false);

  ngOnInit(): void {
    void this.loadAll();
  }

  async loadAll(): Promise<void> {
    this.loading.set(true);
    try {
      const [badgeItems, rankingItems] = await Promise.all([
        this.badgeService.getMyBadges(),
        this.badgeService.getRanking(),
      ]);
      this.badges.set(badgeItems);
      this.ranking.set(rankingItems);
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    } finally {
      this.loading.set(false);
    }
  }
}
