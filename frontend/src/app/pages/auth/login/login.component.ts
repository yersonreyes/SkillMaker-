import { Component, inject } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '@core/services/authService/auth.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import { environment } from '@env/environment';

declare const google: any; // eslint-disable-line @typescript-eslint/no-explicit-any

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [RouterLink],
  templateUrl: './login.component.html',
  styleUrl: './login.component.sass',
})
export class LoginComponent {
  private auth = inject(AuthService);
  private router = inject(Router);
  private dialog = inject(UiDialogService);
  protected loading = false;
  private initialized = false;

  /**
   * Inicializa Google Identity Services. El script de GIS se carga con
   * `async defer` en index.html, por lo que puede no estar listo cuando
   * Angular monta el componente. Por eso se espera hasta que `window.google`
   * exista antes de inicializar — con un timeout corto.
   */
  private async ensureInitialized(): Promise<boolean> {
    if (this.initialized) return true;

    // Espera a que GIS este disponible (script async defer aun cargando)
    const start = Date.now();
    while (typeof google === 'undefined' || !google.accounts?.id) {
      if (Date.now() - start > 5000) {
        this.dialog.showError(
          'Google no disponible',
          'No se pudo cargar Google Identity Services. Verifica tu conexion y recarga la pagina.',
        );
        return false;
      }
      await new Promise((r) => setTimeout(r, 100));
    }

    const initConfig: Record<string, unknown> = {
      client_id: environment.googleClientId,
      callback: (response: { credential: string }) =>
        this.handleCredential(response.credential),
      auto_select: false,
      cancel_on_tap_outside: true,
    };
    // El filtro de dominio solo aplica si esta configurado (prod con Workspace).
    // En dev con Gmail personal queda vacio y se omite.
    if (environment.googleHostedDomain) {
      initConfig['hd'] = environment.googleHostedDomain;
    }
    google.accounts.id.initialize(initConfig);
    this.initialized = true;
    return true;
  }

  async signInWithGoogle(): Promise<void> {
    const ok = await this.ensureInitialized();
    if (!ok) return;
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
