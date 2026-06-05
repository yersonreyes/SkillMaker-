import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { TooltipModule } from 'primeng/tooltip';
import { PopoverModule } from 'primeng/popover';
import { AuthService } from '@core/services/authService/auth.service';
import { HasRoleDirective } from '@shared/directives/has-role.directive';

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
export class PlatformLayoutComponent implements OnInit {
  protected auth = inject(AuthService);

  protected collapsed = signal<boolean>(this.readSidebarState());

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

  ngOnInit(): void {}

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
}
