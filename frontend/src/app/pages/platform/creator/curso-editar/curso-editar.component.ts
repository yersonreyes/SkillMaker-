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
import { InputNumberModule } from 'primeng/inputnumber';
import { TextareaModule } from 'primeng/textarea';
import { TooltipModule } from 'primeng/tooltip';
import { SkeletonModule } from 'primeng/skeleton';
import { DialogModule } from 'primeng/dialog';
import { SelectModule } from 'primeng/select';
import { MultiSelectModule } from 'primeng/multiselect';
import { OrderListModule } from 'primeng/orderlist';

import { CourseService } from '@core/services/courseService/course.service';
import { SectionService } from '@core/services/sectionService/section.service';
import { VideoService } from '@core/services/videoService/video.service';
import { MaterialService } from '@core/services/materialService/material.service';
import { CategoriaService } from '@core/services/categoriaService/categoria.service';
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
import type { CategoriaItem } from '@core/services/categoriaService/categoria.dto';

/** Nivel options for the p-select field. */
export const NIVEL_OPTIONS = [
  { label: 'Basico', value: 'basico' },
  { label: 'Intermedio', value: 'intermedio' },
  { label: 'Avanzado', value: 'avanzado' },
];

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
  descripcion: string;
}

@Component({
  selector: 'app-curso-editar',
  standalone: true,
  imports: [
    FormsModule,
    InputTextModule,
    InputNumberModule,
    TextareaModule,
    TooltipModule,
    SkeletonModule,
    DialogModule,
    SelectModule,
    MultiSelectModule,
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
  private readonly categoriaService = inject(CategoriaService);
  private readonly ui = inject(UiDialogService);
  private readonly approvalService = inject(ApprovalService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  // ── Course form state ────────────────────────────────────────────────────────
  readonly titulo = signal<string>('');
  readonly descripcion = signal<string>('');
  readonly nivel = signal<string | null>(null);
  readonly horasPractico = signal<number>(0);
  readonly selectedCategoriaIds = signal<string[]>([]);
  readonly miniaturaUrl = signal<string | null>(null);
  readonly miniaturaKey = signal<string | null>(null);
  readonly horasVideo = signal<number>(0);
  readonly cantidadClases = signal<number>(0);

  readonly loading = signal<boolean>(false);
  readonly saving = signal<boolean>(false);
  readonly submitting = signal<boolean>(false);
  readonly course = signal<CourseDetail | null>(null);
  readonly hasContent = signal<boolean>(false);

  // ── Categorias ───────────────────────────────────────────────────────────────
  readonly categorias = signal<CategoriaItem[]>([]);

  /** Options for p-multiselect — label/value pairs. */
  readonly categoriaOptions = computed(() =>
    this.categorias().map(c => ({ label: c.nombre, value: c.id })),
  );

  /**
   * C4.1: "Enviar a revisión" is enabled ONLY when:
   *   - course has content (hasContent=true)
   *   - estado ∈ {borrador, rechazado} (can submit / re-submit after rejection)
   */
  readonly submitDisabled = computed(() => {
    const c = this.course();
    if (!this.hasContent() || !c) return true;
    return c.estado !== 'borrador' && c.estado !== 'rechazado';
  });

  /** Latest rejection comment from approval history. */
  readonly rejectionComentario = signal<string | null>(null);

  // ── Nivel options ───────────────────────────────────────────────────────────���─
  readonly nivelOptions = NIVEL_OPTIONS;

  // ── Sections state ───────────────────────────────────────────────────────────
  readonly sections = signal<SectionWithVideos[]>([]);
  readonly sectionsLoading = signal<boolean>(false);

  /** Per-video materials map: videoId → MaterialResponse[] (for the editor list). */
  readonly videoMaterials = signal<Record<string, MaterialResponse[]>>({});

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
    descripcion: '',
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
    void this.loadCategorias();
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
      this.nivel.set(detail.nivel ?? null);
      this.horasPractico.set(detail.horasPractico ?? 0);
      this.miniaturaUrl.set(detail.miniaturaUrl ?? null);
      this.horasVideo.set(detail.horasVideo ?? 0);
      this.cantidadClases.set(detail.cantidadClases ?? 0);

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

    // Load sections in parallel (independent of course form data)
    await this.loadSections();
  }

  async loadCategorias(): Promise<void> {
    try {
      const items = await this.categoriaService.getAll();
      this.categorias.set(items);
    } catch {
      // Non-critical: categorias are informational
    }
  }

  /** Fetch approval history and surface the latest rejection comment. */
  async loadRejectionComment(): Promise<void> {
    if (!this.courseId) return;
    try {
      const history = await this.approvalService.history(this.courseId);
      const rejections = history.filter((h: ApprovalHistoryItem) => h.resultado === 'rechazado');
      if (rejections.length > 0) {
        this.rejectionComentario.set(rejections[0].comentario);
      }
    } catch {
      // Non-critical: rejection comment is informational only
    }
  }

  /** Submit course for admin review. */
  async onSubmitToReview(): Promise<void> {
    if (!this.courseId) return;
    this.submitting.set(true);
    try {
      await this.approvalService.submitToReview(this.courseId);
      this.ui.showSuccess('Curso enviado a revision');
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
      const items = await this.sectionService.listByCourse(this.courseId);
      this.sections.set(items.map(s => ({ ...s, videos: s.videos ?? [] })));
      // Load materials for all videos
      await this.loadAllVideoMaterials(items);
    } catch {
      // Error toast already shown
    } finally {
      this.sectionsLoading.set(false);
    }
  }

  /** Batch load materials for all videos in all sections. */
  async loadAllVideoMaterials(sections: SectionWithVideos[]): Promise<void> {
    const allVideos = sections.flatMap(s => s.videos ?? []);
    if (allVideos.length === 0) return;

    const results = await Promise.allSettled(
      allVideos.map(v => this.materialService.list(v.id).then(mats => ({ videoId: v.id, mats }))),
    );

    const map: Record<string, MaterialResponse[]> = {};
    for (const r of results) {
      if (r.status === 'fulfilled') {
        map[r.value.videoId] = r.value.mats;
      }
    }
    this.videoMaterials.set(map);
  }

  /** Get materials for a specific video (from the local cache). */
  getMaterialsForVideo(videoId: string): MaterialResponse[] {
    return this.videoMaterials()[videoId] ?? [];
  }

  async onSave(): Promise<void> {
    if (!this.courseId) return;
    this.saving.set(true);
    try {
      await this.courseService.update(this.courseId, {
        titulo: this.titulo(),
        descripcion: this.descripcion(),
        nivel: this.nivel() ?? undefined,
        horasPractico: this.horasPractico(),
        categoriaIds: this.selectedCategoriaIds(),
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
    this.videoForm.set({ titulo: '', url: '', proveedor: 'youtube', duracionS: undefined, descripcion: '' });
    this.addVideoVisible.set(true);
  }

  async addVideo(sectionId: string): Promise<void> {
    const form = this.videoForm();
    if (!form.titulo.trim() || !form.url.trim()) return;

    this.addingVideoBusy.set(true);
    try {
      const body: { titulo: string; url: string; proveedor: VideoProveedor; duracionS?: number; descripcion?: string } = {
        titulo: form.titulo.trim(),
        url: form.url.trim(),
        proveedor: form.proveedor,
      };
      if (form.duracionS) body['duracionS'] = form.duracionS;
      if (form.descripcion.trim()) body['descripcion'] = form.descripcion.trim();

      const created = await this.videoService.create(sectionId, body);
      this.sections.update(list =>
        list.map(s =>
          s.id === sectionId ? { ...s, videos: [...s.videos, created] } : s,
        ),
      );
      // Initialize empty materials list for new video
      this.videoMaterials.update(map => ({ ...map, [created.id]: [] }));
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
      // Remove materials cache for this video
      this.videoMaterials.update(map => {
        const next = { ...map };
        delete next[videoId];
        return next;
      });
      this.ui.showSuccess('Video eliminado');
    } catch {
      // Error toast already shown
    }
  }

  // ── Per-video material operations ────────────────────────────────────────────

  onVideoMaterialUploaded(videoId: string, material: MaterialResponse): void {
    this.videoMaterials.update(map => ({
      ...map,
      [videoId]: [...(map[videoId] ?? []), material],
    }));
    this.ui.showSuccess('Material agregado');
  }

  async downloadVideoMaterial(material: MaterialResponse): Promise<void> {
    try {
      const resp = await this.materialService.downloadUrl(material.id);
      window.open(resp.url, '_blank', 'noopener');
    } catch {
      // Error toast already shown
    }
  }

  async deleteVideoMaterial(videoId: string, material: MaterialResponse): Promise<void> {
    const confirmed = await this.ui.confirmDelete(
      `¿Eliminar "${material.nombre}"? Esta accion no se puede deshacer.`,
    );
    if (!confirmed) return;

    try {
      await this.materialService.remove(material.id);
      this.videoMaterials.update(map => ({
        ...map,
        [videoId]: (map[videoId] ?? []).filter(m => m.id !== material.id),
      }));
      this.ui.showSuccess('Material eliminado');
    } catch {
      // Error toast already shown
    }
  }

  // ── Thumbnail ────────────────────────────────────────────────────────────────

  onThumbnailUploaded(key: string): void {
    this.miniaturaKey.set(key);
    this.ui.showSuccess('Miniatura actualizada');
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

  navigateToEvaluation(): Promise<boolean> {
    return this.router.navigateByUrl(`/platform/creator/evaluacion-editar/${this.courseId}`);
  }

}
