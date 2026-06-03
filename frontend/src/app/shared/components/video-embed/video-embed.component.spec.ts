/**
 * video-embed.component.spec.ts — Unit tests for VideoEmbed component.
 *
 * Covers:
 *  - LB-4: toYoutubeEmbed / toVimeoEmbed table-driven URL transforms
 *  - LB-5: VideoEmbed renders a sanitized embed URL (never the raw user URL)
 *          and malformed / non-matching hosts are NOT bypassed raw
 */
import { TestBed } from '@angular/core/testing';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { DomSanitizer, SafeResourceUrl } from '@angular/platform-browser';
import { ConfirmationService, MessageService } from 'primeng/api';

import { VideoEmbedComponent, toYoutubeEmbed, toVimeoEmbed } from './video-embed.component';

// ── LB-4: Pure transform function table-driven tests ─────────────────────────

describe('toYoutubeEmbed', () => {
  it('VE-1-A: converts youtube.com/watch?v= to embed URL', () => {
    expect(toYoutubeEmbed('https://www.youtube.com/watch?v=dQw4w9WgXcQ')).toBe(
      'https://www.youtube.com/embed/dQw4w9WgXcQ',
    );
  });

  it('VE-1-B: converts youtu.be short URL to embed URL', () => {
    expect(toYoutubeEmbed('https://youtu.be/dQw4w9WgXcQ')).toBe(
      'https://www.youtube.com/embed/dQw4w9WgXcQ',
    );
  });

  it('converts youtube.com/watch?v= without www prefix', () => {
    expect(toYoutubeEmbed('https://youtube.com/watch?v=abc123')).toBe(
      'https://www.youtube.com/embed/abc123',
    );
  });

  it('LB-5: returns empty string for non-youtube host (rejects / ignores)', () => {
    expect(toYoutubeEmbed('https://vimeo.com/123456')).toBe('');
  });

  it('LB-5: returns empty string for dailymotion host', () => {
    expect(toYoutubeEmbed('https://dailymotion.com/video/x7')).toBe('');
  });

  it('LB-5: returns empty string for malformed URL', () => {
    expect(toYoutubeEmbed('not-a-url')).toBe('');
  });

  it('LB-5: returns empty string when v param is missing', () => {
    expect(toYoutubeEmbed('https://www.youtube.com/watch')).toBe('');
  });
});

describe('toVimeoEmbed', () => {
  it('VE-1-C: converts vimeo.com/:id to player embed URL', () => {
    expect(toVimeoEmbed('https://vimeo.com/123456789')).toBe(
      'https://player.vimeo.com/video/123456789',
    );
  });

  it('converts vimeo.com/:id without trailing slash', () => {
    expect(toVimeoEmbed('https://vimeo.com/999')).toBe(
      'https://player.vimeo.com/video/999',
    );
  });

  it('LB-5: returns empty string for non-vimeo host', () => {
    expect(toVimeoEmbed('https://youtube.com/watch?v=abc')).toBe('');
  });

  it('LB-5: returns empty string for malformed URL', () => {
    expect(toVimeoEmbed('not-a-url')).toBe('');
  });

  it('LB-5: returns empty string when no video ID is present', () => {
    expect(toVimeoEmbed('https://vimeo.com/')).toBe('');
  });
});

// ── LB-5: Component DomSanitizer bypass tests ─────────────────────────────────

describe('VideoEmbedComponent', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [VideoEmbedComponent],
      providers: [
        ConfirmationService,
        MessageService,
        provideAnimationsAsync(),
      ],
    }).compileComponents();
  });

  afterEach(() => {
    TestBed.resetTestingModule();
  });

  it('VE-1-D: safeUrl() for youtube input is a SafeResourceUrl wrapping the embed URL (not the raw URL)', () => {
    const fixture = TestBed.createComponent(VideoEmbedComponent);
    const comp = fixture.componentInstance;

    comp.url = 'https://www.youtube.com/watch?v=abc';
    comp.proveedor = 'youtube';
    fixture.detectChanges();

    const safe = comp.safeUrl() as SafeResourceUrl;
    // DomSanitizer wraps the value — access via the sanitizer
    const sanitizer = TestBed.inject(DomSanitizer);
    // The safe value must NOT equal the raw URL — it is sanitized/wrapped
    expect(safe).not.toBe('https://www.youtube.com/watch?v=abc');
    // Verify the sanitizer produced a SafeValue (not a plain string)
    expect(safe).toBeTruthy();
    // Re-sanitize the computed embed URL — must produce the same object class
    const expectedSafe = sanitizer.bypassSecurityTrustResourceUrl(
      'https://www.youtube.com/embed/abc',
    );
    // Both should serialize to the same underlying URL string
    expect(sanitizer.sanitize(5 /* SecurityContext.RESOURCE_URL */, safe)).toBe(
      sanitizer.sanitize(5, expectedSafe),
    );
  });

  it('VE-1-E: safeUrl() for vimeo input is a SafeResourceUrl wrapping the embed URL', () => {
    const fixture = TestBed.createComponent(VideoEmbedComponent);
    const comp = fixture.componentInstance;

    comp.url = 'https://vimeo.com/999';
    comp.proveedor = 'vimeo';
    fixture.detectChanges();

    const safe = comp.safeUrl() as SafeResourceUrl;
    const sanitizer = TestBed.inject(DomSanitizer);
    const expectedSafe = sanitizer.bypassSecurityTrustResourceUrl(
      'https://player.vimeo.com/video/999',
    );

    expect(sanitizer.sanitize(5, safe)).toBe(sanitizer.sanitize(5, expectedSafe));
  });

  it('LB-5: safeUrl() returns null for unknown proveedor (no bypass of arbitrary input)', () => {
    const fixture = TestBed.createComponent(VideoEmbedComponent);
    const comp = fixture.componentInstance;

    comp.url = 'https://evil.com/video';
    // Cast to force unknown proveedor to verify safety guard
    (comp as unknown as { proveedor: string }).proveedor = 'dailymotion';
    fixture.detectChanges();

    expect(comp.safeUrl()).toBeNull();
  });

  it('renders an iframe element with the computed embed URL as src', () => {
    const fixture = TestBed.createComponent(VideoEmbedComponent);
    const comp = fixture.componentInstance;

    comp.url = 'https://www.youtube.com/watch?v=test123';
    comp.proveedor = 'youtube';
    fixture.detectChanges();

    const iframe: HTMLIFrameElement | null =
      fixture.nativeElement.querySelector('iframe');
    expect(iframe).not.toBeNull();
    expect(iframe!.src).toContain('youtube.com/embed/test123');
  });
});
