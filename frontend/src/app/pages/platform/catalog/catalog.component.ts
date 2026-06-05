/**
 * catalog.component.ts — Alumno course catalog page (C2.4).
 *
 * Replaces the stub. Renders a grid of CourseCard components with debounced
 * search (300ms via rxjs Subject + debounceTime + takeUntilDestroyed),
 * PrimeNG PaginatorModule, and loading/empty states.
 *
 * Cyanotype Workshop globals used: .page, .page__head, .page__title,
 * .page__eyebrow, .page__lede, .tk-dot, .empty.
 */
import {
  Component,
  inject,
  signal,
  OnInit,
  DestroyRef,
} from '@angular/core';
import { Router } from '@angular/router';
import { Subject } from 'rxjs';
import { debounceTime } from 'rxjs/operators';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { PaginatorModule, PaginatorState } from 'primeng/paginator';
import { SkeletonModule } from 'primeng/skeleton';

import { CourseCatalogService } from '@core/services/courseCatalogService/course-catalog.service';
import { CourseCardComponent } from '@shared/components/course-card/course-card.component';
import type { CatalogCourseCard } from '@core/services/courseCatalogService/course-catalog.dto';

@Component({
  selector: 'app-catalog',
  standalone: true,
  imports: [
    CourseCardComponent,
    PaginatorModule,
    SkeletonModule,
  ],
  templateUrl: './catalog.component.html',
  styleUrl: './catalog.component.sass',
})
export class CatalogComponent implements OnInit {
  private readonly catalogService = inject(CourseCatalogService);
  private readonly router = inject(Router);
  private readonly destroyRef = inject(DestroyRef);

  // ── State (signals) ─────────────────────────────────────────────────────────
  readonly courses = signal<CatalogCourseCard[]>([]);
  readonly total = signal<number>(0);
  readonly page = signal<number>(1);
  readonly size = signal<number>(12);
  readonly q = signal<string>('');
  readonly loading = signal<boolean>(false);

  // ── Debounced search ─────────────────────────────────────────────────────────
  /** Public so specs can push values directly. */
  readonly searchSubject = new Subject<string>();

  constructor() {
    // Wire debounce: 300ms after last emission → set q, reset page, reload.
    this.searchSubject.pipe(
      debounceTime(300),
      takeUntilDestroyed(this.destroyRef),
    ).subscribe(value => {
      this.q.set(value);
      this.page.set(1);
      void this.load();
    });
  }

  ngOnInit(): void {
    void this.load();
  }

  /** Called from template's input (change) event. */
  onSearchInput(value: string): void {
    this.searchSubject.next(value);
  }

  /** Called from PrimeNG Paginator's (onPageChange) event. */
  onPageChange(event: PaginatorState): void {
    // PrimeNG Paginator emits 0-indexed page (optional in PaginatorState);
    // we use a 1-indexed API and fall back to current values if absent.
    this.page.set((event.page ?? 0) + 1);
    this.size.set(event.rows ?? this.size());
    void this.load();
  }

  async load(): Promise<void> {
    this.loading.set(true);
    try {
      const result = await this.catalogService.getCatalog(
        this.page(),
        this.size(),
        this.q(),
      );
      this.courses.set(result.items);
      this.total.set(result.total);
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    } finally {
      this.loading.set(false);
    }
  }

  openDetail(course: CatalogCourseCard): void {
    void this.router.navigate(['/platform/courses', course.id]);
  }
}
