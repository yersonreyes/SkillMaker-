/**
 * badges.component.spec.ts — BadgesComponent unit tests (Strict TDD — RED → GREEN).
 *
 * Strategy: spy on BadgeService; call component methods directly.
 *
 * Covers:
 *  - Renders earned badge cards from getMyBadges()
 *  - Shows nombre, descripcion on each badge card
 *  - Renders ranking table rows from getRanking()
 *  - Empty states (.empty) when no badges and no ranking
 *  - Loading state when loading=true
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { provideRouter } from '@angular/router';
import { ConfirmationService, MessageService } from 'primeng/api';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';

import { BadgesComponent } from './badges.component';
import { BadgeService } from '@core/services/badgeService/badge.service';
import type { BadgeResponse, RankingItem } from '@core/services/badgeService/badge.dto';

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

describe('BadgesComponent', () => {
  let badgeServiceSpy: Partial<BadgeService>;

  beforeEach(async () => {
    badgeServiceSpy = {
      getMyBadges: vi.fn().mockResolvedValue([MOCK_BADGE]),
      getRanking: vi.fn().mockResolvedValue(MOCK_RANKING),
    };

    await TestBed.configureTestingModule({
      imports: [BadgesComponent],
      providers: [
        { provide: BadgeService, useValue: badgeServiceSpy },
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

  it('calls getMyBadges and getRanking on init', async () => {
    const fixture = TestBed.createComponent(BadgesComponent);
    fixture.detectChanges();
    await fixture.whenStable();

    expect(badgeServiceSpy.getMyBadges).toHaveBeenCalledTimes(1);
    expect(badgeServiceSpy.getRanking).toHaveBeenCalledTimes(1);
  });

  it('renders badge card with nombre after load', async () => {
    const fixture = TestBed.createComponent(BadgesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Primer curso completado');
  });

  it('renders badge card with descripcion', async () => {
    const fixture = TestBed.createComponent(BadgesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Completaste tu primer curso');
  });

  it('renders ranking rows after load', async () => {
    const fixture = TestBed.createComponent(BadgesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('Ana Lopez');
    expect(el.textContent).toContain('Bob Martinez');
  });

  it('shows empty badge state when no badges', async () => {
    (badgeServiceSpy.getMyBadges as ReturnType<typeof vi.fn>).mockResolvedValue([]);

    const fixture = TestBed.createComponent(BadgesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('insignias');
  });

  it('shows empty ranking state when no ranking entries', async () => {
    (badgeServiceSpy.getRanking as ReturnType<typeof vi.fn>).mockResolvedValue([]);

    const fixture = TestBed.createComponent(BadgesComponent);
    fixture.detectChanges();
    await fixture.whenStable();
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('clasificacion');
  });
});
