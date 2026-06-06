/**
 * course-card.component.ts — Reusable catalog course card (C2.4).
 *
 * Updated in course-structure-v2:
 * - Renders miniatura img (or placeholder when miniaturaUrl is null)
 * - Shows nivel tag
 * - Shows categorias chips
 * - Shows cantidadClases, horasVideo (1 decimal), horasPractico
 */
import { Component, Input, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';
import type { CatalogCourseCard } from '@core/services/courseCatalogService/course-catalog.dto';

@Component({
  selector: 'app-course-card',
  standalone: true,
  imports: [CommonModule],
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
