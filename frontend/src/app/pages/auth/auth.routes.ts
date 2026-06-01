import { Routes } from '@angular/router';
import { AuthLayoutComponent } from './layout/auth-layout.component';

export const AUTH_ROUTES: Routes = [
  {
    path: '',
    component: AuthLayoutComponent,
    children: [
      {
        path: 'login',
        loadComponent: () =>
          import('./login/login.component').then(m => m.LoginComponent),
      },
      {
        path: 'callback',
        loadComponent: () =>
          import('./callback/callback.component').then(m => m.CallbackComponent),
      },
      { path: '', redirectTo: 'login', pathMatch: 'full' },
    ],
  },
];
