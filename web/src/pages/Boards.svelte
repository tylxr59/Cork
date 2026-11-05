<script lang="ts">
  import Sortable from 'sortablejs';
  import Plus from '@lucide/svelte/icons/plus';
  import { api } from '../lib/api';
  import { navigate } from '../lib/router.svelte';
  import { plural } from '../lib/format';
  import { loadBoards, myBoards, toastError } from '../lib/session.svelte';
  import BoardCard from '../components/BoardCard.svelte';
  import BoardForm from '../components/BoardForm.svelte';

  let loading = $state(true);
  let creating = $state(false);
  let grid: HTMLDivElement | undefined = $state();
  const totalPins = $derived(myBoards.list.reduce((n, b) => n + b.pin_count, 0));

  loadBoards(true)
    .catch(toastError)
    .finally(() => (loading = false));

  $effect(() => {
    if (!grid) return;
    const s = Sortable.create(grid, {
      animation: 180,
      delay: 180,
      delayOnTouchOnly: true,
      onEnd(evt) {
        const { oldIndex: from, newIndex: to, item } = evt;
        if (from === undefined || to === undefined || from === to) return;
        item.remove();
        grid!.insertBefore(item, grid!.children[from] ?? null);
        const next = [...myBoards.list];
        const [moved] = next.splice(from, 1);
        next.splice(to, 0, moved);
        myBoards.list = next;
        api.orderBoards(next.map((b) => b.id)).catch(toastError);
      },
    });
    return () => s.destroy();
  });
</script>

<div class="page">
  <div class="head">
    <div>
      <h1>Your boards</h1>
      <p class="muted">
        {plural(myBoards.list.length, 'board')} · {plural(totalPins, 'pin')}{myBoards.list.length > 1
          ? ' · drag to rearrange'
          : ''}
      </p>
    </div>
    <button class="btn" onclick={() => (creating = true)}><Plus size={18} /> New board</button>
  </div>

  {#if loading && !myBoards.loaded}
    <div class="spinner"></div>
  {:else if myBoards.list.length === 0}
    <div class="empty">
      <h2>No boards yet</h2>
      <p>Boards hold your pins. Make one for each idea, project or collection.</p>
      <button class="btn primary" onclick={() => (creating = true)}><Plus size={18} /> Create your first board</button>
    </div>
  {:else}
    <div class="grid" bind:this={grid}>
      {#each myBoards.list as board (board.id)}
        <BoardCard {board} />
      {/each}
    </div>
  {/if}
</div>

{#if creating}
  <BoardForm onclose={() => (creating = false)} onsaved={(b) => navigate(`/b/${b.id}`)} />
{/if}

<style>
  .head {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 16px;
    margin: 4px 0 24px;
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(min(240px, calc(50% - 8px)), 1fr));
    gap: 20px;
  }
  @media (max-width: 520px) {
    .grid {
      gap: 12px;
    }
  }
  .grid :global(.board) {
    cursor: grab;
  }
</style>
