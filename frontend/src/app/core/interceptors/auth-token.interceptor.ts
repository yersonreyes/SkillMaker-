import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { AuthService } from '@core/services/authService/auth.service';

const SKIP_PATTERNS = ['/auth/google', '/auth/refresh', '/auth/logout'];

export const authTokenInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  const token = auth.getToken();

  if (!token || SKIP_PATTERNS.some(p => req.url.includes(p))) {
    return next(req);
  }

  return next(req.clone({ setHeaders: { Authorization: `Bearer ${token}` } }));
};
