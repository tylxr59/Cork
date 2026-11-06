<script lang="ts">
  import Lock from '@lucide/svelte/icons/lock';
  import Globe from '@lucide/svelte/icons/globe';
  import { plural } from '../lib/format';
  import type { Board } from '../lib/types';

  let { board }: { board: Board } = $props();

  const cover = $derived(board.covers[0]);
</script>

<a class="board" href="/b/{board.id}">
  <div class="board-cover" style:background={cover?.color || 'var(--surface-2)'}>
    {#if cover}<img src={cover.url} alt="" loading="lazy" draggable="false" />{/if}
  </div>
  <div class="info">
    <h3>{board.name}</h3>
    <p>
      <span>{plural(board.pin_count, 'pin')}</span>
      {#if board.is_public}
        <span class="public"><Globe size={12} /> Public</span>
      {:else}
        <Lock size={12} aria-label="Private" />
      {/if}
    </p>
  </div>
</a>

<style>
  /* An album: one cover with the rest of the stack peeking out behind it. */
  .board {
    position: relative;
    display: block;
    padding-top: 10px;
  }
  .board::before,
  .board::after {
    content: '';
    position: absolute;
    height: 18px;
    border-radius: 8px 8px 0 0;
  }
  .board::before {
    top: 0;
    left: 12%;
    right: 12%;
    background: color-mix(in srgb, var(--text) 7%, var(--bg));
  }
  .board::after {
    top: 4px;
    left: 6%;
    right: 6%;
    background: color-mix(in srgb, var(--text) 14%, var(--bg));
  }
  .board-cover {
    position: relative;
    z-index: 1;
    aspect-ratio: 1;
    border-radius: var(--radius);
    overflow: hidden;
    box-shadow: 0 0 0 0.5px var(--hairline);
    transition: transform 0.16s cubic-bezier(0.2, 0.8, 0.2, 1);
  }
  .board:hover .board-cover {
    transform: translateY(-2px);
  }
  .board-cover img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  .info {
    padding-top: 9px;
  }
  h3 {
    font-size: 0.875rem;
    letter-spacing: -0.01em;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  p {
    font-size: 0.8125rem;
    color: var(--text-2);
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .public {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: var(--accent);
    font-size: 0.75rem;
  }
  @media (hover: none) {
    .board:hover .board-cover {
      transform: none;
    }
  }
</style>
