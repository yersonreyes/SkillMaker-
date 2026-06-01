import { HttpEvent, HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { Observable, catchError, from, switchMap, throwError } from 'rxjs';
import { AuthService } from '@core/services/authService/auth.service';

export const authRefreshInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);

  return next(req).pipe(
    catchError((err: HttpErrorResponse) => {
      if (err.status !== 401) return throwError(() => err);
      if (req.url.includes('/auth/refresh')) return throwError(() => err);
      if (!auth.getRefreshToken()) return throwError(() => err);

      return from(auth.refresh()).pipe(
        switchMap(res =>
          next(
            req.clone({
              setHeaders: { Authorization: `Bearer ${res.access_token}` },
            }),
          ),
        ),
        catchError(refreshErr => {
          auth.sessionExpired.set(true);
          return throwError(() => refreshErr);
        }),
      ) as Observable<HttpEvent<unknown>>;
    }),
  );
};
