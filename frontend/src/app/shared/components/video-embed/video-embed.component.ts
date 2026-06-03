import { Component, Input, inject } from '@angular/core';
import { DomSanitizer, SafeResourceUrl } from '@angular/platform-browser';

/**
 * Converts a YouTube watch or short URL to the canonical embed URL.
 * Returns an empty string for non-YouTube hosts or malformed input (LB-5).
 *
 * Supported formats:
 *  - https://www.youtube.com/watch?v=ID
 *  - https://youtube.com/watch?v=ID
 *  - https://youtu.be/ID
 */
export function toYoutubeEmbed(url: string): string {
  try {
    const parsed = new URL(url);
    const host = parsed.hostname.replace(/^www\./, '');

    if (host === 'youtube.com') {
      const id = parsed.searchParams.get('v');
      if (!id) return '';
      return `https://www.youtube.com/embed/${id}`;
    }

    if (host === 'youtu.be') {
      // pathname = /ID
      const id = parsed.pathname.slice(1);
      if (!id) return '';
      return `https://www.youtube.com/embed/${id}`;
    }

    return '';
  } catch {
    return '';
  }
}

/**
 * Converts a Vimeo URL to the canonical player embed URL.
 * Returns an empty string for non-Vimeo hosts or malformed input (LB-5).
 *
 * Supported format:
 *  - https://vimeo.com/:id
 */
export function toVimeoEmbed(url: string): string {
  try {
    const parsed = new URL(url);
    const host = parsed.hostname.replace(/^www\./, '');

    if (host !== 'vimeo.com') return '';

    // pathname = /123456789
    const id = parsed.pathname.slice(1);
    if (!id) return '';
    return `https://player.vimeo.com/video/${id}`;
  } catch {
    return '';
  }
}

/**
 * VideoEmbed — standalone component that renders an embedded video iframe.
 *
 * SECURITY (LB-5): Only the computed embed URL is passed to
 * DomSanitizer.bypassSecurityTrustResourceUrl — NEVER the raw user-supplied
 * `url` input. Non-YouTube/Vimeo hosts produce an empty embed URL and the
 * component renders nothing.
 *
 * Usage:
 *   <app-video-embed url="https://www.youtube.com/watch?v=ID" proveedor="youtube" />
 */
@Component({
  selector: 'app-video-embed',
  standalone: true,
  templateUrl: './video-embed.component.html',
})
export class VideoEmbedComponent {
  @Input() url = '';
  @Input() proveedor: 'youtube' | 'vimeo' = 'youtube';

  private readonly sanitizer = inject(DomSanitizer);

  /**
   * Computes the validated embed URL and wraps it with DomSanitizer.
   * Returns null when the embed URL cannot be computed (bad host / malformed).
   */
  safeUrl(): SafeResourceUrl | null {
    let embedUrl: string;

    if (this.proveedor === 'youtube') {
      embedUrl = toYoutubeEmbed(this.url);
    } else if (this.proveedor === 'vimeo') {
      embedUrl = toVimeoEmbed(this.url);
    } else {
      return null;
    }

    if (!embedUrl) return null;
    return this.sanitizer.bypassSecurityTrustResourceUrl(embedUrl);
  }
}
