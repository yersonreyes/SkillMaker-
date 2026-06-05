import {
  Component,
  inject,
  signal,
  computed,
  OnInit,
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { InputTextModule } from 'primeng/inputtext';
import { TextareaModule } from 'primeng/textarea';
import { TooltipModule } from 'primeng/tooltip';
import { SkeletonModule } from 'primeng/skeleton';
import { DialogModule } from 'primeng/dialog';
import { SelectModule } from 'primeng/select';

import { EvaluationService } from '@core/services/evaluationService/evaluation.service';
import { QuestionService } from '@core/services/questionService/question.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type {
  EvaluationDetail,
  EvaluationResponse,
  QuestionItem,
  OptionItem,
} from '@core/services/evaluationService/evaluation.dto';

/** Local form state for the "Crear/Editar evaluación" section. */
export interface EvaluationFormState {
  notaMinima: number;
  intentosMax: number;
}

/** Local form state for the "Agregar pregunta" modal. */
export interface QuestionFormState {
  enunciado: string;
  tipo: 'opcion_multiple' | 'verdadero_falso';
  puntaje: number;
}

/** Local inline option row for the opcion_multiple editor. */
export interface OptionFormRow {
  texto: string;
  correcta: boolean;
}

@Component({
  selector: 'app-evaluacion-editar',
  standalone: true,
  imports: [
    FormsModule,
    InputTextModule,
    TextareaModule,
    TooltipModule,
    SkeletonModule,
    DialogModule,
    SelectModule,
  ],
  templateUrl: './evaluacion-editar.component.html',
  styleUrl: './evaluacion-editar.component.sass',
})
export class EvaluacionEditarComponent implements OnInit {
  private readonly evaluationService = inject(EvaluationService);
  private readonly questionService   = inject(QuestionService);
  private readonly ui                = inject(UiDialogService);
  private readonly route             = inject(ActivatedRoute);
  private readonly router            = inject(Router);

  // ── State ─────────────────────────────────────────────────────────────────────

  readonly loading   = signal<boolean>(false);
  readonly saving    = signal<boolean>(false);
  readonly evaluation = signal<EvaluationDetail | null>(null);
  readonly questions  = signal<QuestionItem[]>([]);

  /** True when there is no evaluation yet → show the "create" form. */
  readonly showEmptyState = computed(() => this.evaluation() === null);

  // ── Create evaluation form (used in empty state) ──────────────────────────────

  createForm: EvaluationFormState = { notaMinima: 70, intentosMax: 0 };

  // ── "Agregar pregunta" modal ──────────────────────────────────────────────────

  readonly addQuestionVisible = signal<boolean>(false);
  readonly addingQuestionBusy = signal<boolean>(false);
  questionForm: QuestionFormState = { enunciado: '', tipo: 'opcion_multiple', puntaje: 10 };

  /** Pending options for the opcion_multiple editor inside the modal. */
  readonly pendingOptions = signal<OptionFormRow[]>([]);
  readonly optionValidationError = signal<string>('');

  /** Per-question draft text for the inline "add option" input on persisted questions. */
  readonly optionDrafts = signal<Record<string, string>>({});

  // ── tipo options for the Select dropdown ─────────────────────────────────────

  readonly tipoOptions = [
    { label: 'Opcion multiple', value: 'opcion_multiple' as const },
    { label: 'Verdadero / Falso', value: 'verdadero_falso' as const },
  ];

  // ── Busy flags for per-question/option operations ─────────────────────────────

  private courseId = '';

  ngOnInit(): void {
    this.courseId = this.route.snapshot.paramMap.get('courseId') ?? '';
    void this.loadEvaluation();
  }

  // ── Load ──────────────────────────────────────────────────────────────────────

  async loadEvaluation(): Promise<void> {
    if (!this.courseId) return;
    this.loading.set(true);
    try {
      const detail = await this.evaluationService.getByCourse(this.courseId);
      if (detail) {
        this.evaluation.set(detail);
        this.questions.set(detail.questions ?? []);
        // Pre-fill the edit form with existing values
        this.createForm = { notaMinima: detail.notaMinima, intentosMax: detail.intentosMax };
      } else {
        this.evaluation.set(null);
        this.questions.set([]);
      }
    } catch {
      // Error toast shown by HttpPromiseBuilderService (non-404 errors)
    } finally {
      this.loading.set(false);
    }
  }

  // ── Create / Update evaluation ────────────────────────────────────────────────

