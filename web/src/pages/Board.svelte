<script lang="ts">
  import Plus from '@lucide/svelte/icons/plus';
  import Pencil from '@lucide/svelte/icons/pencil';
  import Trash from '@lucide/svelte/icons/trash-2';
  import Globe from '@lucide/svelte/icons/globe';
  import Lock from '@lucide/svelte/icons/lock';
  import LinkIcon from '@lucide/svelte/icons/link';
  import { api, ApiError } from '../lib/api';
  import { plural } from '../lib/format';
  import { navigate } from '../lib/router.svelte';
  import { loadBoards, onPinEvent, toast, toastError } from '../lib/session.svelte';
  import type { Board, Pin } from '../lib/types';
  import Masonry from '../components/Masonry.svelte';
  import BoardForm from '../components/BoardForm.svelte';

  let { id }: { id: string } = $props();

  let board = $state<Board | null>(null);
  let pins = $state<Pin[]>([]);
  let notFound = $state(false);
  let editing = $state(false);

  $effect(() => {
    const boardId = id;
    board = null;
    notFound = false;
    api
      .board(boardId)
      .then((r) => {
        board = r.board;
        pins = r.pins;
      })
      .catch((e) => {
        if (e instanceof ApiError && e.status === 404) notFound = true;
        else toastError(e);
      });
  });

  $effect(() =>
    onPinEvent((e) => {
      if (e.type === 'deleted') pins = pins.filter((p) => p.id !== e.id);
      if (e.type === 'updated') {
        if (e.pin.board_id !== id) pins = pins.filter((p) => p.id !== e.pin.id);
        else pins = pins.map((p) => (p.id === e.pin.id ? e.pin : p));
      }
    }),
  );

  function reorder(next: Pin[]) {
    pins = next;
    api.orderPins(id, next.map((p) => p.id)).catch(toastError);
  }

  async function togglePublic() {
    if (!board) return;
    try {
      board = await api.updateBoard(board.id, { is_public: !board.is_public });
      toast(board.is_public ? 'Board is now public' : 'Board is now private');
      loadBoards(true);
    } catch (e) {
      toastError(e);
    }
  }

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(location.href);
      toast('Link copied');
    } catch {
      toast(location.href);
    }
  }

  async function remove() {
    if (!board) return;
    if (!confirm(`Delete “${board.name}” and its ${plural(pins.length, 'pin')}? This can't be undone.`)) return;
    try {
      await api.deleteBoard(board.id);
      await loadBoards(true);
      toast('Board deleted');
      navigate('/', { replace: true });
    } catch (e) {
      toastError(e);
    }
  }
</script>

<svelte:head><title>{board ? `${board.name} · Cork` : 'Cork'}</title></svelte:head>

<div class="page">
  {#if notFound}
    <div class="empty">
      <h2>Board not found</h2>
      <p>It may be private or have been deleted.</p>
      <a class="btn" href="/">Back to boards</a>
    </div>
  {:else if !board}
    <div class="spinner"></div>
  {:else}
    <section class="hero glass panel">
      {#if pins.length > 0}
        <div class="covers" aria-hidden="true">
          {#each pins.slice(0, 3) as p (p.id)}
            <img src={p.thumb_url} alt="" style:background={p.color} />
          {/each}
        </div>
      {/if}
      <div class="text">
        <h1>
          {board.name}
          {#if board.is_public}
            <span class="pill public"><Globe size={12} /> Public</span>
          {:else}
            <span class="pill"><Lock size={12} /> Private</span>
          {/if}
        </h1>
        {#if board.description}<p class="desc">{board.description}</p>{/if}
        <p class="muted meta">
          {plural(pins.length, 'pin')}
          {#if !board.can_edit}· by {board.owner}{:else if pins.length > 1}· drag pins to reorder{/if}
        </p>
      </div>
      {#if board.can_edit}
        <div class="actions">
          <a class="btn primary add" href="/new?board={board.id}"><Plus size={18} /> <span>Add pin</span></a>
          <button class="btn" onclick={() => (editing = true)}><Pencil size={16} /> <span>Edit</span></button>
          <button class="btn" onclick={togglePublic} title={board.is_public ? 'Make private' : 'Make public'}>
            {#if board.is_public}<Lock size={16} /> <span>Make private</span>{:else}<Globe size={16} /> <span>Make public</span>{/if}
          </button>
          {#if board.is_public}
            <button class="btn" onclick={copyLink}><LinkIcon size={16} /> <span>Copy link</span></button>
          {/if}
          <button class="btn icon danger" aria-label="Delete board" title="Delete board" onclick={remove}>
            <Trash size={16} />
          </button>
        </div>
      {/if}
    </section>

    {#if pins.length === 0}
      <div class="empty">
        <h2>This board is empty</h2>
        {#if board.can_edit}
          <a class="btn primary" href="/new?board={board.id}"><Plus size={18} /> Add the first pin</a>
        {/if}
      </div>
    {:else}
      <Masonry {pins} sortable={board.can_edit} onreorder={reorder} />
    {/if}
  {/if}
</div>

{#if editing && board}
  <BoardForm
    {board}
    onclose={() => (editing = false)}
    onsaved={(b) => {
      board = b;
      editing = false;
    }}
  />
{/if}

<style>
  .hero {
    display: flex;
    align-items: center;
    gap: 24px;
    padding: 24px 28px;
    margin: 4px 0 24px;
    flex-wrap: wrap;
  }
  .covers {
    display: flex;
    flex-shrink: 0;
  }
  .covers img {
    width: 64px;
    height: 64px;
    border-radius: 18px;
    object-fit: cover;
    border: 3px solid var(--surface);
    margin-left: -16px;
    box-shadow: var(--shadow-sm);
  }
  .covers img:first-child {
    margin-left: 0;
  }
  .text {
    flex: 1;
    min-width: 220px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  h1 {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px 12px;
    overflow-wrap: anywhere;
  }
  .desc {
    white-space: pre-line;
  }
  .meta {
    font-size: 0.875rem;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  @media (max-width: 1280px) {
    .actions {
      width: 100%;
    }
  }
  @media (max-width: 640px) {
    .hero {
      padding: 18px;
      gap: 16px;
    }
    .covers img {
      width: 52px;
      height: 52px;
    }
    .actions .btn:not(.add) span {
      display: none;
    }
    .actions .btn:not(.add) {
      width: 44px;
      padding: 0;
    }
    .actions .add {
      flex: 1;
    }
  }
</style>
