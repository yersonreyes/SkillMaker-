import { Component, OnInit, inject } from '@angular/core';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { AuthService } from '@core/services/authService/auth.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import { environment } from '../../../../environments/environment';

declare const google: any; // eslint-disable-line @typescript-eslint/no-explicit-any

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [ButtonModule],
  templateUrl: './login.component.html',
  styleUrl: './login.component.sass',
})
export class LoginComponent implements OnInit {
  private auth = inject(AuthService);
  private router = inject(Router);
  private dialog = inject(UiDialogService);
  protected loading = false;

  ngOnInit(): void {
    if (typeof google === 'undefined' || !google.accounts?.id) {
      console.warn('Google Identity Services no esta cargado todavia');
      return;
    }
    google.accounts.id.initialize({
      client_id: environment.googleClientId,
      hd: environment.googleHostedDomain,
      callback: (response: { credential: string }) =>
        this.handleCredential(response.credential),
      auto_select: false,
      cancel_on_tap_outside: true,
    });
  }

  signInWithGoogle(): void {
    if (typeof google === 'undefined') {
      this.dialog.showError('Google no disponible', 'Recarga la pagina e intenta de nuevo');
      return;
    }
    google.accounts.id.prompt();
  }

  private async handleCredential(idToken: string): Promise<void> {
    this.loading = true;
    try {
      await this.auth.loginWithGoogle(idToken);
      await this.router.navigate(['/platform/catalog']);
    } catch (err: unknown) {
      const e = err as { error?: { code?: string; message?: string } };
      const code = e?.error?.code;
      const msg =
        code === 'UNAUTHORIZED_DOMAIN'
          ? 'Tu cuenta no pertenece al dominio corporativo'
          : code === 'INVALID_GOOGLE_TOKEN'
            ? 'El token de Google no es valido. Reintenta.'
            : 'No se pudo iniciar sesion';
      this.dialog.showError('Error de autenticacion', msg);
    } finally {
      this.loading = false;
    }
  }
}
