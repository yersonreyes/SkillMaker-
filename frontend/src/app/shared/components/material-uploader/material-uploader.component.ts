/**
 * material-uploader.component.ts — Material Attachments uploader (parameterized).
 *
 * Updated in course-structure-v2: parameterized with @Input() target ('material' | 'thumbnail')
 * and @Input() ownerId (videoId for material, courseId for thumbnail).
 * The 'thumbnail' target calls CourseService.presignThumbnail/confirmThumbnail
 * and restricts MIME to image/*.
 *
 * Client-side fail-fast validation (size + MIME) runs BEFORE any network call.
 * Upload flow: validate → presign (API+JWT) → uploadToStorage (raw XHR, no JWT)
 *              → confirm (API+JWT) → emit `uploaded`.
 *
 * Exported `humanizeBytes` is a pure formatting function used in the template
 * and tested independently.
 */
import {
  Component,
  Input,
  Output,
  EventEmitter,
  signal,
  inject,
} from '@angular/core';
import { CommonModule } from '@angular/common';

import { MaterialService } from '@core/services/materialService/material.service';
import { CourseService } from '@core/services/courseService/course.service';
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { MaterialResponse } from '@core/services/materialService/material.types';

// ── Constants (mirror backend — REQ-VAL) ─────────────────────────────────────

/** 50 MiB — must match backend cfg.Storage.MaxUploadBytes. */
export const MAX_UPLOAD_BYTES = 52_428_800;

/**
 * MIME whitelist for material uploads (service-level constant).
 * Must match the backend allowedMIME map in service/material.go (ADR-3).
 */
export const ALLOWED_MIME_TYPES = new Set([
  'application/pdf',
  'application/msword',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  'application/zip',
  'application/x-zip-compressed',
  'image/jpeg',
  'image/png',
  'image/gif',
  'image/webp',
]);

/**
 * MIME whitelist for thumbnail uploads (image/* subset only).
 */
export const THUMBNAIL_MIME_TYPES = new Set([
  'image/jpeg',
  'image/png',
  'image/gif',
  'image/webp',
]);

// ── Pure helper ───────────────────────────────────────────────────────────────

/**
 * Formats a byte count into a human-readable string (e.g. "5.0 MB").
 * Exported for independent testing and template reuse.
 */
export function humanizeBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

// ── Component ─────────────────────────────────────────────────────────────────

@Component({
  selector: 'app-material-uploader',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './material-uploader.component.html',
  styleUrl: './material-uploader.component.sass',
})
export class MaterialUploaderComponent {
  private readonly materialService = inject(MaterialService);
  private readonly courseService = inject(CourseService);
  private readonly ui = inject(UiDialogService);

  /**
   * Upload target mode:
   *  'material'  → uploads a per-video material; ownerId = videoId
   *  'thumbnail' → uploads a course thumbnail image; ownerId = courseId
   */
  @Input() target: 'material' | 'thumbnail' = 'material';

  /**
   * The owner ID for the upload:
   *  - For target='material': the videoId
   *  - For target='thumbnail': the courseId
   *
   * @deprecated Use ownerId instead of courseId for material target.
   */
  @Input() ownerId = '';

  /**
   * @deprecated Legacy input — kept for backward compat (previously courseId for material).
   * Prefer ownerId. If ownerId is empty and courseId is set, courseId is used as fallback.
   */
  @Input() set courseId(value: string) {
    if (!this.ownerId) {
      this.ownerId = value;
    }
  }

  /** Emits the confirmed MaterialResponse when a material upload completes successfully. */
  @Output() uploaded = new EventEmitter<MaterialResponse>();

  /** Emits the confirmed key when a thumbnail upload completes successfully. */
  @Output() thumbnailUploaded = new EventEmitter<string>();

  /** Upload progress (0–100). 0 = idle or done. */
  readonly progress = signal<number>(0);

  /** True while an upload is in progress. */
  readonly uploading = signal<boolean>(false);

  // Expose constants for template usage
  readonly maxBytes = MAX_UPLOAD_BYTES;
  readonly humanizeBytes = humanizeBytes;

  /** Allowed accept string for the file input, based on target mode. */
  get acceptString(): string {
    if (this.target === 'thumbnail') {
      return 'image/jpeg,image/png,image/gif,image/webp';
    }
    return 'application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/zip,application/x-zip-compressed,image/jpeg,image/png,image/gif,image/webp';
  }

