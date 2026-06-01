import { Routes } from '@angular/router';
import { PendingViewComponent } from '@shared/components/pending-view/pending-view.component';

export const CREATOR_ROUTES: Routes = [
  {
    path: 'mi-contenido',
    component: PendingViewComponent,
    data: { title: 'Mi contenido' },
  },
  {
    path: 'crear-curso',
    component: PendingViewComponent,
    data: { title: 'Crear curso' },
  },
  { path: '', redirectTo: 'mi-contenido', pathMatch: 'full' },
];
