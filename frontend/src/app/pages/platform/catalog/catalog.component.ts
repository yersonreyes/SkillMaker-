import { Component } from '@angular/core';
import { CardModule } from 'primeng/card';

@Component({
  selector: 'app-catalog',
  standalone: true,
  imports: [CardModule],
  templateUrl: './catalog.component.html',
  styleUrl: './catalog.component.sass',
})
export class CatalogComponent {}
