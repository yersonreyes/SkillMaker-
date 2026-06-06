/**
 * material.service.ts — Material Attachments service.
 *
 * Re-keyed in course-structure-v2: materials now belong to videos (not courses).
 * All API calls (presign/confirm/list/downloadUrl/remove) go through
 * HttpPromiseBuilderService, which attaches the JWT via Angular's HTTP interceptor.
 *
 * CRITICAL (D7 / REQ-FE-UPLOADER): `uploadToStorage` uses raw XMLHttpRequest, NOT
 * HttpClient. The Angular JWT interceptor would inject "Authorization: Bearer <token>"
 * into the presigned PUT, breaking MinIO's signature verification. Raw XHR bypasses
 * all Angular HTTP interceptors entirely — only Content-Type is set.
 */
import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type {
  MaterialPresignRequest,
  PresignResponse,
  MaterialConfirmRequest,
  MaterialResponse,
  DownloadResponse,
} from './material.types';

@Injectable({ providedIn: 'root' })
export class MaterialService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly videosBase = `${environment.apiBaseUrl}/videos`;
  private readonly materialsBase = `${environment.apiBaseUrl}/materials`;

  /**
   * POST /api/videos/:videoId/materials/presign
   * Requests a presigned PUT URL from the backend (JWT required — same-origin API).
   */
  presign(videoId: string, body: MaterialPresignRequest): Promise<PresignResponse> {
    return this.http
      .request<PresignResponse>()
      .post()
      .url(`${this.videosBase}/${videoId}/materials/presign`)
      .body(body)
      .send();
  }

  /**
   * POST /api/videos/:videoId/materials
   * Confirms the upload after the XHR PUT to MinIO succeeds (JWT required).
   */
  confirm(videoId: string, body: MaterialConfirmRequest): Promise<MaterialResponse> {
    return this.http
      .request<MaterialResponse>()
      .post()
      .url(`${this.videosBase}/${videoId}/materials`)
      .body(body)
      .send();
  }

  /**
   * GET /api/videos/:videoId/materials
   * Lists all materials for a video (JWT required — owner only, creator-editor endpoint).
   */
  list(videoId: string): Promise<MaterialResponse[]> {
    return this.http
      .request<MaterialResponse[]>()
      .get()
      .url(`${this.videosBase}/${videoId}/materials`)
      .send();
  }

  /**
   * GET /api/materials/:materialId/download
   * Returns a presigned GET URL for downloading the material.
   * Works for owner OR enrolled student (OQ3 resolution).
   */
  downloadUrl(materialId: string): Promise<DownloadResponse> {
    return this.http
      .request<DownloadResponse>()
      .get()
      .url(`${this.materialsBase}/${materialId}/download`)
      .send();
  }

  /**
   * DELETE /api/materials/:materialId
   * Deletes a material row and best-effort removes the MinIO object.
   */
  remove(materialId: string): Promise<void> {
    return this.http
      .request<void>()
      .delete()
      .url(`${this.materialsBase}/${materialId}`)
      .send();
  }

  /**
   * Uploads `file` directly to MinIO via a presigned PUT URL using raw XMLHttpRequest.
   *
   * LOAD-BEARING — DO NOT REPLACE WITH HttpClient OR fetch():
   *   - HttpClient: Angular's JWT interceptor would add "Authorization: Bearer <token>",
   *     which breaks the presigned URL signature check in MinIO (403 SignatureDoesNotMatch).
   *   - fetch(): no upload progress events (no equivalent of xhr.upload.onprogress).
   *   Raw XHR bypasses all Angular interceptors and provides progress tracking.
   *
   * Headers set: Content-Type = file.type ONLY. No Authorization. No custom headers.
   *
   * @param uploadUrl  Presigned PUT URL from the backend presign response.
   * @param file       The browser File object to upload.
   * @param onProgress Callback receiving upload percentage (0–100).
   */
  uploadToStorage(
    uploadUrl: string,
    file: File,
    onProgress: (percent: number) => void,
  ): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      const xhr = new XMLHttpRequest();

      // Open the request to the presigned MinIO URL — this goes directly to MinIO,
      // bypassing Angular's HTTP pipeline entirely (no interceptors, no JWT).
      xhr.open('PUT', uploadUrl);

      // Set ONLY Content-Type. Authorization MUST NOT be set here —
      // the presigned URL is self-authenticating; adding a Bearer token
      // breaks the HMAC signature embedded in the URL query parameters.
      xhr.setRequestHeader('Content-Type', file.type);

      // Track upload progress for the progress bar.
      xhr.upload.onprogress = (e: ProgressEvent): void => {
        if (e.lengthComputable) {
          onProgress(Math.round((e.loaded / e.total) * 100));
        }
      };

      // Resolve or reject based on response status.
      xhr.onload = (): void => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve();
        } else {
          reject(new Error(`upload failed ${xhr.status}`));
        }
      };

      // Network error (no response received).
      xhr.onerror = (): void => {
        reject(new Error('network error'));
      };

      xhr.send(file);
    });
  }
}
