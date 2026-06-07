/**
 * notification.service.ts — Notifications API (notifications-inapp).
 *
 * Uses HttpPromiseBuilderService (same pattern as CertificateService).
 * Base URL: /api/notifications.
 */
import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type { NotificationItem, NotificationListResponse, UnreadCountResponse } from './notification.dto';

@Injectable({ providedIn: 'root' })
export class NotificationService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly base = `${environment.apiBaseUrl}/notifications`;

  /**
   * GET /api/notifications/me?page=&size=
   * Returns the authenticated user's notifications ordered by creadoEn DESC.
   */
  getMine(page = 1, size = 20): Promise<NotificationItem[]> {
    return this.http
      .request<NotificationListResponse>()
      .get()
      .url(`${this.base}/me`)
      .queryParam('page', page)
      .queryParam('size', size)
      .send()
      .then(res => res.items ?? []);
  }

  /**
   * GET /api/notifications/me/unread-count
   * Returns the caller's unread notification count.
   */
  getUnreadCount(): Promise<number> {
    return this.http
      .request<UnreadCountResponse>()
      .get()
      .url(`${this.base}/me/unread-count`)
      .send()
      .then(res => res.unread ?? 0);
  }

  /**
   * PATCH /api/notifications/:id/read
   * Marks the specified notification as read (caller-scoped).
   */
  markRead(id: string): Promise<void> {
    return this.http
      .request<unknown>()
      .patch()
      .url(`${this.base}/${id}/read`)
      .send()
      .then(() => undefined);
  }

  /**
   * PATCH /api/notifications/me/read-all
   * Marks all of the caller's notifications as read.
   */
  markAllRead(): Promise<void> {
    return this.http
      .request<unknown>()
      .patch()
      .url(`${this.base}/me/read-all`)
      .send()
      .then(() => undefined);
  }
}
