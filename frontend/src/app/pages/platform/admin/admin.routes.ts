import { Routes } from '@angular/router';
import { PendingViewComponent } from '@shared/components/pending-view/pending-view.component';
import { roleGuard } from '@core/guards/role.guard';

export const ADMIN_ROUTES: Routes = [
  {
    path: 'approvals',
    component: PendingViewComponent,
    data: { title: 'Aprobaciones' },
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
    component: PendingViewComponent,
    data: { title: 'Reportes globales' },
  },
  { path: '', redirectTo: 'approvals', pathMatch: 'full' },
];
