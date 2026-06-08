import { Injectable, inject } from '@angular/core';
import { environment } from '@env/environment';
import { HttpPromiseBuilderService } from '../http-promise-builder.service';
import type { VideoItem } from './video.res.dto';
import type { VideoCreateRequest, VideoUpdateRequest } from './video.req.dto';

@Injectable({ providedIn: 'root' })
export class VideoService {
  private readonly http = inject(HttpPromiseBuilderService);
  private readonly sectionsBase = `${environment.apiBaseUrl}/sections`;
  private readonly videosBase = `${environment.apiBaseUrl}/videos`;

  /** POST /api/sections/:sectionId/videos — adds a video to a section. */
  create(sectionId: string, body: VideoCreateRequest): Promise<VideoItem> {
    return this.http
      .request<VideoItem>()
      .post()
      .url(`${this.sectionsBase}/${sectionId}/videos`)
      .body(body)
      .send();
  }

  /** PATCH /api/videos/:id — partial update; re-validates url/proveedor on change. */
  update(id: string, body: VideoUpdateRequest): Promise<VideoItem> {
    return this.http
      .request<VideoItem>()
      .patch()
      .url(`${this.videosBase}/${id}`)
      .body(body)
      .send();
  }

  /** DELETE /api/videos/:id — removes a video. */
  delete(id: string): Promise<void> {
    return this.http
      .request<void>()
      .delete()
      .url(`${this.videosBase}/${id}`)
      .send();
  }

  /** PATCH /api/sections/:sectionId/videos/reorder — reorders videos by full ids array. */
  reorder(sectionId: string, ids: string[]): Promise<void> {
    return this.http
      .request<void>()
      .patch()
      .url(`${this.sectionsBase}/${sectionId}/videos/reorder`)
      .body({ ids })
      .send();
  }
}