  async createEvaluation(): Promise<void> {
    if (!this.courseId) return;
    this.saving.set(true);
    try {
      const created: EvaluationResponse = await this.evaluationService.create(this.courseId, {
        notaMinima: this.createForm.notaMinima,
        intentosMax: this.createForm.intentosMax,
      });
      // Set the evaluation with empty questions tree (just created)
      this.evaluation.set({
        id: created.id ?? '',
        courseId: created.courseId ?? '',
        notaMinima: created.notaMinima ?? 70,
        intentosMax: created.intentosMax ?? 0,
        questions: [],
      });
      this.questions.set([]);
      this.ui.showSuccess('Evaluacion creada');
    } catch {
      // Error toast already shown
    } finally {
      this.saving.set(false);
    }
  }

  async saveEvaluation(): Promise<void> {
    const ev = this.evaluation();
    if (!ev) return;
    this.saving.set(true);
    try {
      const updated = await this.evaluationService.update(ev.id, {
        notaMinima: this.createForm.notaMinima,
        intentosMax: this.createForm.intentosMax,
      });
      this.evaluation.update(e => e ? {
        ...e,
        notaMinima: updated.notaMinima ?? e.notaMinima,
        intentosMax: updated.intentosMax ?? e.intentosMax,
      } : e);
      this.ui.showSuccess('Evaluacion guardada');
    } catch {
      // Error toast already shown
    } finally {
      this.saving.set(false);
    }
  }

  // ── Questions ─────────────────────────────────────────────────────────────────

  openAddQuestionDialog(): void {
    this.questionForm = { enunciado: '', tipo: 'opcion_multiple', puntaje: 10 };
    this.pendingOptions.set([]);
    this.optionValidationError.set('');
    this.addQuestionVisible.set(true);
  }

  async addQuestion(): Promise<void> {
    const ev = this.evaluation();
    if (!ev) return;
    const { enunciado, tipo, puntaje } = this.questionForm;
    if (!enunciado.trim()) return;

    // Client-side ≥1-correct check for opcion_multiple BEFORE posting
    if (tipo === 'opcion_multiple') {
      const hasCorrect = this.pendingOptions().some(o => o.correcta);
      if (this.pendingOptions().length > 0 && !hasCorrect) {
        this.optionValidationError.set('Debe marcar al menos una opcion como correcta.');
        return;
      }
    }

    this.addingQuestionBusy.set(true);
    try {
      const created = await this.questionService.create(ev.id, {
        enunciado: enunciado.trim(),
        tipo,
        puntaje,
      });

      if (tipo === 'verdadero_falso') {
        // Backend auto-creates V/F options — reload the whole evaluation to get them
        await this.loadEvaluation();
        this.addQuestionVisible.set(false);
        this.ui.showSuccess('Pregunta agregada');
        return;
      }

      // opcion_multiple — post any pending options the author drafted in the modal
      const options: OptionItem[] = [];
      for (const row of this.pendingOptions()) {
        const opt = await this.questionService.createOption(created.id ?? '', {
          texto: row.texto,
          correcta: row.correcta,
        });
        options.push({
          id: opt.id ?? '',
          questionId: opt.questionId ?? '',
          texto: opt.texto ?? '',
          correcta: opt.correcta ?? false,
          orden: opt.orden ?? 0,
        });
      }

      const newQuestion: QuestionItem = {
        id: created.id ?? '',
        evaluationId: created.evaluationId ?? ev.id,
        enunciado: created.enunciado ?? enunciado.trim(),
        tipo: (created.tipo ?? tipo) as 'opcion_multiple' | 'verdadero_falso',
        puntaje: created.puntaje ?? puntaje,
        orden: created.orden ?? 0,
        options,
      };
      this.questions.update(list => [...list, newQuestion]);
      this.addQuestionVisible.set(false);
      this.ui.showSuccess('Pregunta agregada');
    } catch {
      // Error toast already shown
    } finally {
      this.addingQuestionBusy.set(false);
    }
  }

  async deleteQuestion(questionId: string): Promise<void> {
    const confirmed = await this.ui.confirmDelete('¿Eliminar esta pregunta y todas sus opciones?');
    if (!confirmed) return;
    try {
      await this.questionService.delete(questionId);
      this.questions.update(list => list.filter(q => q.id !== questionId));
      this.ui.showSuccess('Pregunta eliminada');
    } catch {
      // Error toast already shown
    }
  }

  // ── Options (opcion_multiple) ─────────────────────────────────────────────────

  // Pending options helpers (modal)

  addPendingOption(): void {
    this.pendingOptions.update(list => [...list, { texto: '', correcta: false }]);
    this.optionValidationError.set('');
  }

  removePendingOption(index: number): void {
    this.pendingOptions.update(list => list.filter((_, i) => i !== index));
    this.optionValidationError.set('');
  }

