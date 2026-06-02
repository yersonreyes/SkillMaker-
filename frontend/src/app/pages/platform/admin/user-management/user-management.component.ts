import {
  Component,
  inject,
  signal,
  OnInit,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { TableModule } from 'primeng/table';
import { DialogModule } from 'primeng/dialog';
import { CheckboxModule } from 'primeng/checkbox';
import { TagModule } from 'primeng/tag';
import { InputTextModule } from 'primeng/inputtext';
import { SelectModule } from 'primeng/select';
import { ButtonModule } from 'primeng/button';
import { SkeletonModule } from 'primeng/skeleton';
import { TooltipModule } from 'primeng/tooltip';

import { UserService } from '@core/services/userService/user.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { UserListItem, UserRole } from '@core/services/userService/user.res.dto';
import type { UserListParams } from '@core/services/userService/user.req.dto';

export type RoleDelta = { add: UserRole[]; remove: UserRole[] };

const ALL_ROLES: UserRole[] = ['alumno', 'creador', 'supervisor', 'administrador'];

@Component({
  selector: 'app-user-management',
  standalone: true,
  imports: [
    FormsModule,
    TableModule,
    DialogModule,
    CheckboxModule,
    TagModule,
    InputTextModule,
    SelectModule,
    ButtonModule,
    SkeletonModule,
    TooltipModule,
  ],
  templateUrl: './user-management.component.html',
})
export class UserManagementComponent implements OnInit {
  private readonly userService = inject(UserService);
  private readonly ui = inject(UiDialogService);

  // ── Pagination / list state ─────────────────────────────────────────────────
  readonly users = signal<UserListItem[]>([]);
  readonly total = signal<number>(0);
  readonly loading = signal<boolean>(false);

  readonly pageSize = signal<number>(20);

  // ── Filter state ────────────────────────────────────────────────────────────
  readonly filterQ = signal<string>('');
  readonly filterRole = signal<UserRole | ''>('');
  readonly filterActive = signal<boolean | undefined>(undefined);

  // ── Role filter options ─────────────────────────────────────────────────────
  readonly roleOptions = [
    { label: 'Todos los roles', value: '' },
    { label: 'Alumno',          value: 'alumno' },
    { label: 'Creador',         value: 'creador' },
    { label: 'Supervisor',      value: 'supervisor' },
    { label: 'Administrador',   value: 'administrador' },
  ];

  readonly activeOptions = [
    { label: 'Todos',     value: undefined },
    { label: 'Activos',   value: true },
    { label: 'Inactivos', value: false },
  ];

  // ── Edit dialog state ──────────────────────────────────────────────────────
  readonly dialogVisible = signal<boolean>(false);
  readonly editingUser = signal<UserListItem | null>(null);
  readonly selectedRoles = signal<UserRole[]>([]);
  readonly allRoles: UserRole[] = ALL_ROLES;
  readonly saving = signal<boolean>(false);

  // ── Cached query for lazy reload ────────────────────────────────────────────
  private lastParams: UserListParams = { page: 1, size: 20 };

  ngOnInit(): void {
    void this.load({ page: 1, size: 20 });
  }

  /** Called by PrimeNG Table's (onLazyLoad) event. */
  async onLazyLoad(event: { first?: number | null; rows?: number | null }): Promise<void> {
    const first = event.first ?? 0;
    const rows = event.rows ?? 20;
    const page = Math.floor(first / rows) + 1;
    const size = rows;
    await this.load({ ...this.lastParams, page, size });
  }

  /** Apply filters and reload from page 1. */
  async applyFilters(): Promise<void> {
    await this.load({
      page: 1,
      size: this.pageSize(),
      q: this.filterQ() || undefined,
      role: this.filterRole() || undefined,
      active: this.filterActive(),
    });
  }

  /** Open the edit roles dialog for a user. */
  async openEditDialog(user: UserListItem): Promise<void> {
    this.editingUser.set(user);
    this.selectedRoles.set([...user.roles]);
    this.dialogVisible.set(true);
  }

  closeDialog(): void {
    this.dialogVisible.set(false);
    this.editingUser.set(null);
  }

  /**
   * Compute add/remove delta between original roles and newly selected roles.
   * Exposed as public so specs can test the logic directly.
   */
  computeRoleDelta(original: UserRole[], selected: UserRole[]): RoleDelta {
    const origSet = new Set(original);
    const selSet = new Set(selected);

    const add = selected.filter(r => !origSet.has(r)) as UserRole[];
    const remove = original.filter(r => !selSet.has(r)) as UserRole[];

    return { add, remove };
  }

  /**
   * Called from the edit dialog's Save button.
   * `add` and `remove` are the computed delta.
   * If either list touches `administrador`, shows a confirm dialog first.
   */
  async saveRoles(userId: string, add: UserRole[], remove: UserRole[]): Promise<void> {
    const touchesAdmin =
      add.includes('administrador') || remove.includes('administrador');

    if (touchesAdmin) {
      const confirmed = await this.ui.confirm({
        message: '¿Estas seguro de modificar el rol de Administrador?',
        header: 'Cambio de administrador',
        icon: 'pi pi-shield',
        acceptLabel: 'Confirmar',
      });
      if (!confirmed) return;
    }

    this.saving.set(true);
    try {
      const updated = await this.userService.updateRoles(userId, { add, remove });
      // Refresh the row in the list
      this.users.update(list =>
        list.map(u => (u.id === userId ? { ...u, roles: updated.roles } : u)),
      );
      this.ui.showSuccess('Roles actualizados');
      this.closeDialog();
    } finally {
      this.saving.set(false);
    }
  }

  /** Called from the Save button in the dialog — computes delta and calls saveRoles. */
  async onSaveDialog(): Promise<void> {
    const user = this.editingUser();
    if (!user) return;
    const delta = this.computeRoleDelta(user.roles, this.selectedRoles());
    await this.saveRoles(user.id, delta.add, delta.remove);
  }

  /**
   * Toggle a user's active status. Shows a confirm dialog before calling the API.
   */
  async toggleActive(user: UserListItem): Promise<void> {
    const nextActive = !user.activo;
    const actionLabel = nextActive ? 'Activar' : 'Desactivar';

    const confirmed = await this.ui.confirm({
      message: `¿${actionLabel} al usuario ${user.nombre}?`,
      header: `${actionLabel} usuario`,
      icon: nextActive ? 'pi pi-user-plus' : 'pi pi-ban',
      acceptLabel: actionLabel,
    });

    if (!confirmed) return;

    try {
      const updated = await this.userService.setActive(user.id, nextActive);
      this.users.update(list =>
        list.map(u => (u.id === user.id ? { ...u, activo: updated.activo } : u)),
      );
      this.ui.showSuccess(`Usuario ${actionLabel.toLowerCase()}do`);
    } catch {
      // Error toast already shown by HttpPromiseBuilderService
    }
  }

  /** Toggle a role in the edit dialog selection. */
  toggleRole(role: UserRole): void {
    const current = this.selectedRoles();
    if (current.includes(role)) {
      this.selectedRoles.set(current.filter(r => r !== role));
    } else {
      this.selectedRoles.set([...current, role]);
    }
  }

  isRoleSelected(role: UserRole): boolean {
    return this.selectedRoles().includes(role);
  }

  // ── Private ─────────────────────────────────────────────────────────────────

  private async load(params: UserListParams): Promise<void> {
    this.lastParams = params;
    this.loading.set(true);
    try {
      const page = await this.userService.getAll(params);
      this.users.set(page.items);
      this.total.set(page.total);
      this.pageSize.set(params.size ?? 20);
    } catch {
      // Error toast already shown by builder
    } finally {
      this.loading.set(false);
    }
  }
}
