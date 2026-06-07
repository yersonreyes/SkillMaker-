/**
 * catalog.component.ts — Alumno course catalog page (C2.4).
 *
 * Renders a grid of CourseCard components with:
 *  - Debounced search (300ms via rxjs Subject + debounceTime + takeUntilDestroyed)
 *  - Filter bar: nivel (p-select), categorias (p-multiselect), sort (p-select)
 *  - Immediate reload on any filter change (no debounce — only q debounces)
 *  - "Limpiar filtros" button when any filter is active
 *  - Distinct empty state: filtered vs. unfiltered
 *  - PrimeNG PaginatorModule, SkeletonModule, loading/empty states
 *
 * Cyanotype Workshop globals used: .page, .page__head, .page__title,
 * .page__eyebrow, .page__lede, .tk-dot, .empty.
 */
import {
  Component,
  computed,
  inject,
  signal,
  OnInit,
  DestroyRef,
} from '@angular/core';
import { Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { Subject } from 'rxjs';
import { debounceTime } from 'rxjs/operators';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { PaginatorModule, PaginatorState } from 'primeng/paginator';
import { SkeletonModule } from 'primeng/skeleton';
import { SelectModule } from 'primeng/select';
import { MultiSelectModule } from 'primeng/multiselect';

import { CourseCatalogService } from '@core/services/courseCatalogService/course-catalog.service';
import { CategoriaService } from '@core/services/categoriaService/categoria.service';
import { CourseCardComponent } from '@shared/components/course-card/course-card.component';
import type { CatalogCourseCard } from '@core/services/courseCatalogService/course-catalog.dto';
import type { CategoriaItem } from '@core/services/categoriaService/categoria.dto';

/** Nivel options for the filter p-select. Todos = null (no filter). */
const NIVEL_OPTIONS = [
  { label: 'Todos', value: null },
  { label: 'Básico', value: 'basico' },
  { label: 'Intermedio', value: 'intermedio' },
  { label: 'Avanzado', value: 'avanzado' },
];

/** Sort options for the filter p-select. */
const SORT_OPTIONS = [
  { label: 'Recientes', value: 'recientes' },
  { label: 'Título', value: 'titulo' },
];

@Component({
  selector: 'app-catalog',
  standalone: true,
  imports: [
    CourseCardComponent,
    PaginatorModule,
    SkeletonModule,
    SelectModule,
    MultiSelectModule,
    FormsModule,
  ],
  templateUrl: './catalog.component.html',
  styleUrl: './catalog.component.sass',
})
export class CatalogComponent implements OnInit {
  private readonly catalogService = inject(CourseCatalogService);
  private readonly categoriaService = inject(CategoriaService);
  private readonly router = inject(Router);
  private readonly destroyRef = inject(DestroyRef);

  // ── State (signals) ─────────────────────────────────────────────────────────
  readonly courses = signal<CatalogCourseCard[]>([]);
  readonly total = signal<number>(0);
  readonly page = signal<number>(1);
  readonly size = signal<number>(12);
  readonly q = signal<string>('');
  readonly loading = signal<boolean>(false);

  // ── Filter signals ───────────────────────────────────────────────────────────
  /** null = "Todos" (no nivel filter) */
  readonly nivel = signal<string | null>(null);
  readonly categoriaIds = signal<string[]>([]);
  /** Default = 'recientes' (matches backend default) */
  readonly sort = signal<string>('recientes');
  readonly categorias = signal<CategoriaItem[]>([]);

  // ── Computed ─────────────────────────────────────────────────────────────────
  readonly hasActiveFilters = computed(
    () =>
      this.nivel() !== null ||
      this.categoriaIds().length > 0 ||
      this.sort() !== 'recientes' ||
      this.q() !== '',
  );

  // ── Exposed options for template ─────────────────────────────────────────────
  readonly nivelOptions = NIVEL_OPTIONS;
  readonly sortOptions = SORT_OPTIONS;

  // ── ngModel bridges (two-way binding for p-select / p-multiselect) ───────────
  // PrimeNG's [(ngModel)] needs a getter+setter bridge to signals.
  get nivelModel(): string | null { return this.nivel(); }
  set nivelModel(v: string | null) { this.nivel.set(v); }

  get categoriaIdsModel(): string[] { return this.categoriaIds(); }
  set categoriaIdsModel(v: string[]) { this.categoriaIds.set(v ?? []); }

  get sortModel(): string { return this.sort(); }
  set sortModel(v: string) { this.sort.set(v); }

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
    // Load categorias for the filter multiselect
    void this.categoriaService.getAll().then(cats => this.categorias.set(cats));
    void this.load();
  }

  /** Called from template's input (change) event. */
  onSearchInput(value: string): void {
    this.searchSubject.next(value);
  }

  /**
   * Called by p-select/p-multiselect (onChange) events.
   * Resets page to 1 and immediately reloads (no debounce).
   * q keeps its existing debounce path unchanged.
   */
  onFilterChange(): void {
    this.page.set(1);
    void this.load();
  }

  /** Called from "Limpiar filtros" button. Resets all filters and reloads. */
  clearFilters(): void {
    this.nivel.set(null);
    this.categoriaIds.set([]);
    this.sort.set('recientes');
    this.q.set('');
    this.page.set(1);
    void this.load();
  }

  /** Called from PrimeNG Paginator's (onPageChange) event. */
  onPageChange(event: PaginatorState): void {
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
        this.nivel() ?? undefined,
        this.categoriaIds(),
        this.sort(),
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