  togglePendingCorrect(index: number): void {
    this.pendingOptions.update(list =>
      list.map((o, i) => i === index ? { ...o, correcta: !o.correcta } : o),
    );
    this.optionValidationError.set('');
  }

  /** Post a new option directly on an existing persisted question. */
  async addOption(questionId: string, texto: string): Promise<void> {
    if (!texto.trim()) return;
    try {
      const opt = await this.questionService.createOption(questionId, { texto: texto.trim(), correcta: false });
      const newOpt: OptionItem = {
        id: opt.id ?? '',
        questionId: opt.questionId ?? questionId,
        texto: opt.texto ?? texto.trim(),
        correcta: opt.correcta ?? false,
        orden: opt.orden ?? 0,
      };
      this.questions.update(list =>
        list.map(q => q.id === questionId ? { ...q, options: [...q.options, newOpt] } : q),
      );
    } catch {
      // Error toast already shown
    }
  }

  async deleteOption(questionId: string, optionId: string): Promise<void> {
    const confirmed = await this.ui.confirmDelete('¿Eliminar esta opcion?');
    if (!confirmed) return;
    try {
      await this.questionService.deleteOption(optionId);
      this.questions.update(list =>
        list.map(q =>
          q.id === questionId ? { ...q, options: q.options.filter(o => o.id !== optionId) } : q,
        ),
      );
      this.ui.showSuccess('Opcion eliminada');
    } catch {
      // Error toast already shown
    }
  }

  /**
   * Toggle a persisted opcion_multiple option's correcta flag (multiple correct allowed).
   * PATCHes the option and reflects the new value locally.
   */
  async toggleOptionCorrect(questionId: string, optionId: string, current: boolean): Promise<void> {
    try {
      const updated = await this.questionService.updateOption(optionId, { correcta: !current });
      this.questions.update(list =>
        list.map(q =>
          q.id === questionId
            ? {
                ...q,
                options: q.options.map(o =>
                  o.id === optionId ? { ...o, correcta: updated.correcta ?? !current } : o,
                ),
              }
            : q,
        ),
      );
    } catch {
      // Error toast already shown
    }
  }

  // ── Inline add-option on a persisted opcion_multiple question ──────────────────

  setOptionDraft(questionId: string, texto: string): void {
    this.optionDrafts.update(m => ({ ...m, [questionId]: texto }));
  }

  optionDraft(questionId: string): string {
    return this.optionDrafts()[questionId] ?? '';
  }

  async submitNewOption(questionId: string): Promise<void> {
    const texto = this.optionDraft(questionId).trim();
    if (!texto) return;
    await this.addOption(questionId, texto);
    this.optionDrafts.update(m => ({ ...m, [questionId]: '' }));
  }

  // ── V/F correct selector ──────────────────────────────────────────────────────

  /**
   * PATCH the selected option correcta=true.
   * The backend automatically clears the sibling (mutual exclusion).
   * We reload the question's options from the backend response.
   */
  async setVFCorrect(questionId: string, optionId: string): Promise<void> {
    try {
      const updated = await this.questionService.updateOption(optionId, { correcta: true });
      // Update local state: mark this option true, let a reload sync the sibling
      this.questions.update(list =>
        list.map(q => {
          if (q.id !== questionId) return q;
          // Optimistic: mark this option true, siblings false (backend enforces it)
          return {
            ...q,
            options: q.options.map(o => ({
              ...o,
              correcta: o.id === optionId ? (updated.correcta ?? true) : false,
            })),
          };
        }),
      );
    } catch {
      // Error toast already shown
    }
  }

  // ── Client-side validation helpers ────────────────────────────────────────────

  /** Returns true if at least one option has correcta=true. */
  hasAtLeastOneCorrect(options: OptionItem[]): boolean {
    return options.some(o => o.correcta);
  }

  /**
   * Returns true when the question is safe to save:
   *  - verdadero_falso: always true (backend manages mutual exclusion)
   *  - opcion_multiple: requires ≥1 correcta=true option
   */
  canSaveQuestion(question: QuestionItem): boolean {
    if (question.tipo === 'verdadero_falso') return true;
    return this.hasAtLeastOneCorrect(question.options);
  }

  // ── Navigation ────────────────────────────────────────────────────────────────

  /**
   * Navigate to the evaluation editor using the ABSOLUTE /platform-prefixed path.
   * CRITICAL: a bare /creator/... path is a known C2.2 bug — never use relative paths here.
   */
  navigateToEvaluation(): Promise<boolean> {
    return this.router.navigateByUrl(`/platform/creator/evaluacion-editar/${this.courseId}`);
  }
}
