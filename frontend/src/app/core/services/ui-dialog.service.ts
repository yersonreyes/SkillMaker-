import { Injectable, inject } from '@angular/core';
import { ConfirmationService, MessageService } from 'primeng/api';

export interface ConfirmOptions {
  message: string;
  header?: string;
  acceptLabel?: string;
  rejectLabel?: string;
  icon?: string;
}

@Injectable({ providedIn: 'root' })
export class UiDialogService {
  private confirmation = inject(ConfirmationService);
  private message = inject(MessageService);

  confirm(options: ConfirmOptions): Promise<boolean> {
    return new Promise(resolve => {
      this.confirmation.confirm({
        message: options.message,
        header: options.header ?? 'Confirmar',
        icon: options.icon ?? 'pi pi-exclamation-triangle',
        acceptLabel: options.acceptLabel ?? 'Si',
        rejectLabel: options.rejectLabel ?? 'No',
        accept: () => resolve(true),
        reject: () => resolve(false),
      });
    });
  }

  confirmDelete(message = '¿Estas seguro de eliminar este registro?'): Promise<boolean> {
    return this.confirm({ message, header: 'Zona peligrosa', icon: 'pi pi-trash', acceptLabel: 'Eliminar' });
  }

  confirmApprove(message = '¿Aprobar?'): Promise<boolean> {
    return this.confirm({ message, header: 'Aprobacion', icon: 'pi pi-check-circle', acceptLabel: 'Aprobar' });
  }

  showSuccess(summary: string, detail = '', life = 3000): void {
    this.message.add({ severity: 'success', summary, detail, life });
  }

  showError(summary: string, detail = '', life = 4000): void {
    this.message.add({ severity: 'error', summary, detail, life });
  }

  showInfo(summary: string, detail = '', life = 3000): void {
    this.message.add({ severity: 'info', summary, detail, life });
  }

  showWarn(summary: string, detail = '', life = 3000): void {
    this.message.add({ severity: 'warn', summary, detail, life });
  }
}
