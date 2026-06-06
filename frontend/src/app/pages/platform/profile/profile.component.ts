import { Component, inject, signal, OnInit } from '@angular/core';
import { DatePipe } from '@angular/common';
import { TableModule } from 'primeng/table';
import { ButtonModule } from 'primeng/button';
import { TagModule } from 'primeng/tag';
import { SkeletonModule } from 'primeng/skeleton';
import { AuthService } from '@core/services/authService/auth.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { SessionResponse } from '@core/services/authService/auth.res.dto';

@Component({
  selector: 'app-profile',
  standalone: true,
  imports: [DatePipe, TableModule, ButtonModule, TagModule, SkeletonModule],
  templateUrl: './profile.component.html',
  styleUrl: './profile.component.sass',
})
export class ProfileComponent implements OnInit {
  protected auth = inject(AuthService);
  private ui = inject(UiDialogService);

  sessions = signal<SessionResponse[]>([]);
  loadingSessions = signal(false);

  ngOnInit(): void {
    this.loadSessions();
  }

  async loadSessions(): Promise<void> {
    this.loadingSessions.set(true);
    try {
      const result = await this.auth.getMySessions();
      this.sessions.set(result);
    } finally {
      this.loadingSessions.set(false);
    }
  }

  async onRevoke(session: SessionResponse): Promise<void> {
    const ok = await this.ui.confirm({
      message: '¿Cerrar esta sesion?',
      header: 'Revocar sesion',
      icon: 'pi pi-sign-out',
      acceptLabel: 'Revocar',
    });
    if (!ok) return;
    await this.auth.revokeSession(session.id);
    this.ui.showSuccess('Sesion revocada');
    await this.loadSessions();
  }

  protected shortId(id: string | undefined | null): string {
    if (!id) return '----';
    return id.slice(0, 4).toUpperCase();
  }

  /** Light UA parse: extracts the browser name from a User-Agent string. */
  protected browserOf(ua?: string): string {
    if (!ua) return '—';
    if (ua.includes('Firefox')) return 'Firefox';
    if (ua.includes('Chrome') && !ua.includes('Edg')) return 'Chrome';
    if (ua.includes('Edg')) return 'Edge';
    if (ua.includes('Safari') && !ua.includes('Chrome')) return 'Safari';
    return ua.slice(0, 32);
  }
}
