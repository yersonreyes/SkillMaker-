import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type { Page, UserListItem, UserDetail } from './user.res.dto';
import type { UserListParams, RolesPatchRequest } from './user.req.dto';

@Injectable({ providedIn: 'root' })
export class UserService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly base = `${environment.apiBaseUrl}/users`;

  /** GET /api/users — paginated + filtered list (admin only). */
  getAll(params: UserListParams): Promise<Page<UserListItem>> {
    return this.http
      .request<Page<UserListItem>>()
      .get()
      .url(this.base)
      .queryParam('page', params.page)
      .queryParam('size', params.size)
      .queryParam('q', params.q)
      .queryParam('role', params.role)
      .queryParam('active', params.active)
      .send();
  }

  /** GET /api/users/:id — user detail (admin only). */
  getById(id: string): Promise<UserDetail> {
    return this.http
      .request<UserDetail>()
      .get()
      .url(`${this.base}/${id}`)
      .send();
  }

  /** GET /api/users/me — current user's own detail (any authenticated user). */
  getMe(): Promise<UserDetail> {
    return this.http
      .request<UserDetail>()
      .get()
      .url(`${this.base}/me`)
      .send();
  }

  /** PATCH /api/users/:id/roles — add/remove roles delta (admin only). */
  updateRoles(id: string, body: RolesPatchRequest): Promise<UserDetail> {
    return this.http
      .request<UserDetail>()
      .patch()
      .url(`${this.base}/${id}/roles`)
      .body(body)
      .send();
  }

  /** PATCH /api/users/:id/active — soft-delete toggle (admin only). */
  setActive(id: string, active: boolean): Promise<UserDetail> {
    return this.http
      .request<UserDetail>()
      .patch()
      .url(`${this.base}/${id}/active`)
      .body({ active })
      .send();
  }
}
