import { Component } from '@angular/core';

@Component({
  selector: 'app-callback',
  standalone: true,
  template: `
    <div class="cb">
      <p class="cb__eyebrow">// Conectando</p>
      <h2 class="cb__display">Trazando<span>.</span></h2>
      <div class="cb__reel" aria-hidden="true">
        <span></span><span></span><span></span><span></span><span></span>
      </div>
      <p class="cb__lede">
        Un momento — estamos abriendo tu acceso a la red de conocimiento.
      </p>
    </div>
  `,
  styles: [
    `
      :host { display: block; color: var(--reel-cream); font-family: var(--font-ui); }
      .cb__eyebrow {
        font-family: var(--font-mono);
        font-size: 10px;
        letter-spacing: 0.28em;
        text-transform: uppercase;
        color: var(--reel-cyan);
        margin: 0 0 16px;
      }
      .cb__display {
        font-family: var(--font-display);
        font-style: italic;
        font-weight: 400;
        font-size: clamp(40px, 5vw, 60px);
        line-height: 1;
        letter-spacing: -0.02em;
        margin: 0 0 28px;
      }
      .cb__display span { color: var(--reel-cyan); font-style: normal; }
      .cb__reel {
        display: flex;
        gap: 6px;
        margin-bottom: 24px;
      }
      .cb__reel span {
        width: 18px;
        height: 4px;
        background: var(--reel-cream-faint);
        border-radius: 1px;
        animation: cb-strip 1.1s ease-in-out infinite;
      }
      .cb__reel span:nth-child(2) { animation-delay: 0.12s; }
      .cb__reel span:nth-child(3) { animation-delay: 0.24s; }
      .cb__reel span:nth-child(4) { animation-delay: 0.36s; }
      .cb__reel span:nth-child(5) { animation-delay: 0.48s; }
      .cb__lede {
        font-weight: 300;
        font-size: 13px;
        line-height: 1.6;
        color: var(--reel-cream-dim);
        margin: 0;
        max-width: 320px;
      }
      @keyframes cb-strip {
        0%, 100% { background: var(--reel-cream-faint); transform: scaleY(1); }
        50% { background: var(--reel-cyan); transform: scaleY(1.6); }
      }
    `,
  ],
})
export class CallbackComponent {}
