/**
 * badge.service.spec.ts — BadgeService unit tests (Strict TDD — RED → GREEN).
 *
 * Strategy: HttpTestingController intercepts real HTTP calls
 * (same pattern as course-catalog.service.spec.ts).
 *
 * Covers:
 *  - getMyBadges() sends GET /api/badges/me; returns array
 *  - getRanking() sends GET /api/badges/ranking; returns ranked array
 */
import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ConfirmationService, MessageService } from 'primeng/api';

import { BadgeService } from './badge.service';
import type { BadgeResponse, RankingItem } from './badge.dto';

const BADGES_BASE = 'http://localhost:3000/api/badges';

const MOCK_BADGE: BadgeResponse = {
  id: 'badge-1',
  nombre: 'Primer curso completado',
  descripcion: 'Completaste tu primer curso',
  otorgadoEn: '2026-01-02T00:00:00Z',
};

const MOCK_RANKING: RankingItem[] = [
  { posicion: 1, userNombre: 'Ana Lopez', certCount: 3 },
  { posicion: 2, userNombre: 'Bob Martinez', certCount: 1 },
];

describe('BadgeService', () => {
  let service: BadgeService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        ConfirmationService,
        MessageService,
      ],
    });
    service = TestBed.inject(BadgeService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
    TestBed.resetTestingModule();
  });

  // ── getMyBadges ──────────────────────────────────────────────────────────────

  it('getMyBadges() sends GET /api/badges/me', async () => {
    const promise = service.getMyBadges();

    const req = httpMock.expectOne(`${BADGES_BASE}/me`);
    expect(req.request.method).toBe('GET');
    req.flush({ badges: [MOCK_BADGE] });

    const result = await promise;
    expect(result).toHaveLength(1);
    expect(result[0].nombre).toBe('Primer curso completado');
  });

  it('getMyBadges() returns empty array when backend returns null badges', async () => {
    const promise = service.getMyBadges();

    const req = httpMock.expectOne(`${BADGES_BASE}/me`);
    req.flush({ badges: null });

    const result = await promise;
    expect(result).toHaveLength(0);
  });

  // ── getRanking ───────────────────────────────────────────────────────────────

  it('getRanking() sends GET /api/badges/ranking', async () => {
    const promise = service.getRanking();

    const req = httpMock.expectOne(`${BADGES_BASE}/ranking`);
    expect(req.request.method).toBe('GET');
    req.flush({ ranking: MOCK_RANKING });

    const result = await promise;
    expect(result).toHaveLength(2);
    expect(result[0].posicion).toBe(1);
    expect(result[0].userNombre).toBe('Ana Lopez');
    expect(result[1].certCount).toBe(1);
  });

  it('getRanking() returns empty array when ranking is null', async () => {
    const promise = service.getRanking();

    const req = httpMock.expectOne(`${BADGES_BASE}/ranking`);
    req.flush({ ranking: null });

    const result = await promise;
    expect(result).toHaveLength(0);
  });
});
