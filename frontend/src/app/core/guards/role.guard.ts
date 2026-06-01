import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { RoleCheckService } from '@core/services/common/role-check.service';
import type { UserRole } from '@core/services/authService/auth.res.dto';

export const roleGuard: CanActivateFn = route => {
  const roles = inject(RoleCheckService);
  const router = inject(Router);
  const required = (route.data?.['roles'] ?? []) as UserRole[];
  if (required.length === 0) return true;
  if (roles.hasAnyRole(required)) return true;
  router.navigate(['/platform']);
  return false;
};
