import { Routes } from '@angular/router';
import { roleGuard } from '@core/guards/role.guard';

export const SUPERVISOR_ROUTES: Routes = [
  {
    path: 'team-progress',
    loadComponent: () =>
      import('./team-progress/team-progress.component').then(
        m => m.TeamProgressComponent,
      ),
    canActivate: [roleGuard],
    data: { title: 'Avance del equipo', roles: ['supervisor'] },
  },
  { path: '', redirectTo: 'team-progress', pathMatch: 'full' },
];
