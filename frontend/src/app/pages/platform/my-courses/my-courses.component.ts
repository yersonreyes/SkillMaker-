/**
 * my-courses.component.ts — Alumno enrolled courses page (C2.4).
 *
 * Replaces the stub. Uses PrimeNG TableModule to list the caller's enrolled courses.
 * "Completado" badge (.tag) when completado=true.
 * "Continuar" button navigates with ABSOLUTE /platform/courses/:id path (C2.2 anti-pattern prevention).
 */
import {
  Component,
  inject,
  signal,
  OnInit,
} from '@angular/core';
import { Router } from '@angular/router';
import { DatePipe } from '@angular/common';
import { TableModule } from 'primeng/table';
import { TagModule } from 'primeng/tag';
import { SkeletonModule } from 'primeng/skeleton';

import { CourseCatalogService } from '@core/services/courseCatalogService/course-catalog.service';
import type { MyCourseItem } from '@core/services/courseCatalogService/course-catalog.dto';

@Component({
  selector: 'app-my-courses',
  standalone: true,
  imports: [
    DatePipe,
    TableModule,
    TagModule,
    SkeletonModule,
  ],
  templateUrl: './my-courses.component.html',
  styleUrl: './my-courses.component.sass',
})
export class MyCoursesComponent implements OnInit {
  private readonly catalogService = inject(CourseCatalogService);
  private readonly router = inject(Router);

  // ── State ──────────────────────────────────────────────────────────────────
  readonly courses = signal<MyCourseItem[]>([]);
  readonly loading = signal<boolean>(false);

  ngOnInit(): void {
    void this.loadCourses();
  }

  async loadCourses(): Promise<void> {
    this.loading.set(true);
    try {
      const items = await this.catalogService.getMyCourses();
      this.courses.set(items);
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    } finally {
      this.loading.set(false);
    }
  }

  /** Navigate to course detail using ABSOLUTE path (no relative nav — C2.2 fix). */
  goToCourse(course: MyCourseItem): void {
    void this.router.navigate(['/platform/courses', course.courseId]);
  }
}
