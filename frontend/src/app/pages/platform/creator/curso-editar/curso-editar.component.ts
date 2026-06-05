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
import { OrderListModule } from 'primeng/orderlist';

import { CourseService } from '@core/services/courseService/course.service';
import { SectionService } from '@core/services/sectionService/section.service';
import { VideoService } from '@core/services/videoService/video.service';
import { MaterialService } from '@core/services/materialService/material.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import { ApprovalService } from '@core/services/approvalService/approval.service';
import { VideoEmbedComponent } from '@shared/components/video-embed/video-embed.component';
import { MaterialUploaderComponent, humanizeBytes } from '@shared/components/material-uploader/material-uploader.component';
import type { CourseDetail, CourseEstado } from '@core/services/courseService/course.res.dto';
import type { ApprovalHistoryItem } from '@core/services/approvalService/approval.dto';
import type { SectionItem } from '@core/services/sectionService/section.res.dto';
import type { VideoItem } from '@core/services/videoService/video.res.dto';
import type { VideoProveedor } from '@core/services/videoService/video.req.dto';
import type { MaterialResponse } from '@core/services/materialService/material.types';

/** Section enriched with its videos list for the UI. */
export interface SectionWithVideos extends SectionItem {
  videos: VideoItem[];
}

/** Form state for the add/edit video dialog. */
export interface VideoFormState {
  titulo: string;
  url: string;
  proveedor: VideoProveedor;
  duracionS: number | undefined;
}

@Component({
  selector: 'app-curso-editar',
  standalone: true,
  imports: [
    FormsModule,
    InputTextModule,
    TextareaModule,
    TooltipModule,
    SkeletonModule,
    DialogModule,
    SelectModule,
    OrderListModule,
    VideoEmbedComponent,
    MaterialUploaderComponent,
  ],
  templateUrl: './curso-editar.component.html',
  styleUrl: './curso-editar.component.sass',
})
export class CursoEditarComponent implements OnInit {
  private readonly courseService = inject(CourseService);
  private readonly sectionService = inject(SectionService);
  private readonly videoService = inject(VideoService);
  private readonly materialService = inject(MaterialService);
  private readonly ui = inject(UiDialogService);
  private readonly approvalService = inject(ApprovalService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  // ── Course form state ────────────────────────────────────────────────────────
  readonly titulo = signal<string>('');
  readonly descripcion = signal<string>('');
  readonly loading = signal<boolean>(false);
  readonly saving = signal<boolean>(false);
  readonly submitting = signal<boolean>(false);
  readonly course = signal<CourseDetail | null>(null);
  readonly hasContent = signal<boolean>(false);

  /**
   * C4.1: "Enviar a revisión" is enabled ONLY when:
   *   - course has content (hasContent=true)
   *   - estado ∈ {borrador, rechazado} (can submit / re-submit after rejection)
   * Disabled for en_revision (already pending) and aprobado (already published).
   */
  readonly submitDisabled = computed(() => {
    const c = this.course();
    if (!this.hasContent() || !c) return true;
    return c.estado !== 'borrador' && c.estado !== 'rechazado';
  });

  /** Latest rejection comment from approval history (shown when estado=rechazado). */
  readonly rejectionComentario = signal<string | null>(null);

  // ── Sections state ───────────────────────────────────────────────────────────
  readonly sections = signal<SectionWithVideos[]>([]);
  readonly sectionsLoading = signal<boolean>(false);

  // ── Materials state ──────────────────────────────────────────────────────────
  readonly materials = signal<MaterialResponse[]>([]);
  readonly materialsLoading = signal<boolean>(false);

  /** Expose humanizeBytes for the template (pure function, no injection needed). */
  readonly humanizeBytes = humanizeBytes;

  // ── Add section dialog ───────────────────────────────────────────────────────
  readonly addSectionVisible = signal<boolean>(false);
  readonly newSectionTitulo = signal<string>('');
  readonly addingSectionBusy = signal<boolean>(false);

  // ── Add video dialog ─────────────────────────────────────────────────────────
  readonly addVideoVisible = signal<boolean>(false);
  readonly addVideoSectionId = signal<string>('');
  readonly addingVideoBusy = signal<boolean>(false);
  readonly videoForm = signal<VideoFormState>({
    titulo: '',
    url: '',
    proveedor: 'youtube',
    duracionS: undefined,
  });

  // ── Proveedor options ────────────────────────────────────────────────────────
  readonly proveedorOptions = [
    { label: 'YouTube', value: 'youtube' as const },
    { label: 'Vimeo',   value: 'vimeo'   as const },
  ];

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
      this.hasContent.set(detail.hasContent ?? false);

      // FE-4: when rechazado, load the latest rejection comment so the creator
      // knows WHY the course was rejected without needing an email.
      if (detail.estado === 'rechazado') {
        void this.loadRejectionComment();
      } else {
        this.rejectionComentario.set(null);
      }
    } catch {
      // Error toast already shown by HttpPromiseBuilderService
    } finally {
      this.loading.set(false);
    }

