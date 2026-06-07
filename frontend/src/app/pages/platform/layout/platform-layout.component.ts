import { Component, OnInit, OnDestroy, inject, signal, computed } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet, Router } from '@angular/router';
import { TooltipModule } from 'primeng/tooltip';
import { PopoverModule } from 'primeng/popover';
import { AuthService } from '@core/services/authService/auth.service';
import { HasRoleDirective } from '@shared/directives/has-role.directive';
import { NotificationService } from '@core/services/notificationService/notification.service';
import type { NotificationItem } from '@core/services/notificationService/notification.dto';

interface NavItem {
  label: string;
  icon: string;
  route: string;
  queryParams?: Record<string, string | number>;
}

@Component({
  selector: 'app-platform-layout',
  standalone: true,
  imports: [
    RouterOutlet,
    RouterLink,
    RouterLinkActive,
    TooltipModule,
    PopoverModule,
    HasRoleDirective,
  ],
  templateUrl: './platform-layout.component.html',
  styleUrl: './platform-layout.component.sass',
})
export class PlatformLayoutComponent implements OnInit, OnDestroy {
  protected auth = inject(AuthService);
  private readonly notifService = inject(NotificationService);
  private readonly router = inject(Router);

  protected collapsed = signal<boolean>(this.readSidebarState());

  notifications = signal<NotificationItem[]>([]);
  unreadCount = signal<number>(0);

  private pollId: ReturnType<typeof setInterval> | null = null;

  protected userInitials = computed(() => {
    const name = this.auth.user()?.nombre ?? '';
    return (
      name
        .split(' ')
        .filter(Boolean)
        .slice(0, 2)
        .map(p => p[0]?.toUpperCase())
        .join('') || '?'
    );
  });

  protected primaryRole = computed(() => {
    const r = this.auth.userRoles();
    if (r.includes('administrador')) return 'Administrador';
    if (r.includes('supervisor')) return 'Supervisor';
    if (r.includes('creador')) return 'Creador';
    return 'Alumno';
  });

  protected commonItems: NavItem[] = [
    { label: 'Catalogo',     icon: 'pi pi-th-large',  route: '/platform/catalog' },
    { label: 'Mis cursos',   icon: 'pi pi-book',      route: '/platform/my-courses' },
    { label: 'Certificados', icon: 'pi pi-verified',  route: '/platform/certificates' },
    { label: 'Insignias',    icon: 'pi pi-star',      route: '/platform/badges' },
    { label: 'Perfil',       icon: 'pi pi-user',      route: '/platform/profile' },
  ];

  protected creatorItems: NavItem[] = [
    { label: 'Mi contenido', icon: 'pi pi-folder',      route: '/platform/creator/mi-contenido' },
    { label: 'Crear curso',  icon: 'pi pi-plus-circle', route: '/platform/creator/mi-contenido', queryParams: { nuevo: 1 } },
  ];

  protected supervisorItems: NavItem[] = [
    { label: 'Avance equipo', icon: 'pi pi-chart-line', route: '/platform/supervisor/team-progress' },
  ];

  protected adminItems: NavItem[] = [
    { label: 'Aprobaciones',      icon: 'pi pi-check-square',  route: '/platform/admin/approvals' },
    { label: 'Gestion usuarios',  icon: 'pi pi-users',          route: '/platform/admin/user-management' },
    { label: 'Supervision',       icon: 'pi pi-sitemap',        route: '/platform/admin/supervision' },
    { label: 'Reportes',          icon: 'pi pi-chart-bar',      route: '/platform/admin/reports' },
    { label: 'Reportes por curso', icon: 'pi pi-table',         route: '/platform/admin/course-reports' },
  ];

  ngOnInit(): void {
    void this.loadUnread();
    void this.loadList();
    this.pollId = setInterval(() => {
      void this.loadUnread();
      void this.loadList();
    }, 30000);
  }

  ngOnDestroy(): void {
    if (this.pollId !== null) {
      clearInterval(this.pollId);
      this.pollId = null;
    }
  }

  toggleSidebar(): void {
    const next = !this.collapsed();
    this.collapsed.set(next);
    localStorage.setItem('ui.sidebar.collapsed', String(next));
  }

  private readSidebarState(): boolean {
    return localStorage.getItem('ui.sidebar.collapsed') === 'true';
  }

  async logout(): Promise<void> {
    await this.auth.logout();
  }

  // ── Notification methods ──────────────────────────────────────────────────────

  onBellOpen(): void {
    void this.loadList();
  }

  async onNotificationClick(n: NotificationItem): Promise<void> {
    if (!n.leida) {
      await this.notifService.markRead(n.id!);
      this.unreadCount.update(c => Math.max(0, c - 1));
    }
    this.navigate(n);
  }

  async markAll(): Promise<void> {
    await this.notifService.markAllRead();
    this.unreadCount.set(0);
    this.notifications.update(list => list.map(x => ({ ...x, leida: true })));
  }

  relativeTime(iso: string | undefined): string {
    if (!iso) return '';
    const diff = Date.now() - new Date(iso).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return 'Ahora';
    if (mins < 60) return `Hace ${mins}m`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `Hace ${hours}h`;
    const days = Math.floor(hours / 24);
    return `Hace ${days}d`;
  }

  private async loadUnread(): Promise<void> {
    const count = await this.notifService.getUnreadCount();
    this.unreadCount.set(count);
  }

  private async loadList(): Promise<void> {
    const items = await this.notifService.getMine(1, 20);
    this.notifications.set(items);
  }

  private navigate(n: NotificationItem): void {
    const tipo = n.tipo ?? '';
    const refId = n.refId ?? '';
    if (tipo === 'curso_aprobado' || tipo === 'curso_rechazado') {
      void this.router.navigate(['/platform/courses', refId]);
    } else if (tipo === 'certificado_emitido') {
      void this.router.navigate(['/platform/certificates', refId]);
    }
  }
}
