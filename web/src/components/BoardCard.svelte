<script lang="ts">
  import Lock from '@lucide/svelte/icons/lock';
  import Globe from '@lucide/svelte/icons/globe';
  import { plural } from '../lib/format';
  import type { Board } from '../lib/types';

  let { board }: { board: Board } = $props();

  const slots = $derived([0, 1, 2].map((i) => board.covers[i]));
</script>

<a class="board" href="/b/{board.id}">
  <div class="board-cover">
    {#each slots as c, i (i)}
      <div class="slot s{i}" style:background={c?.color || 'var(--surface-2)'}>
        {#if c}<img src={c.url} alt="" loading="lazy" draggable="false" />{/if}
      </div>
    {/each}
  </div>
  <div class="info">
    <h3>{board.name}</h3>
    <p>
      <span>{plural(board.pin_count, 'pin')}</span>
      {#if board.is_public}
        <span class="pill public"><Globe size={11} /> Public</span>
      {:else}
        <Lock size={12} aria-label="Private" />
      {/if}
    </p>
  </div>
</a>

<style>
  .board {
    display: block;
    padding: 8px;
    border-radius: var(--radius-lg);
    background: var(--surface);
    box-shadow: var(--shadow);
    transition:
      transform 0.2s,
      box-shadow 0.2s;
  }
  .board:hover {
    transform: translateY(-4px);
    box-shadow: var(--shadow-lg);
  }
  .board-cover {
    display: grid;
    grid-template-columns: 2fr 1fr;
    grid-template-rows: 1fr 1fr;
    gap: 4px;
    aspect-ratio: 4 / 3;
    border-radius: 18px;
    overflow: hidden;
  }
  .slot {
    overflow: hidden;
  }
  .s0 {
    grid-row: span 2;
  }
  .slot img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  .info {
    padding: 10px 8px 6px;
  }
  h3 {
    font-size: 1.02rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  p {
    margin-top: 2px;
    font-size: 0.82rem;
    color: var(--text-2);
    display: flex;
    align-items: center;
    gap: 8px;
  }
  @media (hover: none) {
    .board:hover {
      transform: none;
    }
  }
</style>
