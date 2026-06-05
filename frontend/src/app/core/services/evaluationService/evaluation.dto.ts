/**
 * evaluation.dto.ts — Request/response types for the evaluations API.
 * Prefer generated @api/* types for flat shapes; define nested tree locally
 * since codegen does not produce the EvaluationDetail + questions + options tree.
 */
import type { definitions } from '@api/types';

// ── Flat types (from generated types.ts) ──────────────────────────────────────

export type EvaluationCreateRequest = definitions['dto.EvaluationCreateRequest'];
export type EvaluationUpdateRequest = definitions['dto.EvaluationUpdateRequest'];
export type EvaluationResponse     = definitions['dto.EvaluationResponse'];

export type QuestionCreateRequest = definitions['dto.QuestionCreateRequest'];
export type QuestionUpdateRequest = definitions['dto.QuestionUpdateRequest'];
export type QuestionResponse      = definitions['dto.QuestionResponse'];

export type OptionCreateRequest = definitions['dto.OptionCreateRequest'];
export type OptionUpdateRequest = definitions['dto.OptionUpdateRequest'];
export type OptionResponse      = definitions['dto.OptionResponse'];

// ── Nested tree (not produced by codegen — EvaluationDetail with questions) ───

export interface OptionItem {
  id: string;
  questionId: string;
  texto: string;
  correcta: boolean;
  orden: number;
}

export interface QuestionItem {
  id: string;
  evaluationId: string;
  enunciado: string;
  tipo: 'opcion_multiple' | 'verdadero_falso';
  puntaje: number;
  orden: number;
  options: OptionItem[];
}

export interface EvaluationDetail {
  id: string;
  courseId: string;
  notaMinima: number;
  intentosMax: number;
  questions: QuestionItem[];
}
