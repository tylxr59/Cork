<script lang="ts">
  import ExternalLink from '@lucide/svelte/icons/external-link';
  import LinkIcon from '@lucide/svelte/icons/link';
  import Pencil from '@lucide/svelte/icons/pencil';
  import { domain, formatPrice } from '../lib/format';
  import type { Pin } from '../lib/types';

  let { pin, showBoard = false, canEdit = false }: { pin: Pin; showBoard?: boolean; canEdit?: boolean } = $props();

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
  {#if canEdit || pin.source_url}
    <div class="actions">
      {#if canEdit}
        <a href="/p/{pin.id}?edit" data-modal aria-label="Edit pin" title="Edit"><Pencil size={15} /></a>
      {/if}
      {#if pin.source_url}
        <a href={pin.source_url} target="_blank" rel="noopener noreferrer nofollow" aria-label="Visit site" title="Visit site">
          <ExternalLink size={15} />
        </a>
      {/if}
    </div>
  {/if}
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
  /* One object: the picture and its label share a card. */
  .pin {
    position: relative;
    border-radius: var(--radius);
    overflow: hidden;
    background: var(--side);
    box-shadow: 0 0 0 0.5px var(--hairline);
    transition:
      transform 0.16s cubic-bezier(0.2, 0.8, 0.2, 1),
      box-shadow 0.16s;
  }
  .pin:hover {
    transform: translateY(-2px);
    box-shadow: var(--shadow);
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
    left: 8px;
    bottom: 8px;
    padding: 3px 8px;
    border-radius: 999px;
    background: #fff;
    color: #1c1b1a;
    font-size: 0.78rem;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    box-shadow: 0 1px 4px rgb(0 0 0 / 25%);
  }
  .actions {
    position: absolute;
    top: 8px;
    right: 8px;
    display: flex;
    gap: 4px;
    opacity: 0;
    transition: opacity 0.12s;
  }
  .pin:hover .actions,
  .pin:focus-within .actions {
    opacity: 1;
  }
  .actions a {
    width: 30px;
    height: 30px;
    display: grid;
    place-items: center;
    border-radius: 50%;
    background: #fff;
    color: #1c1b1a;
    box-shadow: 0 1px 4px rgb(0 0 0 / 28%);
  }
  .actions a:hover {
    background: #f0ece6;
  }
  .meta {
    padding: 9px 12px 10px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .title {
    font-size: 0.875rem;
    font-weight: 600;
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
    gap: 5px;
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
    font-weight: 550;
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
      box-shadow: 0 0 0 0.5px var(--hairline);
    }
    .actions {
      display: none;
    }
  }
</style>
