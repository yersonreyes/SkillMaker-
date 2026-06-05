/**
 * course-detail.component.ts — Alumno course detail page (C2.4).
 *
 * Replaces the PendingViewComponent stub for the courses/:id route.
 * Branches on the `enrolled` discriminator from the backend response:
 *  - enrolled=false → preview block (titulo, descripcion, creadorNombre, "Inscribirme")
 *  - enrolled=true  → full tree (secciones → VideoEmbed + materiales)
 *
 * "Inscribirme" calls enroll(id) → on success re-fetches getDetail(id) to flip to enrolled view.
 * "Mi certificado" (C5.1) appears when a certificate matching this courseId exists.
 * "Rendir evaluación" (student-eval-discovery) appears when enrolled and the course has an evaluation.
 * Navigation uses ABSOLUTE /platform/... paths (C2.2 relative-nav bug prevention).
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
import { VideoEmbedComponent } from '@shared/components/video-embed/video-embed.component';
import type {
  CourseDetailResponse,
  CourseDetailAlumnoResponse,
  CoursePreviewResponse,
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
   * Set from ActivatedRoute in ngOnInit.
   */
  private readonly courseIdSignal = signal<string>('');

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
   * Match is done by courseId from the certificate list.
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

      // Fetch eval summary when enrolled — catch 404 silently (no evaluation = null).
      if (result?.enrolled) {
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

  /** Called from the "Inscribirme" button in the preview branch. */
  async onEnroll(): Promise<void> {
    if (!this.courseId) return;
    this.enrolling.set(true);
    try {
      await this.catalogService.enroll(this.courseId);
      // Re-fetch detail to flip to enrolled view (no full page reload)
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
}
