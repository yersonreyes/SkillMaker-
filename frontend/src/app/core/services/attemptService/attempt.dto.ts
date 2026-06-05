/**
 * attempt.dto.ts — Request/response types for the student attempt lifecycle API.
 *
 * SECURITY INVARIANT (ADR-E): AttemptStateOption has NO `correcta` field.
 * This is a structural guarantee — the backend omits correcta from the API response,
 * and the frontend type mirrors that omission to prevent accidental leaks in the UI.
 *
 * These are LOCAL types, NOT imported from @api/types, because they represent
 * student-facing shapes that must structurally exclude correcta.
 */

// ── Attempt start ─────────────────────────────────────────────────────────────

export interface AttemptStartResponse {
  attemptId: string;
  numero: number;
  iniciadoEn: string;
}

// ── Attempt state (in-progress or submitted) ──────────────────────────────────

/** Student-facing option — NO correcta field (structural no-leak guarantee). */
export interface AttemptStateOption {
  id: string;
  texto: string;
  // correcta is intentionally absent — the student must not see correct answers.
}

export interface AttemptStateQuestion {
  id: string;
  enunciado: string;
  tipo: 'opcion_multiple' | 'verdadero_falso';
  puntaje: number;
  options: AttemptStateOption[];
}

export interface AttemptAnswerView {
  questionId: string;
  optionId: string;
}

/** Full attempt state returned by GET /api/attempts/:id. */
export interface AttemptState {
  attemptId: string;
  numero: number;
  submitted: boolean;
  puntaje: number;    // meaningful only when submitted
  aprobado: boolean;  // meaningful only when submitted
  questions: AttemptStateQuestion[];
  answers: AttemptAnswerView[];
}

// ── Answer request ────────────────────────────────────────────────────────────

export interface AnswerRequest {
  questionId: string;
  optionId: string;
}

// ── Submit result ─────────────────────────────────────────────────────────────

export interface SubmitResponse {
  puntaje: number;
  aprobado: boolean;
}
