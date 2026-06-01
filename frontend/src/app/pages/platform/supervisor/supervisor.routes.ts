import { Routes } from '@angular/router';
import { PendingViewComponent } from '@shared/components/pending-view/pending-view.component';

export const SUPERVISOR_ROUTES: Routes = [
  {
    path: 'team-progress',
    component: PendingViewComponent,
    data: { title: 'Avance del equipo' },
  },
  { path: '', redirectTo: 'team-progress', pathMatch: 'full' },
];
