import { Routes } from '@angular/router';
import { Component } from '@angular/core';

// Placeholder component used for /auth and /platform until Phase 7-8 implement
// the real route modules (auth.routes.ts and platform.routes.ts).
// Replace loadComponent with loadChildren once those modules exist.
@Component({
  standalone: true,
  template: '<div class="p-8 text-center"><h2>Bootstrap shell</h2><p>Pages are coming in subsequent commits.</p></div>',
})
class PlaceholderComponent {}

export const routes: Routes = [
  {
    path: 'auth',
    loadComponent: () => Promise.resolve(PlaceholderComponent),
  },
  {
    path: 'platform',
    loadComponent: () => Promise.resolve(PlaceholderComponent),
  },
  { path: '', redirectTo: '/auth', pathMatch: 'full' },
  { path: '**', redirectTo: '/auth' },
];
