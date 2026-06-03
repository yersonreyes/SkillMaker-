import {
  Component,
  inject,
  signal,
  OnInit,
} from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { TagModule } from 'primeng/tag';
import { InputTextModule } from 'primeng/inputtext';
import { ButtonModule } from 'primeng/button';
import { TextareaModule } from 'primeng/textarea';
import { TooltipModule } from 'primeng/tooltip';
import { SkeletonModule } from 'primeng/skeleton';

import { CourseService } from '@core/services/courseService/course.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { CourseDetail, CourseEstado } from '@core/services/courseService/course.res.dto';

type TagSeverity = 'success' | 'info' | 'warn' | 'danger' | 'secondary' | 'contrast';

const ESTADO_SEVERITY: Record<CourseEstado, TagSeverity> = {
  borrador:    'secondary',
  en_revision: 'info',
  aprobado:    'success',
  rechazado:   'danger',
};

@Component({
  selector: 'app-curso-editar',
  standalone: true,
  imports: [
    FormsModule,
    TagModule,
    InputTextModule,
    ButtonModule,
    TextareaModule,
    TooltipModule,
    SkeletonModule,
  ],
  templateUrl: './curso-editar.component.html',
})
export class CursoEditarComponent implements OnInit {
  private readonly courseService = inject(CourseService);
  private readonly ui = inject(UiDialogService);
  private readonly route = inject(ActivatedRoute);

  // ── Form state ───────────────────────────────────────────────────────────────
  readonly titulo = signal<string>('');
  readonly descripcion = signal<string>('');
  readonly loading = signal<boolean>(false);
  readonly saving = signal<boolean>(false);
  readonly course = signal<CourseDetail | null>(null);

  /**
   * C2.1: "Enviar a revisión" is rendered but always disabled.
   * Will be wired in C2.2 when HasContent(courseID) is available.
   */
  readonly submitDisabled = true as const;

  private courseId = '';

  ngOnInit(): void {
    this.courseId = this.route.snapshot.paramMap.get('id') ?? '';
    void this.loadCourse();
  }

  async loadCourse(): Promise<void> {
    if (!this.courseId) return;
    this.loading.set(true);
    try {
      const detail = await this.courseService.getById(this.courseId);
      this.course.set(detail);
      this.titulo.set(detail.titulo);
      this.descripcion.set(detail.descripcion);
    } catch {
      // Error toast already shown by HttpPromiseBuilderService
    } finally {
      this.loading.set(false);
    }
  }

  async onSave(): Promise<void> {
    if (!this.courseId) return;
    this.saving.set(true);
    try {
      await this.courseService.update(this.courseId, {
        titulo: this.titulo(),
        descripcion: this.descripcion(),
      });
      this.ui.showSuccess('Curso guardado');
    } catch {
      // Error toast already shown by HttpPromiseBuilderService
    } finally {
      this.saving.set(false);
    }
  }

  estadoSeverity(estado: CourseEstado): TagSeverity {
    return ESTADO_SEVERITY[estado] ?? 'secondary';
  }

  estadoLabel(estado: CourseEstado): string {
    const labels: Record<CourseEstado, string> = {
      borrador:    'Borrador',
      en_revision: 'En revision',
      aprobado:    'Aprobado',
      rechazado:   'Rechazado',
    };
    return labels[estado] ?? estado;
  }
}
