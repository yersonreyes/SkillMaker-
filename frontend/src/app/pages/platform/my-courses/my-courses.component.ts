import { Component } from '@angular/core';
import { CardModule } from 'primeng/card';

@Component({
  selector: 'app-my-courses',
  standalone: true,
  imports: [CardModule],
  templateUrl: './my-courses.component.html',
  styleUrl: './my-courses.component.sass',
})
export class MyCoursesComponent {}
