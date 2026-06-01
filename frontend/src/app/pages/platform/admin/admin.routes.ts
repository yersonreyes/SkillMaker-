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
  { path: '', redirectTo: 'approvals', pathMatch: 'full' },
];
