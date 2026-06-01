import { Component, inject } from '@angular/core';
import { AuthService } from '@core/services/authService/auth.service';

@Component({
  selector: 'app-profile',
  standalone: true,
  imports: [],
  templateUrl: './profile.component.html',
  styleUrl: './profile.component.sass',
})
export class ProfileComponent {
  protected auth = inject(AuthService);

  protected shortId(id: string | undefined | null): string {
    if (!id) return '----';
    return id.slice(0, 4).toUpperCase();
  }
}
