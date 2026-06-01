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

      // Domain stubs (load PendingViewComponent)
      { path: 'courses/:id',      component: PendingViewComponent, data: { title: 'Detalle del curso' } },
      { path: 'evaluations/:id',  component: PendingViewComponent, data: { title: 'Evaluacion' } },
      { path: 'certificates',     component: PendingViewComponent, data: { title: 'Certificados' } },
      { path: 'badges',           component: PendingViewComponent, data: { title: 'Insignias' } },

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
