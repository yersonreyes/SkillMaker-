import { Component } from '@angular/core';

@Component({
  selector: 'app-callback',
  standalone: true,
  template: `
    <div class="text-center space-y-4 p-8">
      <i class="pi pi-spin pi-spinner text-4xl text-primary-500"></i>
      <p class="text-gray-600">Procesando autenticacion...</p>
    </div>
  `,
  styles: [],
})
export class CallbackComponent {}
