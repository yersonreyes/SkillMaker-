import { Injectable, Signal, computed, inject } from '@angular/core';
import { AuthService } from '@core/services/authService/auth.service';
import type { UserRole } from '@core/services/authService/auth.res.dto';

@Injectable({ providedIn: 'root' })
export class RoleCheckService {
  private auth = inject(AuthService);

  get roles(): Signal<UserRole[]> {
    return this.auth.userRoles;
  }

  hasRole(role: UserRole): boolean {
    return this.auth.userRoles().includes(role);
  }

  hasAnyRole(roles: UserRole[]): boolean {
    if (!roles?.length) return true;
    const userRoles = this.auth.userRoles();
    return roles.some(r => userRoles.includes(r));
  }

  isAdmin = (): boolean => this.hasRole('administrador');
  isCreator = (): boolean => this.hasRole('creador');
  isSupervisor = (): boolean => this.hasRole('supervisor');

  hasRole$(role: UserRole): Signal<boolean> {
    return computed(() => this.auth.userRoles().includes(role));
  }
}
