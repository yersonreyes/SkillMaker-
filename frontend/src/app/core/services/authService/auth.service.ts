import { Injectable, signal, WritableSignal, inject } from '@angular/core';
import { Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '@env/environment';
import type { LoginResponse, RefreshTokenResponse, UserPublic, UserRole, JwtPayload } from './auth.res.dto';

declare const google: { accounts: { id: { disableAutoSelect: () => void } } };

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly baseUrl = `${environment.apiBaseUrl}/auth`;
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);

  // ─── Signals ─────────────────────────────────────────────
  public readonly user: WritableSignal<UserPublic | null> = signal(null);
  public readonly userRoles: WritableSignal<UserRole[]> = signal([]);
  public readonly sessionExpired: WritableSignal<boolean> = signal(false);

  // ─── Singleton anti-race ─────────────────────────────────
  private isRefreshing = false;
  private refreshPromise: Promise<RefreshTokenResponse> | null = null;
  private expirationTimer: ReturnType<typeof setTimeout> | null = null;

  constructor() {
    this.restoreSession();
  }

  // ─── Token storage ───────────────────────────────────────
  getToken(): string | null {
    return localStorage.getItem('auth_token');
  }

  getRefreshToken(): string | null {
    return localStorage.getItem('auth_refresh_token');
  }

  private setTokens(access: string, refresh: string | undefined): void {
    localStorage.setItem('auth_token', access);
    if (refresh) localStorage.setItem('auth_refresh_token', refresh);
  }

  private clearTokens(): void {
    localStorage.removeItem('auth_token');
    localStorage.removeItem('auth_refresh_token');
    localStorage.removeItem('auth_user');
  }

  // ─── JWT decoding (sin libs) ─────────────────────────────
  private decodeToken(token: string | null | undefined): JwtPayload | null {
    if (!token) return null;
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    try {
      const payload = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'));
      return JSON.parse(payload) as JwtPayload;
    } catch {
      return null;
    }
  }

  isTokenExpired(): boolean {
    const decoded = this.decodeToken(this.getToken());
    if (!decoded?.exp) return true;
    return Date.now() >= decoded.exp * 1000;
  }

  // ─── Login con Google ────────────────────────────────────
  async loginWithGoogle(idToken: string): Promise<LoginResponse> {
    const res = await firstValueFrom(
      this.http.post<LoginResponse>(`${this.baseUrl}/google`, { idToken }),
    );
    this.setTokens(res.access_token, res.refresh_token);
    this.applyLogin(res);
    return res;
  }

  private applyLogin(res: LoginResponse): void {
    localStorage.setItem('auth_user', JSON.stringify(res.user));
    this.user.set(res.user);
    this.userRoles.set(res.user.roles);
    this.sessionExpired.set(false);
    this.startExpirationTimer();
  }

  // ─── Refresh con singleton anti-race ─────────────────────
  async refresh(): Promise<RefreshTokenResponse> {
    if (this.isRefreshing && this.refreshPromise) return this.refreshPromise;
    this.isRefreshing = true;
    this.refreshPromise = this._doRefresh();
    try {
      return await this.refreshPromise;
    } finally {
      this.isRefreshing = false;
      this.refreshPromise = null;
    }
  }

  private async _doRefresh(): Promise<RefreshTokenResponse> {
    const refreshToken = this.getRefreshToken();
    if (!refreshToken) throw new Error('No refresh token');
    const res = await firstValueFrom(
      this.http.post<RefreshTokenResponse>(`${this.baseUrl}/refresh`, { refreshToken }),
    );
    this.setTokens(res.access_token, res.refresh_token);
    this.applyLogin(res);
    return res;
  }

  // Retry con backoff exponencial (3 intentos: 1s/2s/4s)
  async refreshWithRetry(maxAttempts = 3): Promise<RefreshTokenResponse> {
    let lastErr: unknown;
    for (let i = 0; i < maxAttempts; i++) {
      try {
        return await this.refresh();
      } catch (err) {
        lastErr = err;
        if (i < maxAttempts - 1) {
          await new Promise(r => setTimeout(r, 1000 * Math.pow(2, i)));
        }
      }
    }
    throw lastErr;
  }

  // ─── Timer 5min antes de expiración ──────────────────────
  private startExpirationTimer(): void {
    this.stopExpirationTimer();
    const decoded = this.decodeToken(this.getToken());
    if (!decoded?.exp) return;
    const expiresIn = decoded.exp * 1000 - Date.now();
    const refreshBefore = 5 * 60 * 1000;
    const timeout = Math.max(expiresIn - refreshBefore, 0);
    this.expirationTimer = setTimeout(async () => {
      try {
        await this.refreshWithRetry();
      } catch {
        this.sessionExpired.set(true);
      }
    }, timeout);
  }

  private stopExpirationTimer(): void {
    if (this.expirationTimer) {
      clearTimeout(this.expirationTimer);
      this.expirationTimer = null;
    }
  }

  // ─── Restore session ─────────────────────────────────────
  private restoreSession(): void {
    const token = this.getToken();
    const userJson = localStorage.getItem('auth_user');
    if (!token || !userJson) return;

    try {
      const user = JSON.parse(userJson) as UserPublic;
      this.user.set(user);
      this.userRoles.set(user.roles);
    } catch {
      this.clearTokens();
      return;
    }

    if (!this.isTokenExpired()) {
      this.startExpirationTimer();
    } else if (this.getRefreshToken()) {
      this.refreshWithRetry().catch(() => this.sessionExpired.set(true));
    } else {
      this.sessionExpired.set(true);
    }
  }

  // ─── Logout ──────────────────────────────────────────────
  async logout(): Promise<void> {
    const refreshToken = this.getRefreshToken();
    try {
      await firstValueFrom(
        this.http.post(`${this.baseUrl}/logout`, { refreshToken: refreshToken ?? '' }),
      );
    } catch {
      // silencioso — el logout local siempre procede
    }
    try {
      google?.accounts?.id?.disableAutoSelect?.();
    } catch {
      // noop
    }
    this.clearTokens();
    this.user.set(null);
    this.userRoles.set([]);
    this.sessionExpired.set(false);
    this.stopExpirationTimer();
    await this.router.navigate(['/auth/login']);
  }
}
