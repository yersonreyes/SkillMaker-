import {
  Component,
  inject,
  signal,
  OnInit,
} from '@angular/core';
import { Router } from '@angular/router';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TableModule } from 'primeng/table';
import { DialogModule } from 'primeng/dialog';
import { TagModule } from 'primeng/tag';
import { InputTextModule } from 'primeng/inputtext';
import { ButtonModule } from 'primeng/button';
import { TextareaModule } from 'primeng/textarea';
import { TooltipModule } from 'primeng/tooltip';

import { CourseService } from '@core/services/courseService/course.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { CourseListItem, CourseEstado } from '@core/services/courseService/course.res.dto';
import type { CourseListParams } from '@core/services/courseService/course.req.dto';

type TagSeverity = 'success' | 'info' | 'warn' | 'danger' | 'secondary' | 'contrast';

const ESTADO_SEVERITY: Record<CourseEstado, TagSeverity> = {
  borrador:    'secondary',
  en_revision: 'info',
  aprobado:    'success',
  rechazado:   'danger',
};

@Component({
  selector: 'app-mi-contenido',
  standalone: true,
  imports: [
    DatePipe,
    FormsModule,
    TableModule,
    DialogModule,
    TagModule,
    InputTextModule,
    ButtonModule,
    TextareaModule,
    TooltipModule,
  ],
  templateUrl: './mi-contenido.component.html',
})
export class MiContenidoComponent implements OnInit {
  private readonly courseService = inject(CourseService);
  private readonly ui = inject(UiDialogService);
  private readonly router = inject(Router);

  // ── List state ───────────────────────────────────────────────────────────────
  readonly courses = signal<CourseListItem[]>([]);
  readonly total = signal<number>(0);
  readonly loading = signal<boolean>(false);
  readonly pageSize = signal<number>(20);

  // ── Create dialog state ──────────────────────────────────────────────────────
  readonly dialogVisible = signal<boolean>(false);
  readonly saving = signal<boolean>(false);
  readonly newTitulo = signal<string>('');
  readonly newDescripcion = signal<string>('');

  // ── Cached params for lazy reload ────────────────────────────────────────────
  private lastParams: CourseListParams = { page: 1, size: 20 };

  ngOnInit(): void {
    void this.load({ page: 1, size: 20 });
  }

  /** Called by PrimeNG Table's (onLazyLoad) event. */
  async onLazyLoad(event: { first?: number | null; rows?: number | null }): Promise<void> {
    const first = event.first ?? 0;
    const rows = event.rows ?? 20;
    const page = Math.floor(first / rows) + 1;
    const size = rows;
    await this.load({ ...this.lastParams, page, size });
  }

  /** Open the create course dialog. */
  openCreateDialog(): void {
    this.newTitulo.set('');
    this.newDescripcion.set('');
    this.dialogVisible.set(true);
  }

  closeDialog(): void {
    this.dialogVisible.set(false);
  }

  /** Called from dialog Save button — creates a new borrador course. */
  async onSaveDialog(): Promise<void> {
    const titulo = this.newTitulo().trim();
    if (!titulo) return;

    this.saving.set(true);
    try {
      const created = await this.courseService.create({
        titulo,
        descripcion: this.newDescripcion().trim() || undefined,
      });
      this.dialogVisible.set(false);
      this.ui.showSuccess('Curso creado');
      // Navigate to edit view for the new course
      void this.router.navigate(['/creator/curso-editar', created.id]);
    } catch {
      // Error toast already shown by HttpPromiseBuilderService
    } finally {
      this.saving.set(false);
    }
  }

  /** Navigate to curso-editar/:id on row click. */
  navigateToCourse(id: string): void {
    void this.router.navigate(['/creator/curso-editar', id]);
  }

  /** Map estado to PrimeNG Tag severity. */
  estadoSeverity(estado: CourseEstado): TagSeverity {
    return ESTADO_SEVERITY[estado] ?? 'secondary';
  }

  /** Map estado to display label. */
  estadoLabel(estado: CourseEstado): string {
    const labels: Record<CourseEstado, string> = {
      borrador:    'Borrador',
      en_revision: 'En revision',
      aprobado:    'Aprobado',
      rechazado:   'Rechazado',
    };
    return labels[estado] ?? estado;
  }

  // ── Private ──────────────────────────────────────────────────────────────────

  private async load(params: CourseListParams): Promise<void> {
    this.lastParams = params;
    this.loading.set(true);
    try {
      const page = await this.courseService.listByMe(params);
      this.courses.set(page.items);
      this.total.set(page.total);
      this.pageSize.set(params.size ?? 20);
    } catch {
      // Error toast already shown by builder
    } finally {
      this.loading.set(false);
    }
  }
}
