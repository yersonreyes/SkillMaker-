/**
 * certificate.service.ts — Certificate API (C5.1).
 *
 * Uses HttpPromiseBuilderService (same pattern as CourseCatalogService).
 * Base URL: /api/certificates.
 */
import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type {
  CertificateListItem,
  CertificateResponse,
  DownloadURLResponse,
  VerifyCertificateResponse,
} from './certificate.dto';

@Injectable({ providedIn: 'root' })
export class CertificateService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly base = `${environment.apiBaseUrl}/certificates`;

  /**
   * GET /api/certificates/me
   * Returns the authenticated user's certificates ordered by emitidoEn DESC.
   */
  getMyCertificates(): Promise<CertificateListItem[]> {
    return this.http
      .request<{ certificates?: CertificateListItem[] | null }>()
      .get()
      .url(`${this.base}/me`)
      .send()
      .then(res => res.certificates ?? []);
  }

  /**
   * GET /api/certificates/:id
   * Owner-scoped: returns 404 for non-owners (anti-enumeration).
   */
  getDetail(id: string): Promise<CertificateResponse> {
    return this.http
      .request<CertificateResponse>()
      .get()
      .url(`${this.base}/${id}`)
      .send();
  }

  /**
   * GET /api/certificates/:id/download
   * Owner-scoped: returns presigned URL valid for ~15 minutes.
   */
  getDownloadUrl(id: string): Promise<DownloadURLResponse> {
    return this.http
      .request<DownloadURLResponse>()
      .get()
      .url(`${this.base}/${id}/download`)
      .send();
  }

  /**
   * GET /api/certificates/verify/:codigo
   * PUBLIC (no auth). Confirms a certificate's authenticity by its code.
   * `.silent()` so a 404 (invalid code) is handled inline by the caller,
   * not surfaced as an error dialog.
   */
  verify(codigo: string): Promise<VerifyCertificateResponse> {
    return this.http
      .request<VerifyCertificateResponse>()
      .get()
      .url(`${this.base}/verify/${encodeURIComponent(codigo)}`)
      .silent()
      .send();
  }
}
