import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type {
  QuestionCreateRequest,
  QuestionUpdateRequest,
  QuestionResponse,
  OptionCreateRequest,
  OptionUpdateRequest,
  OptionResponse,
} from '../evaluationService/evaluation.dto';

@Injectable({ providedIn: 'root' })
export class QuestionService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly evaluationsBase = `${environment.apiBaseUrl}/evaluations`;
  private readonly questionsBase   = `${environment.apiBaseUrl}/questions`;
  private readonly optionsBase     = `${environment.apiBaseUrl}/options`;

  // ── Questions ─────────────────────────────────────────────────────────────────

  /** POST /api/evaluations/:evalId/questions */
  create(evalId: string, body: QuestionCreateRequest): Promise<QuestionResponse> {
    return this.http
      .request<QuestionResponse>()
      .post()
      .url(`${this.evaluationsBase}/${evalId}/questions`)
      .body(body)
      .send();
  }

  /** PATCH /api/questions/:id — partial update (enunciado, puntaje, orden). */
  update(id: string, body: QuestionUpdateRequest): Promise<QuestionResponse> {
    return this.http
      .request<QuestionResponse>()
      .patch()
      .url(`${this.questionsBase}/${id}`)
      .body(body)
      .send();
  }

  /** DELETE /api/questions/:id — cascade-deletes options. */
  delete(id: string): Promise<void> {
    return this.http
      .request<void>()
      .delete()
      .url(`${this.questionsBase}/${id}`)
      .send();
  }

  // ── Options (folded in — options never exist without a question) ──────────────

  /** POST /api/questions/:questionId/options */
  createOption(questionId: string, body: OptionCreateRequest): Promise<OptionResponse> {
    return this.http
      .request<OptionResponse>()
      .post()
      .url(`${this.questionsBase}/${questionId}/options`)
      .body(body)
      .send();
  }

  /** PATCH /api/options/:optionId — update texto and/or correcta. */
  updateOption(optionId: string, body: OptionUpdateRequest): Promise<OptionResponse> {
    return this.http
      .request<OptionResponse>()
      .patch()
      .url(`${this.optionsBase}/${optionId}`)
      .body(body)
      .send();
  }

  /** DELETE /api/options/:optionId */
  deleteOption(optionId: string): Promise<void> {
    return this.http
      .request<void>()
      .delete()
      .url(`${this.optionsBase}/${optionId}`)
      .send();
  }
}
