/**
 * categorias.component.spec.ts — admin categoría CRUD.
 *
 * Covers: load on init, create, edit (update), delete (confirm flow + reject),
 * nombre validation, and a render smoke test (template binding).
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { CategoriasComponent } from './categorias.component';
import { CategoriaService } from '@core/services/categoriaService/categoria.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { CategoriaItem } from '@core/services/categoriaService/categoria.dto';

const MOCK: CategoriaItem[] = [
  { id: 'c1', nombre: 'Backend', slug: 'backend' },
  { id: 'c2', nombre: 'Frontend', slug: 'frontend' },
];

describe('CategoriasComponent', () => {
  let serviceSpy: Partial<CategoriaService>;
  let uiSpy: Partial<UiDialogService>;

  function build() {
    TestBed.configureTestingModule({
      imports: [CategoriasComponent],
      providers: [
        { provide: CategoriaService, useValue: serviceSpy },
        { provide: UiDialogService, useValue: uiSpy },
        ConfirmationService,
        MessageService,
        provideAnimationsAsync(),
      ],
    });
    return TestBed.createComponent(CategoriasComponent);
  }

  beforeEach(() => {
    serviceSpy = {
      getAll: vi.fn().mockResolvedValue([...MOCK]),
      create: vi.fn().mockResolvedValue({ id: 'c3', nombre: 'DevOps', slug: 'devops' }),
      update: vi.fn().mockResolvedValue({ id: 'c1', nombre: 'Backend Pro', slug: 'backend-pro' }),
      delete: vi.fn().mockResolvedValue(undefined),
    };
    uiSpy = {
      confirmDelete: vi.fn().mockResolvedValue(true),
      showSuccess: vi.fn(),
      showError: vi.fn(),
    };
  });

  it('loads categorías on init', async () => {
    const comp = build().componentInstance;
    comp.ngOnInit();
    await Promise.resolve();
    expect(serviceSpy.getAll).toHaveBeenCalled();
    expect(comp.categorias()).toHaveLength(2);
  });

  it('create mode: save() calls service.create and appends sorted', async () => {
    const comp = build().componentInstance;
    comp.categorias.set([...MOCK]);
    comp.openCreateDialog();
    expect(comp.editingId()).toBeNull();
    comp.formNombre.set('DevOps');

    await comp.save();

    expect(serviceSpy.create).toHaveBeenCalledWith({ nombre: 'DevOps' });
    expect(comp.categorias().map(c => c.nombre)).toEqual(['Backend', 'DevOps', 'Frontend']);
    expect(comp.dialogVisible()).toBe(false);
  });

  it('edit mode: save() calls service.update and replaces the row', async () => {
    const comp = build().componentInstance;
    comp.categorias.set([...MOCK]);
    comp.openEditDialog(MOCK[0]);
    expect(comp.editingId()).toBe('c1');
    expect(comp.formNombre()).toBe('Backend');
    comp.formNombre.set('Backend Pro');

    await comp.save();

    expect(serviceSpy.update).toHaveBeenCalledWith('c1', { nombre: 'Backend Pro' });
    expect(comp.categorias().find(c => c.id === 'c1')?.nombre).toBe('Backend Pro');
  });

  it('save() is a no-op when nombre is too short (< 2 chars)', async () => {
    const comp = build().componentInstance;
    comp.openCreateDialog();
    comp.formNombre.set('a');
    await comp.save();
    expect(serviceSpy.create).not.toHaveBeenCalled();
  });

  it('remove() confirms then calls service.delete and removes the row', async () => {
    const comp = build().componentInstance;
    comp.categorias.set([...MOCK]);

    await comp.remove(MOCK[0]);

    expect(uiSpy.confirmDelete).toHaveBeenCalled();
    expect(serviceSpy.delete).toHaveBeenCalledWith('c1');
    expect(comp.categorias().some(c => c.id === 'c1')).toBe(false);
  });

  it('remove() does NOT delete when confirm is rejected', async () => {
    uiSpy.confirmDelete = vi.fn().mockResolvedValue(false);
    const comp = build().componentInstance;
    comp.categorias.set([...MOCK]);

    await comp.remove(MOCK[0]);

    expect(serviceSpy.delete).not.toHaveBeenCalled();
    expect(comp.categorias()).toHaveLength(2);
  });

  it('renders without template errors (header + create button)', async () => {
    const fixture = build();
    fixture.componentInstance.categorias.set([...MOCK]);
    fixture.componentInstance.loading.set(false);
    fixture.detectChanges(); // throws on any template binding error
    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Categorías');
    expect(text).toContain('Nueva categoría');
    // p-table row rendering is unreliable in jsdom; row content is asserted via state, not DOM.
    expect(fixture.componentInstance.categorias()).toHaveLength(2);
  });
});
