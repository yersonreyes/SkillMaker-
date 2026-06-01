import {
  Directive,
  Input,
  TemplateRef,
  ViewContainerRef,
  DestroyRef,
  effect,
  inject,
} from '@angular/core';
import { RoleCheckService } from '@core/services/common/role-check.service';
import type { UserRole } from '@core/services/authService/auth.res.dto';

@Directive({ selector: '[hasRole]', standalone: true })
export class HasRoleDirective {
  private roles = inject(RoleCheckService);
  private templateRef = inject(TemplateRef<unknown>);
  private viewContainer = inject(ViewContainerRef);
  private destroyRef = inject(DestroyRef);
  private allowed: UserRole[] = [];
  private rendered = false;

  @Input()
  set hasRole(value: UserRole | UserRole[]) {
    this.allowed = Array.isArray(value) ? value : [value];
  }

  constructor() {
    const e = effect(() => {
      this.roles.roles(); // dependencia reactiva — re-evalua cuando cambian los roles
      this.updateView();
    });
    this.destroyRef.onDestroy(() => e.destroy());
  }

  private updateView(): void {
    const hasAccess = this.roles.hasAnyRole(this.allowed);
    if (hasAccess && !this.rendered) {
      this.viewContainer.createEmbeddedView(this.templateRef);
      this.rendered = true;
    } else if (!hasAccess && this.rendered) {
      this.viewContainer.clear();
      this.rendered = false;
    }
  }
}
