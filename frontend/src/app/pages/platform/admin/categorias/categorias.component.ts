/**
 * categorias.component.ts — Admin CRUD for course categorías.
 *
 * Admin-only (roleGuard). Lists curated categorías and lets an admin create,
 * rename, and delete them. Delete is blocked by the backend (409 CATEGORIA_IN_USE)
 * when the categoría is still assigned to courses — the builder surfaces that message.
 */
import { Component, inject, signal, computed, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { TableModule } from 'primeng/table';
import { DialogModule } from 'primeng/dialog';
import { InputTextModule } from 'primeng/inputtext';
import { SkeletonModule } from 'primeng/skeleton';
import { TooltipModule } from 'primeng/tooltip';

import { CategoriaService } from '@core/services/categoriaService/categoria.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { CategoriaItem } from '@core/services/categoriaService/categoria.dto';

@Component({
  selector: 'app-categorias',
  standalone: true,
  imports: [
    FormsModule,
    TableModule,
    DialogModule,
    InputTextModule,
    SkeletonModule,
    TooltipModule,
  ],
  templateUrl: './categorias.component.html',
  styleUrl: './categorias.component.sass',
})
export class CategoriasComponent implements OnInit {
  private readonly service = inject(CategoriaService);
  private readonly ui = inject(UiDialogService);

  readonly categorias = signal<CategoriaItem[]>([]);
  readonly loading = signal<boolean>(false);

  // ── Create/edit dialog ────────────────────────────────────────────────────
  readonly dialogVisible = signal<boolean>(false);
  readonly editingId = signal<string | null>(null);
  readonly formNombre = signal<string>('');
  readonly saving = signal<boolean>(false);

  /** Mirrors backend binding: min=2, max=60. */
  readonly nombreInvalid = computed(() => {
    const n = this.formNombre().trim();
    return n.length < 2 || n.length > 60;
  });

  ngOnInit(): void {
    void this.load();
  }

  async load(): Promise<void> {
    this.loading.set(true);
    try {
      this.categorias.set(await this.service.getAll());
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    } finally {
      this.loading.set(false);
    }
  }

  openCreateDialog(): void {
    this.editingId.set(null);
    this.formNombre.set('');
    this.dialogVisible.set(true);
  }

  openEditDialog(cat: CategoriaItem): void {
    this.editingId.set(cat.id);
    this.formNombre.set(cat.nombre);
    this.dialogVisible.set(true);
  }

  async save(): Promise<void> {
    const nombre = this.formNombre().trim();
    if (this.nombreInvalid()) return;

    const id = this.editingId();
    this.saving.set(true);
    try {
      if (id) {
        const updated = await this.service.update(id, { nombre });
        this.categorias.update(list => this.sorted(list.map(c => (c.id === id ? updated : c))));
        this.ui.showSuccess('Categoría actualizada');
      } else {
        const created = await this.service.create({ nombre });
        this.categorias.update(list => this.sorted([...list, created]));
        this.ui.showSuccess('Categoría creada');
      }
      this.dialogVisible.set(false);
    } catch {
      // Error toast (incl. 409 duplicate) shown by HttpPromiseBuilderService
    } finally {
      this.saving.set(false);
    }
  }

  async remove(cat: CategoriaItem): Promise<void> {
    const confirmed = await this.ui.confirmDelete(
      `¿Eliminar la categoría "${cat.nombre}"? Si está asignada a algún curso, no podrá eliminarse.`,
    );
    if (!confirmed) return;

    try {
      await this.service.delete(cat.id);
      this.categorias.update(list => list.filter(c => c.id !== cat.id));
      this.ui.showSuccess('Categoría eliminada');
    } catch {
      // 409 (in use) / other errors shown by HttpPromiseBuilderService
    }
  }

  private sorted(list: CategoriaItem[]): CategoriaItem[] {
    return [...list].sort((a, b) => a.nombre.localeCompare(b.nombre));
  }
}
