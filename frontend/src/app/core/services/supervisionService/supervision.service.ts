import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type { SupervisionItem } from '../userService/user.res.dto';
import type { SupervisionCreateRequest } from '../userService/user.req.dto';

@Injectable({ providedIn: 'root' })
export class SupervisionService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly base = `${environment.apiBaseUrl}/supervisions`;

  /** GET /api/supervisions — list all supervision relations (admin only). */
  getAll(): Promise<SupervisionItem[]> {
    return this.http
      .request<SupervisionItem[]>()
      .get()
      .url(this.base)
      .send();
  }

  /** POST /api/supervisions — create a supervision relation (admin only). */
  create(body: SupervisionCreateRequest): Promise<SupervisionItem> {
    return this.http
      .request<SupervisionItem>()
      .post()
      .url(this.base)
      .body(body)
      .send();
  }

  /** DELETE /api/supervisions/:id — remove a supervision relation (admin only). */
  delete(id: string): Promise<void> {
    return this.http
      .request<void>()
      .delete()
      .url(`${this.base}/${id}`)
      .send();
  }
}
