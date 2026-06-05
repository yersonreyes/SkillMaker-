/**
 * course-detail-cert.spec.ts — CourseDetailAlumnoComponent "Mi certificado" hook tests (C5.1).
 *
 * Extends existing course-detail specs with certificate-specific scenarios.
 * Does NOT modify the existing spec file (separation of concern).
 *
 * Covers:
 *  - "Mi certificado" button shown when getMyCertificates returns a cert for the current courseId
 *  - "Mi certificado" button absent when no matching certificate
 *  - downloadCertificate() calls getDownloadUrl and window.open when cert is present
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, afterEach, vi } from 'vitest';
import {
  ActivatedRoute,
  convertToParamMap,
} from '@angular/router';
import { of } from 'rxjs';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { CourseDetailAlumnoComponent } from './course-detail.component';
import { CourseCatalogService } from '@core/services/courseCatalogService/course-catalog.service';
import { CertificateService } from '@core/services/certificateService/certificate.service';
import type {
  CoursePreviewResponse,
} from '@core/services/courseCatalogService/course-catalog.dto';
import type { CertificateListItem } from '@core/services/certificateService/certificate.dto';

const MOCK_PREVIEW: CoursePreviewResponse = {
  enrolled: false,
  id: 'course-1',
  titulo: 'Go Avanzado',
  descripcion: 'Aprende Go de verdad',
  creadorNombre: 'Yerson Reyes',
};

const MOCK_CERT_MATCH: CertificateListItem = {
  id: 'cert-1',
  courseId: 'course-1',
  courseTitulo: 'Go Avanzado',
  codigo: 'ABCDE',
  emitidoEn: '2026-01-01T00:00:00Z',
};

const MOCK_CERT_OTHER: CertificateListItem = {
  id: 'cert-2',
  courseId: 'course-99',
  courseTitulo: 'Otro Curso',
  codigo: 'FGHIJ',
  emitidoEn: '2026-01-02T00:00:00Z',
};

async function createComponent(
  catalogSpy: Partial<CourseCatalogService>,
  certSpy: Partial<CertificateService>,
) {
  await TestBed.configureTestingModule({
    imports: [CourseDetailAlumnoComponent],
    providers: [
      { provide: CourseCatalogService, useValue: catalogSpy },
      { provide: CertificateService, useValue: certSpy },
      // Inject ActivatedRoute with id='course-1'. Do NOT use provideRouter here
      // because provideRouter overrides ActivatedRoute snapshot (C5.1 lesson).
      {
        provide: ActivatedRoute,
        useValue: {
          snapshot: { paramMap: convertToParamMap({ id: 'course-1' }) },
          params: of({ id: 'course-1' }),
        },
      },
      provideAnimationsAsync(),
      ConfirmationService,
      MessageService,
    ],
  }).compileComponents();

  const fixture = TestBed.createComponent(CourseDetailAlumnoComponent);
  const comp = fixture.componentInstance;
  return { fixture, comp };
}

describe('CourseDetailAlumnoComponent — Mi certificado hook', () => {
  afterEach(() => {
    TestBed.resetTestingModule();
  });

  it('"Mi certificado" button visible when a matching certificate exists', async () => {
    const catalogSpy: Partial<CourseCatalogService> = {
      getDetail: vi.fn().mockResolvedValue(MOCK_PREVIEW),
      enroll: vi.fn().mockResolvedValue({ courseId: 'course-1', enrolled: true }),
    };
    const certSpy: Partial<CertificateService> = {
      getMyCertificates: vi.fn().mockResolvedValue([MOCK_CERT_MATCH]),
      getDownloadUrl: vi.fn().mockResolvedValue({ url: 'https://minio/cert.pdf', expiresAt: '...' }),
    };

    const { fixture, comp } = await createComponent(catalogSpy, certSpy);
    // Zoneless Angular 21: whenStable() does NOT track void Promises started by ngOnInit.
    // Pattern: set courseId + signal → call loadDetail() explicitly → detectChanges().
    // detectChanges() is called AFTER loadDetail() so loading=false and data is populated.
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseIdSignal'].set('course-1');
    await comp.loadDetail();
    // detectChanges() triggers ngOnInit which sets courseId again (ActivatedRoute returns 'course-1')
    // and calls void loadDetail() — but this async call does NOT affect the current synchronous
    // signal state for the CURRENT detectChanges render. We need a second explicit loadDetail after.
    fixture.detectChanges();
    // Call loadDetail again to settle the ngOnInit-triggered async load
    await comp.loadDetail();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    const certBtn = Array.from(el.querySelectorAll('button')).find(b =>
      b.textContent?.trim().includes('Mi certificado'),
    );
    expect(certBtn).toBeDefined();
  });

  it('"Mi certificado" button absent when no matching certificate', async () => {
    const catalogSpy: Partial<CourseCatalogService> = {
      getDetail: vi.fn().mockResolvedValue(MOCK_PREVIEW),
      enroll: vi.fn().mockResolvedValue({ courseId: 'course-1', enrolled: true }),
    };
    const certSpy: Partial<CertificateService> = {
      getMyCertificates: vi.fn().mockResolvedValue([MOCK_CERT_OTHER]),
      getDownloadUrl: vi.fn().mockResolvedValue({ url: 'https://minio/cert.pdf', expiresAt: '...' }),
    };

    const { fixture, comp } = await createComponent(catalogSpy, certSpy);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseIdSignal'].set('course-1');
    await comp.loadDetail();
    fixture.detectChanges();
    await comp.loadDetail();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    const certBtn = Array.from(el.querySelectorAll('button')).find(b =>
      b.textContent?.trim().includes('Mi certificado'),
    );
    expect(certBtn).toBeUndefined();
  });

  it('"Mi certificado" button absent when certificates list is empty', async () => {
    const catalogSpy: Partial<CourseCatalogService> = {
      getDetail: vi.fn().mockResolvedValue(MOCK_PREVIEW),
      enroll: vi.fn().mockResolvedValue({ courseId: 'course-1', enrolled: true }),
    };
    const certSpy: Partial<CertificateService> = {
      getMyCertificates: vi.fn().mockResolvedValue([]),
      getDownloadUrl: vi.fn().mockResolvedValue({ url: '', expiresAt: '' }),
    };

    const { fixture, comp } = await createComponent(catalogSpy, certSpy);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseId'] = 'course-1';
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (comp as any)['courseIdSignal'].set('course-1');
    await comp.loadDetail();
    fixture.detectChanges();
    await comp.loadDetail();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    const certBtn = Array.from(el.querySelectorAll('button')).find(b =>
      b.textContent?.trim().includes('Mi certificado'),
    );
    expect(certBtn).toBeUndefined();
  });

  it('downloadCertificate() opens presigned URL via window.open', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null);

    const catalogSpy: Partial<CourseCatalogService> = {
      getDetail: vi.fn().mockResolvedValue(MOCK_PREVIEW),
      enroll: vi.fn().mockResolvedValue({ courseId: 'course-1', enrolled: true }),
    };
    const certSpy: Partial<CertificateService> = {
      getMyCertificates: vi.fn().mockResolvedValue([MOCK_CERT_MATCH]),
      getDownloadUrl: vi.fn().mockResolvedValue({ url: 'https://minio/cert.pdf', expiresAt: '...' }),
    };

    const { fixture, comp } = await createComponent(catalogSpy, certSpy);
    fixture.detectChanges();
    await fixture.whenStable();

    await comp.downloadCertificate('cert-1');

    expect(certSpy.getDownloadUrl).toHaveBeenCalledWith('cert-1');
    expect(openSpy).toHaveBeenCalledWith('https://minio/cert.pdf', '_blank');

    openSpy.mockRestore();
  });
});
