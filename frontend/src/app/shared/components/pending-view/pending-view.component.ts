import { Component, Input } from '@angular/core';

@Component({
  selector: 'app-pending-view',
  standalone: true,
  templateUrl: './pending-view.component.html',
  styleUrls: ['./pending-view.component.sass'],
})
export class PendingViewComponent {
  @Input() title?: string;
}
