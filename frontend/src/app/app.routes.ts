import { Routes } from '@angular/router';
import { authGuard } from '@core/guards/auth.guard';
import { guestGuard } from '@core/guards/guest.guard';

export const routes: Routes = [
  {
    path: 'auth',
    canActivate: [guestGuard],
    loadChildren: () =>
      import('@pages/auth/auth.routes').then(m => m.AUTH_ROUTES),
  },
  {
    path: 'platform',
    canActivate: [authGuard],
    loadChildren: () =>
      import('@pages/platform/platform.routes').then(m => m.PLATFORM_ROUTES),
  },
  { path: '', redirectTo: '/auth', pathMatch: 'full' },
  { path: '**', redirectTo: '/auth' },
];
