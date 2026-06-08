/**
 * verificar-certificado.component.spec.ts — PUBLIC certificate verification page.
 *
 * Covers:
 *  - verify() success → state 'found' + result populated
 *  - verify() 404 → state 'notfound' (no error surfaced)
 *  - verify() transport error → state 'error'
 *  - empty code → no-op (service not called)
 *  - :codigo deep-link auto-verifies on init
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { ActivatedRoute, convertToParamMap } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';

import { VerificarCertificadoComponent } from './verificar-certificado.component';
import { CertificateService } from '@core/services/certificateService/certificate.service';
import type { VerifyCertificateResponse } from '@core/services/certificateService/certificate.dto';

const MOCK_RESULT: VerifyCertificateResponse = {
  codigo: 'ABCD1234EFG12',
  holderNombre: 'Ana Perez',
  courseTitulo: 'Go Avanzado',
  emitidoEn: '2026-06-01T00:00:00Z',
};

describe('VerificarCertificadoComponent', () => {
  let certSpy: Partial<CertificateService>;

  function build(codigoParam: string | null = null) {
    TestBed.configureTestingModule({
      imports: [VerificarCertificadoComponent],
      providers: [
        { provide: CertificateService, useValue: certSpy },
        {
          provide: ActivatedRoute,
          useValue: { snapshot: { paramMap: convertToParamMap(codigoParam ? { codigo: codigoParam } : {}) } },
        },
      ],
    });
    return TestBed.createComponent(VerificarCertificadoComponent);
  }

  function setup(codigoParam: string | null = null) {
    return build(codigoParam).componentInstance;
  }

  beforeEach(() => {
    certSpy = { verify: vi.fn().mockResolvedValue(MOCK_RESULT) };
  });

  it("verify() sets state 'found' and stores the result on success", async () => {
    const comp = setup();
    comp.codigo.set('ABCD1234EFG12');

    await comp.verify();

    expect(certSpy.verify).toHaveBeenCalledWith('ABCD1234EFG12');
    expect(comp.state()).toBe('found');
    expect(comp.result()?.holderNombre).toBe('Ana Perez');
  });

  it("verify() sets state 'notfound' on a 404 (no error surfaced)", async () => {
    certSpy.verify = vi.fn().mockRejectedValue(new HttpErrorResponse({ status: 404 }));
    const comp = setup();
    comp.codigo.set('UNKNOWN');

    await comp.verify();

    expect(comp.state()).toBe('notfound');
    expect(comp.result()).toBeNull();
  });

  it("verify() sets state 'error' on a non-404 transport failure", async () => {
    certSpy.verify = vi.fn().mockRejectedValue(new HttpErrorResponse({ status: 500 }));
    const comp = setup();
    comp.codigo.set('ABCD1234EFG12');

    await comp.verify();

    expect(comp.state()).toBe('error');
  });

  it('verify() is a no-op when the code is empty/whitespace', async () => {
    const comp = setup();
    comp.codigo.set('   ');

    await comp.verify();

    expect(certSpy.verify).not.toHaveBeenCalled();
    expect(comp.state()).toBe('idle');
  });

  it('renders the form and the success panel without template errors', async () => {
    const fixture = build();
    fixture.detectChanges(); // compiles + renders template (catches binding errors vitest+tsc miss)
    expect(fixture.nativeElement.querySelector('#codigo')).toBeTruthy();

    const comp = fixture.componentInstance;
    comp.codigo.set('ABCD1234EFG12');
    await comp.verify();
    fixture.detectChanges();

    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Certificado válido');
    expect(text).toContain('Ana Perez');
    expect(text).toContain('Go Avanzado');
  });

  it('auto-verifies when a :codigo deep-link param is present', async () => {
    const comp = setup('DEEPLINK123');
    comp.ngOnInit();
    // Allow the awaited verify() microtask to settle.
    await Promise.resolve();
    await Promise.resolve();

    expect(comp.codigo()).toBe('DEEPLINK123');
    expect(certSpy.verify).toHaveBeenCalledWith('DEEPLINK123');
  });
});
