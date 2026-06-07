import {
  Component,
  inject,
  signal,
  computed,
  OnInit,
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { RadioButtonModule } from 'primeng/radiobutton';
import { SkeletonModule } from 'primeng/skeleton';
import { FormsModule } from '@angular/forms';

import { AttemptService } from '@core/services/attemptService/attempt.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type {
  AttemptStateQuestion,
  SubmitResponse,
} from '@core/services/attemptService/attempt.dto';

type Phase = 'start' | 'taking' | 'result';

@Component({
  selector: 'app-evaluacion-tomar',
  standalone: true,
  imports: [FormsModule, RadioButtonModule, SkeletonModule],
  templateUrl: './evaluacion-tomar.component.html',
  styleUrl: './evaluacion-tomar.component.sass',
})
export class EvaluacionTomarComponent implements OnInit {
  private readonly attemptService = inject(AttemptService);
  private readonly ui             = inject(UiDialogService);
  private readonly route          = inject(ActivatedRoute);
  private readonly router         = inject(Router);

  // ── State ──────────────────────────────────────────────────────────────────

  readonly phase        = signal<Phase>('start');
  readonly questions    = signal<AttemptStateQuestion[]>([]);
  readonly result       = signal<SubmitResponse | null>(null);
  readonly startBlocked = signal<boolean>(false);
  readonly busy         = signal<boolean>(false);

  /** Current attempt number (shown after start). */
  readonly attemptNumero = signal<number>(0);

  /** Map questionId → chosen optionId. */
  private readonly answersMap = signal<Record<string, string>>({});

  private evaluationId = '';
  private attemptId    = '';

  // ── Computed ───────────────────────────────────────────────────────────────

  readonly isAprobado = computed(() => this.result()?.aprobado ?? false);
  readonly puntaje        = computed(() => this.result()?.puntaje ?? 0);
  readonly answeredCount  = computed(() => Object.keys(this.answersMap()).length);
  readonly totalQuestions = computed(() => this.questions().length);

  // ── Lifecycle ──────────────────────────────────────────────────────────────

  ngOnInit(): void {
    this.evaluationId = this.route.snapshot.paramMap.get('id') ?? '';
  }

  // ── Public API (called from template and specs) ────────────────────────────

  /** Returns the currently selected optionId for a given questionId, or null. */
  selectedAnswer(questionId: string): string | null {
    return this.answersMap()[questionId] ?? null;
  }

  /** Start the attempt: POST /evaluations/:id/attempts → GET /attempts/:id */
  async startAttempt(): Promise<void> {
    if (!this.evaluationId) return;
    this.busy.set(true);
    try {
      const start = await this.attemptService.startAttempt(this.evaluationId);
      this.attemptId = start.attemptId;
      this.attemptNumero.set(start.numero);

      const state = await this.attemptService.getAttempt(this.attemptId);
      // Restore any previously saved answers (if reload mid-attempt)
      const map: Record<string, string> = {};
      for (const ans of state.answers ?? []) {
        map[ans.questionId] = ans.optionId;
      }
      this.answersMap.set(map);
      this.questions.set(state.questions ?? []);
      this.phase.set('taking');
    } catch {
      // HttpPromiseBuilderService already showed the error toast.
      // Mark start as blocked so the template can show a friendly message.
      this.startBlocked.set(true);
    } finally {
      this.busy.set(false);
    }
  }

  /** Save-on-change: called when the student picks a radio option. */
  async selectOption(questionId: string, optionId: string): Promise<void> {
    // Optimistically update local map first for instant UI feedback.
    this.answersMap.update(m => ({ ...m, [questionId]: optionId }));
    try {
      await this.attemptService.saveAnswer(this.attemptId, { questionId, optionId });
    } catch {
      // Toast already shown; local state is still updated so the radio stays selected.
    }
  }

  /** Submit: confirm → POST /attempts/:id/submit → result phase. */
  async submitAttempt(): Promise<void> {
    const confirmed = await this.ui.confirm({
      message: '¿Estas seguro de enviar la evaluacion? No podras cambiar tus respuestas.',
      header: 'Finalizar intento',
      acceptLabel: 'Enviar',
      rejectLabel: 'Cancelar',
    });
    if (!confirmed) return;

    this.busy.set(true);
    try {
      const submitResult = await this.attemptService.submitAttempt(this.attemptId);
      this.result.set(submitResult);
      this.phase.set('result');
    } catch {
      // Toast already shown
    } finally {
      this.busy.set(false);
    }
  }

  /** Navigate back to the platform catalog or my-courses. */
  goBack(): void {
    void this.router.navigateByUrl('/platform/my-courses');
  }
}