    // Load sections and materials in parallel (independent of course form data)
    await Promise.all([this.loadSections(), this.loadMaterials()]);
  }

  /** Fetch approval history and surface the latest rejection comment (FE-4). */
  async loadRejectionComment(): Promise<void> {
    if (!this.courseId) return;
    try {
      const history = await this.approvalService.history(this.courseId);
      const rejections = history.filter((h: ApprovalHistoryItem) => h.resultado === 'rechazado');
      if (rejections.length > 0) {
        // History returned most-recent-first per spec (resueltoEn DESC).
        this.rejectionComentario.set(rejections[0].comentario);
      }
    } catch {
      // Non-critical: rejection comment is informational only
    }
  }

  /**
   * FE-2 (C4.1): Submit course for admin review.
   * On success, reloads the course so estado → en_revision is reflected
   * and the button becomes disabled.
   */
  async onSubmitToReview(): Promise<void> {
    if (!this.courseId) return;
    this.submitting.set(true);
    try {
      await this.approvalService.submitToReview(this.courseId);
      this.ui.showSuccess('Curso enviado a revision');
      // Reload so estado signal reflects en_revision and button disables.
      await this.loadCourse();
    } catch {
      // Error toast already shown by HttpPromiseBuilderService
    } finally {
      this.submitting.set(false);
    }
  }

  async loadSections(): Promise<void> {
    if (!this.courseId) return;
    this.sectionsLoading.set(true);
    try {
      // listByCourse returns SectionWithVideos[] — sections with nested videos from the API.
      // CRITICAL fix: use the videos from the API response so existing sections/videos render
      // on page reload (instead of wiping them with videos: []).
      const items = await this.sectionService.listByCourse(this.courseId);
      this.sections.set(items.map(s => ({ ...s, videos: s.videos ?? [] })));
    } catch {
      // Error toast already shown
    } finally {
      this.sectionsLoading.set(false);
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

  // ── Section operations ───────────────────────────────────────────────────────

  openAddSectionDialog(): void {
    this.newSectionTitulo.set('');
    this.addSectionVisible.set(true);
  }

  async addSection(): Promise<void> {
    if (!this.newSectionTitulo().trim()) return;
    if (!this.courseId) return;

    this.addingSectionBusy.set(true);
    try {
      const created = await this.sectionService.create(this.courseId, {
        titulo: this.newSectionTitulo().trim(),
      });
      this.sections.update(list => [...list, { ...created, videos: [] }]);
      this.addSectionVisible.set(false);
      this.ui.showSuccess('Seccion agregada');
    } catch {
      // Error toast already shown
    } finally {
      this.addingSectionBusy.set(false);
    }
  }

  async deleteSection(sectionId: string): Promise<void> {
    const confirmed = await this.ui.confirmDelete('¿Eliminar esta seccion y todos sus videos?');
    if (!confirmed) return;

    try {
      await this.sectionService.delete(sectionId);
      this.sections.update(list => list.filter(s => s.id !== sectionId));
      this.ui.showSuccess('Seccion eliminada');
    } catch {
      // Error toast already shown
    }
  }

  async onSectionsReorder(newOrder: SectionWithVideos[]): Promise<void> {
    if (!this.courseId) return;
    const ids = newOrder.map(s => s.id);
    try {
      await this.sectionService.reorder(this.courseId, ids);
      this.sections.set(newOrder);
    } catch {
      // Error toast already shown
    }
  }

  // ── Video operations ─────────────────────────────────────────────────────────

  openAddVideoDialog(sectionId: string): void {
    this.addVideoSectionId.set(sectionId);
    this.videoForm.set({ titulo: '', url: '', proveedor: 'youtube', duracionS: undefined });
    this.addVideoVisible.set(true);
  }

  async addVideo(sectionId: string): Promise<void> {
    const form = this.videoForm();
    if (!form.titulo.trim() || !form.url.trim()) return;

    this.addingVideoBusy.set(true);
    try {
      const body: { titulo: string; url: string; proveedor: VideoProveedor; duracionS?: number } = {
        titulo: form.titulo.trim(),
        url: form.url.trim(),
        proveedor: form.proveedor,
      };
      if (form.duracionS) body['duracionS'] = form.duracionS;

      const created = await this.videoService.create(sectionId, body);
      this.sections.update(list =>
        list.map(s =>
          s.id === sectionId ? { ...s, videos: [...s.videos, created] } : s,
        ),
      );
      this.addVideoVisible.set(false);
      this.ui.showSuccess('Video agregado');
    } catch {
      // Error toast already shown
    } finally {
      this.addingVideoBusy.set(false);
    }
  }

  async deleteVideo(sectionId: string, videoId: string): Promise<void> {
    const confirmed = await this.ui.confirmDelete('¿Eliminar este video?');
    if (!confirmed) return;

    try {
      await this.videoService.delete(videoId);
      this.sections.update(list =>
        list.map(s =>
          s.id === sectionId
            ? { ...s, videos: s.videos.filter(v => v.id !== videoId) }
            : s,
        ),
      );
      this.ui.showSuccess('Video eliminado');
    } catch {
      // Error toast already shown
    }
  }

  // ── Material operations ──────────────────────────────────────────────────────

  async loadMaterials(): Promise<void> {
    if (!this.courseId) return;
    this.materialsLoading.set(true);
    try {
      const items = await this.materialService.list(this.courseId);
      this.materials.set(items);
    } catch {
      // Error toast already shown by HttpPromiseBuilderService
    } finally {
      this.materialsLoading.set(false);
    }
  }

  onMaterialUploaded(material: MaterialResponse): void {
    // Append the new material to the list (no full reload needed).
    this.materials.update(list => [...list, material]);
    this.ui.showSuccess('Material agregado');
  }

  async downloadMaterial(material: MaterialResponse): Promise<void> {
    if (!this.courseId) return;
    try {
      const resp = await this.materialService.downloadUrl(this.courseId, material.id);
      window.open(resp.url, '_blank', 'noopener');
    } catch {
      // Error toast already shown
    }
  }

  async deleteMaterial(material: MaterialResponse): Promise<void> {
    const confirmed = await this.ui.confirmDelete(
      `¿Eliminar "${material.nombre}"? Esta accion no se puede deshacer.`,
    );
    if (!confirmed) return;

    try {
      await this.materialService.remove(material.id);
      this.materials.update(list => list.filter(m => m.id !== material.id));
      this.ui.showSuccess('Material eliminado');
    } catch {
      // Error toast already shown
    }
  }

  // ── Helpers ──────────────────────────────────────────────────────────────────

  estadoLabel(estado: CourseEstado): string {
    const labels: Record<CourseEstado, string> = {
      borrador:    'Borrador',
      en_revision: 'En revision',
      aprobado:    'Aprobado',
      rechazado:   'Rechazado',
    };
    return labels[estado] ?? estado;
  }

  // ── Navigation ────────────────────────────────────────────────────────────────

  /**
   * Navigate to the evaluation editor.
   * CRITICAL: uses the ABSOLUTE /platform-prefixed path.
   * A bare /creator/... path is a known C2.2 bug pattern — never use relative paths here.
   */
  navigateToEvaluation(): Promise<boolean> {
    return this.router.navigateByUrl(`/platform/creator/evaluacion-editar/${this.courseId}`);
  }
}
