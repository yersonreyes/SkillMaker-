/**
 * badge.service.ts — Badge + Ranking API (C5.1).
 *
 * Uses HttpPromiseBuilderService (same pattern as CourseCatalogService).
 * Base URL: /api/badges.
 */
import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type { BadgeResponse, RankingItem } from './badge.dto';

@Injectable({ providedIn: 'root' })
export class BadgeService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly base = `${environment.apiBaseUrl}/badges`;

  /**
   * GET /api/badges/me
   * Returns the authenticated user's earned badges.
   */
  getMyBadges(): Promise<BadgeResponse[]> {
    return this.http
      .request<{ badges?: BadgeResponse[] | null }>()
      .get()
      .url(`${this.base}/me`)
      .send()
      .then(res => res.badges ?? []);
  }

  /**
   * GET /api/badges/ranking
   * Returns the top-10 users by certificate count (0-cert users excluded).
   */
  getRanking(): Promise<RankingItem[]> {
    return this.http
      .request<{ ranking?: RankingItem[] | null }>()
      .get()
      .url(`${this.base}/ranking`)
      .send()
      .then(res => res.ranking ?? []);
  }
}
