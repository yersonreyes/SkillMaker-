/**
 * notification.dto.ts — DTOs for the notifications API.
 *
 * These mirror the Go backend DTOs exactly (NotificationResponse, NotificationListResponse,
 * UnreadCountResponse). Fields are optional to match the generated @api types.ts shape.
 */

/** One notification entry from GET /api/notifications/me. */
export interface NotificationItem {
  id?: string;
  tipo?: string; // 'curso_aprobado' | 'curso_rechazado' | 'certificado_emitido'
  titulo?: string;
  cuerpo?: string;
  leida?: boolean;
  refId?: string;
  creadoEn?: string; // ISO 8601
}

/** Paginated list response from GET /api/notifications/me. */
export interface NotificationListResponse {
  items?: NotificationItem[] | null;
  page?: number;
  size?: number;
  total?: number;
  totalPages?: number;
}

/** Response from GET /api/notifications/me/unread-count. */
export interface UnreadCountResponse {
  unread?: number;
}
