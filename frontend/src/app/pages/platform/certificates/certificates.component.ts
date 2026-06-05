/**
 * certificates.component.ts — Certificados page (C5.1).
 *
 * Replaces the PendingViewComponent stub for the /platform/certificates route.
 * Shows a grid of certificate cards with courseTitulo, emitidoEn, codigo, and
 * a "Descargar PDF" button that fetches a presigned URL and opens it.
 */
import {
  Component,
  inject,
  signal,
  OnInit,
} from '@angular/core';
import { DatePipe } from '@angular/common';
import { SkeletonModule } from 'primeng/skeleton';

import { CertificateService } from '@core/services/certificateService/certificate.service';
import type { CertificateListItem } from '@core/services/certificateService/certificate.dto';

@Component({
  selector: 'app-certificates',
  standalone: true,
  imports: [DatePipe, SkeletonModule],
  templateUrl: './certificates.component.html',
  styleUrl: './certificates.component.sass',
})
export class CertificatesComponent implements OnInit {
  private readonly certService = inject(CertificateService);

  // ── State ──────────────────────────────────────────────────────────────────
  readonly certificates = signal<CertificateListItem[]>([]);
  readonly loading = signal<boolean>(false);
  readonly downloadingId = signal<string | null>(null);

  ngOnInit(): void {
    void this.loadCertificates();
  }

  async loadCertificates(): Promise<void> {
    this.loading.set(true);
    try {
      const items = await this.certService.getMyCertificates();
      this.certificates.set(items);
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    } finally {
      this.loading.set(false);
    }
  }

  /** Fetch presigned URL and open PDF in new tab. */
  async downloadCertificate(certId: string): Promise<void> {
    this.downloadingId.set(certId);
    try {
      const res = await this.certService.getDownloadUrl(certId);
      if (res.url) {
        window.open(res.url, '_blank');
      }
    } catch {
      // Error toast shown by HttpPromiseBuilderService
    } finally {
      this.downloadingId.set(null);
    }
  }
}