  /** Human-readable hint for the upload zone, based on target mode. */
  get hintText(): string {
    if (this.target === 'thumbnail') {
      return 'Seleccionar miniatura — JPG, PNG, GIF, WebP · max 50 MB';
    }
    return 'Seleccionar archivo — PDF, Word, ZIP o imagen · max 50 MB';
  }

  /**
   * Entry point called by the native file-input change event.
   * Validates the file client-side, then runs the full upload flow.
   */
  async handleFileSelected(file: File): Promise<void> {
    // ── Client-side fail-fast validation ────────────────────────────────────
    if (file.size > MAX_UPLOAD_BYTES) {
      this.ui.showError(
        'Archivo demasiado grande',
        `El archivo supera el limite de ${humanizeBytes(MAX_UPLOAD_BYTES)}. Tamaño actual: ${humanizeBytes(file.size)}.`,
      );
      return;
    }

    const allowedSet = this.target === 'thumbnail' ? THUMBNAIL_MIME_TYPES : ALLOWED_MIME_TYPES;
    if (!allowedSet.has(file.type)) {
      const hint = this.target === 'thumbnail'
        ? 'Use JPG, PNG, GIF o WebP para la miniatura.'
        : `El tipo "${file.type || 'desconocido'}" no está permitido. Use PDF, Word, ZIP o imágenes.`;
      this.ui.showError('Tipo de archivo no permitido', hint);
      return;
    }

    // ── Upload flow ──────────────────────────────────────────────────────────
    this.uploading.set(true);
    this.progress.set(0);

    try {
      if (this.target === 'thumbnail') {
        await this.handleThumbnailUpload(file);
      } else {
        await this.handleMaterialUpload(file);
      }
    } catch (err) {
      // Presign or confirm network error — already handled by HttpPromiseBuilderService toast.
      void err;
    } finally {
      this.uploading.set(false);
      this.progress.set(0);
    }
  }

  private async handleMaterialUpload(file: File): Promise<void> {
    // Step 1: Presign (API call — JWT attached by interceptor).
    const presignResp = await this.materialService.presign(this.ownerId, {
      nombre: file.name,
      contentType: file.type,
      tamanoBytes: file.size,
    });

    // Step 2: Upload directly to MinIO via raw XHR (NO JWT — see materialService).
    try {
      await this.materialService.uploadToStorage(
        presignResp.uploadUrl,
        file,
        (pct: number) => this.progress.set(pct),
      );
    } catch (uploadErr) {
      const msg = uploadErr instanceof Error ? uploadErr.message : 'Error al subir el archivo';
      this.ui.showError('Error al subir el archivo', msg);
      return;
    }

    // Step 3: Confirm (API call — JWT attached by interceptor).
    const material = await this.materialService.confirm(this.ownerId, {
      key: presignResp.key,
      nombre: file.name,
      contentType: file.type,
      tamanoBytes: file.size,
    });

    // Step 4: Notify parent — material appears in the list.
    this.uploaded.emit(material);
  }

  private async handleThumbnailUpload(file: File): Promise<void> {
    // Step 1: Presign thumbnail (API call — JWT attached by interceptor).
    const presignResp = await this.courseService.presignThumbnail(this.ownerId, {
      nombre: file.name,
      contentType: file.type,
      tamanoBytes: file.size,
    });

    // Step 2: Upload directly to MinIO via raw XHR (NO JWT — see materialService).
    try {
      await this.materialService.uploadToStorage(
        presignResp.uploadUrl,
        file,
        (pct: number) => this.progress.set(pct),
      );
    } catch (uploadErr) {
      const msg = uploadErr instanceof Error ? uploadErr.message : 'Error al subir la miniatura';
      this.ui.showError('Error al subir la miniatura', msg);
      return;
    }

    // Step 3: Confirm thumbnail (API call — JWT attached by interceptor).
    await this.courseService.confirmThumbnail(this.ownerId, { key: presignResp.key });

    // Step 4: Notify parent with the key.
    this.thumbnailUploaded.emit(presignResp.key);
  }

  /** Called by the native file input (change event). */
  onInputChange(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    // Reset the input so the same file can be re-selected after an error.
    input.value = '';
    void this.handleFileSelected(file);
  }
}
