import {
  Component,
  inject,
  signal,
  OnInit,
} from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { DatePipe } from '@angular/common';
import { TableModule } from 'primeng/table';
import { SelectModule } from 'primeng/select';
import { ButtonModule } from 'primeng/button';
import { TagModule } from 'primeng/tag';
import { TooltipModule } from 'primeng/tooltip';

import { SupervisionService } from '@core/services/supervisionService/supervision.service';
import { UserService } from '@core/services/userService/user.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { SupervisionItem, UserListItem } from '@core/services/userService/user.res.dto';

@Component({
  selector: 'app-supervision',
  standalone: true,
  imports: [
    FormsModule,
    DatePipe,
    TableModule,
    SelectModule,
    ButtonModule,
    TagModule,
    TooltipModule,
  ],
  templateUrl: './supervision.component.html',
})
export class SupervisionComponent implements OnInit {
  private readonly supervisionService = inject(SupervisionService);
  private readonly userService = inject(UserService);
  private readonly ui = inject(UiDialogService);

  // ── State ──────────────────────────────────────────────────────────────────
  readonly supervisions = signal<SupervisionItem[]>([]);
  readonly users = signal<UserListItem[]>([]);
  readonly loading = signal<boolean>(false);
  readonly creating = signal<boolean>(false);

  // ── Create form state ──────────────────────────────────────────────────────
  readonly selectedSupervisorId = signal<string>('');
  readonly selectedEmpleadoId = signal<string>('');

  ngOnInit(): void {
    void this.loadAll();
  }

  // ── Load ───────────────────────────────────────────────────────────────────

  private async loadAll(): Promise<void> {
    this.loading.set(true);
    try {
      const [supervisions, usersPage] = await Promise.all([
        this.supervisionService.getAll(),
        this.userService.getAll({ page: 1, size: 100 }),
      ]);
      this.supervisions.set(supervisions);
      this.users.set(usersPage.items);
    } catch {
      // Error toast shown by builder
    } finally {
      this.loading.set(false);
    }
  }

  // ── User picker options ────────────────────────────────────────────────────

  get userOptions(): { label: string; value: string }[] {
    return this.users().map(u => ({ label: `${u.nombre} (${u.email})`, value: u.id }));
  }

  // ── Create ─────────────────────────────────────────────────────────────────

  /** Create a new supervision relation. Guards self-supervision client-side. */
  async createSupervision(): Promise<void> {
    const supervisorId = this.selectedSupervisorId();
    const empleadoId = this.selectedEmpleadoId();

    if (!supervisorId || !empleadoId) {
      this.ui.showWarn('Selecciona supervisor y empleado');
      return;
    }

    if (supervisorId === empleadoId) {
      this.ui.showWarn('Atencion', 'Un usuario no puede supervisarse a si mismo.');
      return;
    }

    this.creating.set(true);
    try {
      const created = await this.supervisionService.create({ supervisorId, empleadoId });
      this.supervisions.update(list => [...list, created]);
      this.selectedSupervisorId.set('');
      this.selectedEmpleadoId.set('');
      this.ui.showSuccess('Relacion de supervision creada');
    } catch (err) {
      if (err instanceof HttpErrorResponse && err.status === 409) {
        this.ui.showError(
          'Conflicto de supervision',
          'El empleado ya tiene un supervisor asignado. Elimine la relacion existente primero.',
        );
      }
      // Other errors handled by builder
    } finally {
      this.creating.set(false);
    }
  }

  // ── Remove ─────────────────────────────────────────────────────────────────

  /** Remove a supervision relation after confirmation. */
  async removeSupervision(supervision: SupervisionItem): Promise<void> {
    const confirmed = await this.ui.confirmDelete(
      `¿Eliminar la supervision de ${supervision.supervisorName} sobre ${supervision.empleadoName}?`,
    );
    if (!confirmed) return;

    try {
      await this.supervisionService.delete(supervision.id);
      this.supervisions.update(list => list.filter(s => s.id !== supervision.id));
      this.ui.showSuccess('Relacion de supervision eliminada');
    } catch {
      // Error toast shown by builder
    }
  }
}
