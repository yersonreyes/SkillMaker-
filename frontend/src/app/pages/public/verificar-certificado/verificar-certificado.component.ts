/**
 * verificar-certificado.component.ts — PUBLIC certificate verification page.
 *
 * Root-level route (NO auth guard): anyone with a certificate code can confirm
 * its authenticity. Calls GET /api/certificates/verify/:codigo (public endpoint).
 * Supports an optional :codigo deep-link (e.g. a QR printed on the certificate).
 */
import { Component, inject, signal, OnInit } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { InputTextModule } from 'primeng/inputtext';

import { CertificateService } from '@core/services/certificateService/certificate.service';
import type { VerifyCertificateResponse } from '@core/services/certificateService/certificate.dto';

/** UI state machine for the verification flow. */
type VerifyState = 'idle' | 'loading' | 'found' | 'notfound' | 'error';

@Component({
  selector: 'app-verificar-certificado',
  standalone: true,
  imports: [DatePipe, FormsModule, InputTextModule],
  templateUrl: './verificar-certificado.component.html',
  styleUrl: './verificar-certificado.component.sass',
})
export class VerificarCertificadoComponent implements OnInit {
  private readonly certService = inject(CertificateService);
  private readonly route = inject(ActivatedRoute);

  readonly codigo = signal<string>('');
  readonly state = signal<VerifyState>('idle');
  readonly result = signal<VerifyCertificateResponse | null>(null);

  ngOnInit(): void {
    const param = this.route.snapshot.paramMap.get('codigo');
    if (param) {
      this.codigo.set(param);
      void this.verify();
    }
  }

  async verify(): Promise<void> {
    const code = this.codigo().trim();
    if (!code) return;

    this.state.set('loading');
    this.result.set(null);
    try {
      const res = await this.certService.verify(code);
      this.result.set(res);
      this.state.set('found');
    } catch (err) {
      // 404 = code does not exist (expected, not an error). Anything else = transport error.
      if (err instanceof HttpErrorResponse && err.status === 404) {
        this.state.set('notfound');
      } else {
        this.state.set('error');
      }
    }
  }
}
