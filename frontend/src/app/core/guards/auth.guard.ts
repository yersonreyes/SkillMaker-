import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '@core/services/authService/auth.service';

export const authGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const token = auth.getToken();

  if (token && !auth.isTokenExpired()) return true;
  if (token && auth.isTokenExpired() && auth.getRefreshToken()) return true;

  router.navigate(['/auth/login']);
  return false;
};
