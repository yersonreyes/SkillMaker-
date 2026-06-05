/**
 * course-card.component.ts — Reusable catalog course card (C2.4).
 *
 * Standalone component. Displays one CatalogCourseCard with titulo, descripcion,
 * creadorNombre, and a "Ver detalle" CTA. Emits @Output() open when clicked.
 * Uses Cyanotype Workshop globals (.panel, .btn--cyan).
 */
import { Component, Input, Output, EventEmitter } from '@angular/core';
import type { CatalogCourseCard } from '@core/services/courseCatalogService/course-catalog.dto';

@Component({
  selector: 'app-course-card',
  standalone: true,
  imports: [],
  templateUrl: './course-card.component.html',
  styleUrl: './course-card.component.sass',
})
export class CourseCardComponent {
  @Input({ required: true }) card!: CatalogCourseCard;
  @Output() readonly open = new EventEmitter<void>();

  onOpen(): void {
    this.open.emit();
  }
}
