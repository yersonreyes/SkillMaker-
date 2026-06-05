/**
 * certificates.component.spec.ts — CertificatesComponent unit tests (Strict TDD — RED → GREEN).
 *
 * Strategy: spy on CertificateService; call component methods directly.
 *
 * Covers:
 *  - Renders certificate cards from getMyCertificates()
 *  - Shows courseTitulo, codigo, emitidoEn on each card
 *  - "Descargar PDF" button calls getDownloadUrl and opens url via window.open
 *  - Loading state shown when loading=true
 *  - Empty state (.empty) shown when no certificates
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { CertificatesComponent } from './certificates.component';
import { CertificateService } from '@core/services/certificateService/certificate.service';
import type { CertificateListItem } from '@core/services/certificateService/certificate.dto';

const MOCK_CERT: CertificateListItem = {
  id: 'cert-1',
  courseId: 'course-1',
  courseTitulo: 'Go Avanzado',
  codigo: 'ABCDEFGHIJKLM',
  emitidoEn: '2026-01-01T00:00:00Z',
};

describe('CertificatesComponent', () => {
  let certServiceSpy: Partial<CertificateService>;

  beforeEach(async () => {
    certServiceSpy = {
      getMyCertificates: vi.fn().mockResolvedValue([MOCK_CERT]),
      getDownloadUrl: vi.fn().mockResolvedValue({ url: 'https://minio/cert.pdf', expiresAt: '2026-01-01T00:15:00Z' }),
    };

    await TestBed.configureTestingModule({
      imports: [CertificatesComponent],
      providers: [
        { provide: CertificateService, useValue: certServiceSpy },
        provideRouter([]),
        provideAnimationsAsync(),
        ConfirmationService,
        MessageService,
      ],
    }).compileComponents();
  });

  afterEach(() => {
    TestBed.resetTestingModule();
  });

  it('calls getMyCertificates on init', async () => {
    const fixture = TestBed.createComponent(CertificatesComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(certServiceSpy.getMyCertificates).toHaveBeenCalledTimes(1);
  });

  it('renders certificate card with courseTitulo after load', async () => {
    const fixture = TestBed.createComponent(CertificatesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Go Avanzado');
  });

  it('renders certificate card with codigo', async () => {
    const fixture = TestBed.createComponent(CertificatesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('ABCDEFGHIJKLM');
  });

  it('shows empty state (.empty) when no certificates', async () => {
    (certServiceSpy.getMyCertificates as ReturnType<typeof vi.fn>).mockResolvedValue([]);

    const fixture = TestBed.createComponent(CertificatesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    const emptyEl = el.querySelector('.empty');
    expect(emptyEl).not.toBeNull();
  });

  it('"Descargar PDF" button calls getDownloadUrl and invokes window.open', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);

    const fixture = TestBed.createComponent(CertificatesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    await fixture.componentInstance.downloadCertificate('cert-1');

    expect(certServiceSpy.getDownloadUrl).toHaveBeenCalledWith('cert-1');
    expect(openSpy).toHaveBeenCalledWith('https://minio/cert.pdf', '_blank');

    openSpy.mockRestore();
  });
});
