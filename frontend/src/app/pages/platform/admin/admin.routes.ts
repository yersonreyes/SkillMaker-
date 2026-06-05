import { Routes } from '@angular/router';
import { roleGuard } from '@core/guards/role.guard';

export const ADMIN_ROUTES: Routes = [
  {
    path: 'approvals',
    loadComponent: () =>
      import('./approvals/aprovaciones.component').then(
        m => m.AprovacionesComponent,
      ),
    canActivate: [roleGuard],
    data: { title: 'Aprobaciones', roles: ['administrador'] },
  },
  {
    path: 'user-management',
    loadComponent: () =>
      import('./user-management/user-management.component').then(
        m => m.UserManagementComponent,
      ),
    canActivate: [roleGuard],
    data: { title: 'Gestion de usuarios', roles: ['administrador'] },
  },
  {
    path: 'supervision',
    loadComponent: () =>
      import('./supervision/supervision.component').then(
        m => m.SupervisionComponent,
      ),
    canActivate: [roleGuard],
    data: { title: 'Asignacion supervisor-empleados', roles: ['administrador'] },
  },
  {
    path: 'reports',
    loadComponent: () =>
      import('@shared/components/pending-view/pending-view.component').then(
        m => m.PendingViewComponent,
      ),
    data: { title: 'Reportes globales' },
  },
  { path: '', redirectTo: 'approvals', pathMatch: 'full' },
];
