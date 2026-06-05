/**
 * course-detail.component.ts — Alumno course detail page (C2.4).
 *
 * Replaces the PendingViewComponent stub for the courses/:id route.
 * Branches on the `enrolled` discriminator from the backend response:
 *  - enrolled=false → preview block (titulo, descripcion, creadorNombre, "Inscribirme")
 *  - enrolled=true  → full tree (secciones → VideoEmbed + materiales)
 *
 * "Inscribirme" calls enroll(id) → on success re-fetches getDetail(id) to flip to enrolled view.
 * Navigation uses ABSOLUTE /platform/... paths (C2.2 relative-nav bug prevention).
 */
import {
  Component,
  inject,
  signal,
  computed,
  OnInit,
} from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { SkeletonModule } from 'primeng/skeleton';

import { CourseCatalogService } from '@core/services/courseCatalogService/course-catalog.service';
import { VideoEmbedComponent } from '@shared/components/video-embed/video-embed.component';
import type {
  CourseDetailResponse,
  CourseDetailAlumnoResponse,
  CoursePreviewResponse,
} from '@core/services/courseCatalogService/course-catalog.dto';

@Component({
  selector: 'app-course-detail-alumno',
  standalone: true,
  imports: [SkeletonModule, VideoEmbedComponent],
  templateUrl: './course-detail.component.html',
  styleUrl: './course-detail.component.sass',
})
export class CourseDetailAlumnoComponent implements OnInit {
  private readonly catalogService = inject(CourseCatalogService);
  private readonly route = inject(ActivatedRoute);

  // ── State ──────────────────────────────────────────────────────────────────
  readonly detail = signal<CourseDetailResponse | null>(null);
  readonly loading = signal<boolean>(false);
  readonly enrolling = signal<boolean>(false);

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

  private courseId = '';

  ngOnInit(): void {
    this.courseId = this.route.snapshot.paramMap.get('id') ?? '';
    void this.loadDetail();
  }

  async loadDetail(): Promise<void> {
    if (!this.courseId) return;
    this.loading.set(true);
    try {
      const result = await this.catalogService.getDetail(this.courseId);
      this.detail.set(result);
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
}
