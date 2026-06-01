import { Component, computed, inject } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { toSignal } from '@angular/core/rxjs-interop';

@Component({
  selector: 'app-pending-view',
  standalone: true,
  templateUrl: './pending-view.component.html',
  styleUrls: ['./pending-view.component.sass'],
})
export class PendingViewComponent {
  private route = inject(ActivatedRoute);
  private data = toSignal(this.route.data);
  title = computed(
    () => (this.data()?.['title'] as string | undefined) ?? 'Pendiente de implementacion',
  );
}
