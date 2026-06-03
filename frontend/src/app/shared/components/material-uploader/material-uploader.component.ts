/**
 * material-uploader.component.ts — C2.3 Material Attachments uploader.
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
import { UiDialogService } from '@core/services/ui-dialog.service';
import type { MaterialResponse } from '@core/services/materialService/material.types';

// ── Constants (mirror backend — REQ-VAL) ─────────────────────────────────────

/** 50 MiB — must match backend cfg.Storage.MaxUploadBytes. */
export const MAX_UPLOAD_BYTES = 52_428_800;

/**
 * MIME whitelist (service-level constant).
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
  private readonly ui = inject(UiDialogService);

  /** The course ID to attach materials to. */
  @Input() courseId = '';

  /** Emits the confirmed MaterialResponse when an upload completes successfully. */
  @Output() uploaded = new EventEmitter<MaterialResponse>();

  /** Upload progress (0–100). 0 = idle or done. */
  readonly progress = signal<number>(0);

  /** True while an upload is in progress. */
  readonly uploading = signal<boolean>(false);

  // Expose constants for template usage
  readonly maxBytes = MAX_UPLOAD_BYTES;
  readonly humanizeBytes = humanizeBytes;

  /**
   * Entry point called by the native file-input change event.
   * Validates the file client-side, then runs the full upload flow.
   */
  async handleFileSelected(file: File): Promise<void> {
    // ── Client-side fail-fast validation (REQ-FE-UPLOADER) ───────────────────
    if (file.size > MAX_UPLOAD_BYTES) {
      this.ui.showError(
        'Archivo demasiado grande',
        `El archivo supera el limite de ${humanizeBytes(MAX_UPLOAD_BYTES)}. Tamaño actual: ${humanizeBytes(file.size)}.`,
      );
      return;
    }

    if (!ALLOWED_MIME_TYPES.has(file.type)) {
      this.ui.showError(
        'Tipo de archivo no permitido',
        `El tipo "${file.type || 'desconocido'}" no está permitido. Use PDF, Word, ZIP o imágenes.`,
      );
      return;
    }

    // ── Upload flow ───────────────────────────────────────────────────────────
    this.uploading.set(true);
    this.progress.set(0);

    try {
      // Step 1: Presign (API call — JWT attached by interceptor).
      const presignResp = await this.materialService.presign(this.courseId, {
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
        // PUT to MinIO failed — show error, DO NOT call confirm.
        const msg = uploadErr instanceof Error ? uploadErr.message : 'Error al subir el archivo';
        this.ui.showError('Error al subir el archivo', msg);
        return;
      }

      // Step 3: Confirm (API call — JWT attached by interceptor).
      const material = await this.materialService.confirm(this.courseId, {
        key: presignResp.key,
        nombre: file.name,
        contentType: file.type,
        tamanoBytes: file.size,
      });

      // Step 4: Notify parent — material appears in the list.
      this.uploaded.emit(material);
    } catch (err) {
      // Presign or confirm network error — already handled by HttpPromiseBuilderService toast.
      void err;
    } finally {
      this.uploading.set(false);
      this.progress.set(0);
    }
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
