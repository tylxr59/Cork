<script lang="ts">
  import LinkIcon from '@lucide/svelte/icons/link';
  import { domain, formatPrice } from '../lib/format';
  import type { Pin } from '../lib/types';

  let { pin, showBoard = false }: { pin: Pin; showBoard?: boolean } = $props();

  let loaded = $state(false);
  const host = $derived(domain(pin.source_url));
  const price = $derived(formatPrice(pin.price, pin.currency));
</script>

<article class="pin">
  <a class="media" href="/p/{pin.id}" data-modal style:background={pin.color || 'var(--surface-2)'} aria-label={pin.title || 'Open pin'}>
    <img
      src={pin.thumb_url}
      alt={pin.title}
      width={pin.width}
      height={pin.height}
      style:aspect-ratio="{pin.width} / {pin.height}"
      loading="lazy"
      decoding="async"
      draggable="false"
      class:loaded
      onload={() => (loaded = true)}
    />
    {#if price}<span class="price">{price}</span>{/if}
  </a>
  {#if pin.title || host || showBoard}
    <div class="meta">
      {#if pin.title}<a class="title" href="/p/{pin.id}" data-modal>{pin.title}</a>{/if}
      <div class="sub">
        {#if host}
          <a class="src" href={pin.source_url} target="_blank" rel="noopener noreferrer nofollow" title={pin.source_url}>
            <LinkIcon size={12} />
            <span>{host}</span>
          </a>
        {/if}
        {#if showBoard}<a class="board" href="/b/{pin.board_id}">{pin.board_name}</a>{/if}
      </div>
    </div>
  {/if}
</article>

<style>
  .pin {
    border-radius: var(--radius-lg);
    overflow: hidden;
    background: var(--surface);
    box-shadow: var(--shadow);
    transition:
      transform 0.2s,
      box-shadow 0.2s;
  }
  .pin:hover {
    transform: translateY(-4px);
    box-shadow: var(--shadow-lg);
  }
  .media {
    display: block;
    position: relative;
  }
  .media img {
    width: 100%;
    height: auto;
    opacity: 0;
    transition: opacity 0.3s;
  }
  .media img.loaded {
    opacity: 1;
  }
  .price {
    position: absolute;
    left: 10px;
    bottom: 10px;
    padding: 5px 11px;
    border-radius: 999px;
    background: rgb(255 255 255 / 92%);
    color: #1f1d2b;
    font-size: 0.8rem;
    font-weight: 800;
    font-variant-numeric: tabular-nums;
    box-shadow: 0 4px 12px rgb(0 0 0 / 15%);
    -webkit-backdrop-filter: blur(8px);
    backdrop-filter: blur(8px);
  }
  .meta {
    padding: 11px 14px 13px;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .title {
    font-weight: 700;
    line-height: 1.3;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .sub {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.78rem;
    color: var(--text-2);
    min-width: 0;
  }
  .src {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }
  .src :global(svg) {
    flex-shrink: 0;
  }
  .src span,
  .board {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .board {
    margin-left: auto;
    font-weight: 650;
    flex-shrink: 1;
  }
  .src:hover,
  .board:hover,
  .title:hover {
    color: var(--text);
    text-decoration: underline;
  }
  @media (hover: none) {
    .pin:hover {
      transform: none;
      box-shadow: var(--shadow);
    }
  }
</style>
