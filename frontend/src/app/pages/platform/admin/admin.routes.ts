import { Routes } from '@angular/router';
import { PendingViewComponent } from '@shared/components/pending-view/pending-view.component';

export const ADMIN_ROUTES: Routes = [
  {
    path: 'approvals',
    component: PendingViewComponent,
    data: { title: 'Aprobaciones' },
  },
  {
    path: 'user-management',
    component: PendingViewComponent,
    data: { title: 'Gestion de usuarios' },
  },
  {
    path: 'supervision',
    component: PendingViewComponent,
    data: { title: 'Asignacion supervisor-empleados' },
  },
  {
    path: 'reports',
    component: PendingViewComponent,
    data: { title: 'Reportes globales' },
  },
  { path: '', redirectTo: 'approvals', pathMatch: 'full' },
];
