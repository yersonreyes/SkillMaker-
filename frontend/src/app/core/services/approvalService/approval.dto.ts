/**
 * approval.dto.ts — Request/response types for the Approvals API (C4.1).
 *
 * Types mirror the backend dto package (approvals/dto/dto.go).
 * These are LOCAL types to give clean camelCase aliases over the generated @api/types.
 */

// ── Pending list ──────────────────────────────────────────────────────────────

/** One item in the pending-review list (GET /api/approvals/pending). */
export interface PendingItem {
  id: string;
  titulo: string;
  creadorId: string;
  estado: string;
  /** OQ-3: proxy for submission time — updatedAt from the course row. */
  fechaEnvio: string; // ISO 8601
}

// ── History ───────────────────────────────────────────────────────────────────

/** One approval/rejection event (GET /api/courses/:id/approvals). */
export interface ApprovalHistoryItem {
  id: string;
  resultado: 'aprobado' | 'rechazado';
  comentario: string;
  adminId: string;
  resueltoEn: string; // ISO 8601
}

// ── Submit to review ──────────────────────────────────────────────────────────

/** Response from POST /api/courses/:courseId/submit. */
export interface ApprovalSubmitResponse {
  courseId: string;
  estado: string; // 'en_revision'
}

// ── Approve request ───────────────────────────────────────────────────────────

export interface ApproveRequest {
  comentario?: string;
}

// ── Reject request ────────────────────────────────────────────────────────────

export interface RejectRequest {
  comentario: string;
}
