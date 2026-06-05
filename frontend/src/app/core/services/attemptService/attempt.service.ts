import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type {
  AttemptStartResponse,
  AttemptState,
  AnswerRequest,
  SubmitResponse,
} from './attempt.dto';

@Injectable({ providedIn: 'root' })
export class AttemptService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly evaluationsBase = `${environment.apiBaseUrl}/evaluations`;
  private readonly attemptsBase    = `${environment.apiBaseUrl}/attempts`;

  /** POST /api/evaluations/:evaluationId/attempts — start a new attempt. */
  startAttempt(evaluationId: string): Promise<AttemptStartResponse> {
    return this.http
      .request<AttemptStartResponse>()
      .post()
      .url(`${this.evaluationsBase}/${evaluationId}/attempts`)
      .send();
  }

  /** GET /api/attempts/:attemptId — fetch attempt state (questions + current answers). */
  getAttempt(attemptId: string): Promise<AttemptState> {
    return this.http
      .request<AttemptState>()
      .get()
      .url(`${this.attemptsBase}/${attemptId}`)
      .send();
  }

  /**
   * POST /api/attempts/:attemptId/answers — save (upsert) one answer.
   * The backend returns 204 No Content; we return void.
   */
  saveAnswer(attemptId: string, body: AnswerRequest): Promise<void> {
    return this.http
      .request<void>()
      .post()
      .url(`${this.attemptsBase}/${attemptId}/answers`)
      .body(body)
      .send();
  }

  /** POST /api/attempts/:attemptId/submit — finalize the attempt and receive score. */
  submitAttempt(attemptId: string): Promise<SubmitResponse> {
    return this.http
      .request<SubmitResponse>()
      .post()
      .url(`${this.attemptsBase}/${attemptId}/submit`)
      .send();
  }
}
