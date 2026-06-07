/**
 * course-detail.component.ts — Alumno course detail page (C2.4).
 *
 * Updated in course-structure-v2:
 * - Renders video.descripcion and per-video materiales[]
 * - Course-level materiales REMOVED (now per-video)
 * - Course metadata: nivel, categorias, cantidadClases, horasVideo, horasPractico, miniaturaUrl
 *
 * Updated in course-player-progress (WU-2):
 * - 2-column player: LEFT stage (active video + materiales) + RIGHT curriculum
 * - activeVideoId signal: defaults to first incomplete video on load
 * - flatVideos / activeVideo / completedCount / progreso computed signals
 * - selectVideo() / markCompleted() methods
 *
 * Branches on the `enrolled` discriminator from the backend response:
 *  - enrolled=false → preview block (metadata + "Inscribirme") — UNCHANGED
 *  - enrolled=true  → 2-column player (stage + curriculum)
 */
import {
  Component,
  inject,
  signal,
  computed,
  OnInit,
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { SkeletonModule } from 'primeng/skeleton';

import { CourseCatalogService } from '@core/services/courseCatalogService/course-catalog.service';
import { CertificateService } from '@core/services/certificateService/certificate.service';
import { EvaluationService } from '@core/services/evaluationService/evaluation.service';
import { MaterialService } from '@core/services/materialService/material.service';
import { VideoEmbedComponent } from '@shared/components/video-embed/video-embed.component';
import type {
  CourseDetailResponse,
  CourseDetailAlumnoResponse,
  CoursePreviewResponse,
  VideoResponseItem,
} from '@core/services/courseCatalogService/course-catalog.dto';
import type { CertificateListItem } from '@core/services/certificateService/certificate.dto';
import type { EvaluationSummary } from '@core/services/evaluationService/evaluation.dto';

@Component({
  selector: 'app-course-detail-alumno',
  standalone: true,
  imports: [SkeletonModule, VideoEmbedComponent],
  templateUrl: './course-detail.component.html',
  styleUrl: './course-detail.component.sass',
})
export class CourseDetailAlumnoComponent implements OnInit {
  private readonly catalogService = inject(CourseCatalogService);
  private readonly certService = inject(CertificateService);
  private readonly evalService = inject(EvaluationService);
  readonly materialService = inject(MaterialService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  // ── State ──────────────────────────────────────────────────────────────────
  readonly detail = signal<CourseDetailResponse | null>(null);
  readonly loading = signal<boolean>(false);
  readonly enrolling = signal<boolean>(false);

  /** All user certificates — populated after loadDetail. */
  private readonly myCerts = signal<CertificateListItem[]>([]);

  /** Evaluation summary for the enrolled student — null means no evaluation available. */
  readonly evalSummary = signal<EvaluationSummary | null>(null);

  /**
   * courseId as a signal so `myCertificate` computed stays reactive when it updates.
   */
  private readonly courseIdSignal = signal<string>('');

  // ── Player state (course-player-progress WU-2) ──────────────────────────────

  /** Currently active video ID — drives the LEFT stage. */
  readonly activeVideoId = signal<string>('');

  /** Flat, ordered video list across all secciones (for resume + nav + counts). */
  readonly flatVideos = computed((): VideoResponseItem[] => {
    const d = this.enrolledDetail();
    if (!d) return [];
    return d.secciones.flatMap(s => s.videos);
  });

  /** The VideoResponseItem for activeVideoId (falls back to first video). */
  readonly activeVideo = computed((): VideoResponseItem | null =>
    this.flatVideos().find(v => v.id === this.activeVideoId()) ?? this.flatVideos()[0] ?? null,
  );

  /** Count of videos the caller has completed. */
  readonly completedCount = computed((): number =>
    this.flatVideos().filter(v => v.completado).length,
  );

  /**
   * Progress percentage rounded to nearest integer.
   * Uses cantidadClases from the course (authoritative total).
   */
  readonly progreso = computed((): number => {
    const total = this.enrolledDetail()?.cantidadClases ?? 0;
    return total === 0 ? 0 : Math.round((this.completedCount() / total) * 100);
  });

  // ── Computed discriminators ─────────────────────────────────────────────────
  readonly enrolled = computed(() => this.detail()?.enrolled ?? false);
  readonly preview = computed((): CoursePreviewResponse | null => {
    const d = this.detail();
    if (!d || d.enrolled) return null;
    return d as CoursePreviewResponse;
  });
  readonly enrolledDetail = computed((): CourseDetailAlumnoResponse | null => {
    const d = this.detail();
    if (!d || !d.enrolled) return null;
    return d as CourseDetailAlumnoResponse;
  });

  /**
   * The certificate for this specific course, if the user has earned it.
   */
  readonly myCertificate = computed((): CertificateListItem | null => {
    const certs = this.myCerts();
    const id = this.courseIdSignal();
    const found = certs.find(c => c.courseId === id);
    return found ?? null;
  });

  private courseId = '';

  ngOnInit(): void {
    this.courseId = this.route.snapshot.paramMap.get('id') ?? '';
    this.courseIdSignal.set(this.courseId);
    void this.loadDetail();
  }

  async loadDetail(): Promise<void> {
    if (!this.courseId) return;
    this.loading.set(true);
    try {
      const [result, certs] = await Promise.all([
        this.catalogService.getDetail(this.courseId),
        this.certService.getMyCertificates().catch(() => [] as CertificateListItem[]),
      ]);
      this.detail.set(result);
      this.myCerts.set(certs);

      if (result?.enrolled) {
        this.initActiveVideo();
        const summary = await this.evalService
          .getCourseEvaluationSummary(this.courseId)
          .catch(() => null);
        this.evalSummary.set(summary);
      } else {
        this.evalSummary.set(null);
      }
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    } finally {
      this.loading.set(false);
    }
  }

  /**
   * Lightweight resume (D5): open to first incomplete video, else first.
   * Called from loadDetail after this.detail.set(result) for enrolled branch.
   */
  private initActiveVideo(): void {
    const vids = this.flatVideos();
    const firstIncomplete = vids.find(v => !v.completado);
    this.activeVideoId.set((firstIncomplete ?? vids[0])?.id ?? '');
  }

  /** Sets the active video (drives the LEFT stage). */
  selectVideo(id: string): void {
    this.activeVideoId.set(id);
  }

  /**
   * Toggles completado for the given video.
   * Calls PUT /api/videos/:id/progress; on success optimistically updates the detail signal.
   */
  async markCompleted(video: VideoResponseItem): Promise<void> {
    const next = !video.completado;
    try {
      await this.catalogService.markVideoProgress(video.id, next);
      // Optimistic update: rebuild detail signal with the toggled video
      this.detail.update(d => {
        if (!d || !d.enrolled) return d;
        const enrolled = d as CourseDetailAlumnoResponse;
        return {
          ...enrolled,
          secciones: enrolled.secciones.map(s => ({
            ...s,
            videos: s.videos.map(v =>
              v.id === video.id ? { ...v, completado: next } : v,
            ),
          })),
        };
      });
    } catch {
      // Error toast shown by builder
    }
  }

  /** Called from the "Inscribirme" button in the preview branch. */
  async onEnroll(): Promise<void> {
    if (!this.courseId) return;
    this.enrolling.set(true);
    try {
      await this.catalogService.enroll(this.courseId);
      await this.loadDetail();
    } catch {
      // Error toast shown by builder
    } finally {
      this.enrolling.set(false);
    }
  }

  /** Navigate to the evaluation attempt page. */
  goToEvaluation(evaluationId: string): void {
    void this.router.navigate(['/platform/evaluations', evaluationId]);
  }

  /** Fetch presigned download URL and open in new tab. */
  async downloadCertificate(certId: string): Promise<void> {
    try {
      const res = await this.certService.getDownloadUrl(certId);
      if (res.url) {
        window.open(res.url, '_blank');
      }
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    }
  }

  /** Download material — resolves ownership chain on backend, works for enrolled alumno. */
  async downloadMaterial(materialId: string): Promise<void> {
    try {
      const res = await this.materialService.downloadUrl(materialId);
      window.open(res.url, '_blank', 'noopener');
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    }
  }
}
