import { Routes } from '@angular/router';
import { PlatformLayoutComponent } from './layout/platform-layout.component';
import { PendingViewComponent } from '@shared/components/pending-view/pending-view.component';
import { roleGuard } from '@core/guards/role.guard';

export const PLATFORM_ROUTES: Routes = [
  {
    path: '',
    component: PlatformLayoutComponent,
    children: [
      {
        path: 'catalog',
        loadComponent: () =>
          import('./catalog/catalog.component').then(m => m.CatalogComponent),
      },
      {
        path: 'my-courses',
        loadComponent: () =>
          import('./my-courses/my-courses.component').then(m => m.MyCoursesComponent),
      },
      {
        path: 'profile',
        loadComponent: () =>
          import('./profile/profile.component').then(m => m.ProfileComponent),
      },

      // Course detail — alumno-facing (C2.4). Branches on enrolled flag.
      {
        path: 'courses/:id',
        loadComponent: () =>
          import('./course-detail/course-detail.component').then(m => m.CourseDetailAlumnoComponent),
        data: { title: 'Detalle del curso' },
      },
      {
        // Student attempt page — replaces the evaluations/:id stub (C3.2).
        // :id = evaluationId. No role guard — any authenticated user may attempt.
        path: 'evaluations/:id',
        loadComponent: () =>
          import('./evaluacion-tomar/evaluacion-tomar.component').then(m => m.EvaluacionTomarComponent),
        data: { title: 'Evaluacion' },
      },
      { path: 'certificates', component: PendingViewComponent, data: { title: 'Certificados' } },
      { path: 'badges',       component: PendingViewComponent, data: { title: 'Insignias' } },

      // Sub-routers by role
      {
        path: 'creator',
        canActivate: [roleGuard],
        data: { roles: ['creador'] },
        loadChildren: () =>
          import('./creator/creator.routes').then(m => m.CREATOR_ROUTES),
      },
      {
        path: 'admin',
        canActivate: [roleGuard],
        data: { roles: ['administrador'] },
        loadChildren: () =>
          import('./admin/admin.routes').then(m => m.ADMIN_ROUTES),
      },
      {
        path: 'supervisor',
        canActivate: [roleGuard],
        data: { roles: ['supervisor'] },
        loadChildren: () =>
          import('./supervisor/supervisor.routes').then(m => m.SUPERVISOR_ROUTES),
      },

      { path: '', redirectTo: 'catalog', pathMatch: 'full' },
    ],
  },
];
