import { Component, inject } from '@angular/core';
import { AuthService } from '@core/services/authService/auth.service';
import { CardModule } from 'primeng/card';
import { TagModule } from 'primeng/tag';

@Component({
  selector: 'app-profile',
  standalone: true,
  imports: [CardModule, TagModule],
  templateUrl: './profile.component.html',
  styleUrl: './profile.component.sass',
})
export class ProfileComponent {
  protected auth = inject(AuthService);
}
