import { Routes } from '@angular/router';

export const CREATOR_ROUTES: Routes = [
  {
    path: 'mi-contenido',
    loadComponent: () =>
      import('./mi-contenido/mi-contenido.component').then(m => m.MiContenidoComponent),
    data: { title: 'Mi contenido' },
  },
  {
    path: 'curso-editar/:id',
    loadComponent: () =>
      import('./curso-editar/curso-editar.component').then(m => m.CursoEditarComponent),
    data: { title: 'Editar curso' },
  },
  { path: '', redirectTo: 'mi-contenido', pathMatch: 'full' },
];
