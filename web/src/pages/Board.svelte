<script lang="ts">
  import Plus from '@lucide/svelte/icons/plus';
  import Pencil from '@lucide/svelte/icons/pencil';
  import Trash from '@lucide/svelte/icons/trash-2';
  import Globe from '@lucide/svelte/icons/globe';
  import Lock from '@lucide/svelte/icons/lock';
  import LinkIcon from '@lucide/svelte/icons/link';
  import Upload from '@lucide/svelte/icons/upload';
  import { api, ApiError } from '../lib/api';
  import { plural } from '../lib/format';
  import { droppedFiles, importFiles } from '../lib/imports.svelte';
  import { navigate } from '../lib/router.svelte';
  import { loadBoards, onPinEvent, toast, toastError } from '../lib/session.svelte';
  import type { Board, Pin } from '../lib/types';
  import Masonry from '../components/Masonry.svelte';
  import PinSizeControl from '../components/PinSizeControl.svelte';
  import BoardForm from '../components/BoardForm.svelte';

  let { id }: { id: string } = $props();

  let board = $state<Board | null>(null);
  let pins = $state<Pin[]>([]);
  let notFound = $state(false);
  let editing = $state(false);
  let dropping = $state(false);
  let fileInput: HTMLInputElement | undefined = $state();
  // dragenter/dragleave fire for every child crossed; the overlay goes when they balance
  let dragDepth = 0;
  // set while dragging something from this page, like a pin being reordered
  let draggingOwn = false;

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
      if (e.type === 'created' && e.pin.board_id === id && !pins.some((p) => p.id === e.pin.id)) {
        pins = [e.pin, ...pins];
      }
      if (e.type === 'deleted') pins = pins.filter((p) => p.id !== e.id);
      if (e.type === 'updated') {
        if (e.pin.board_id !== id) pins = pins.filter((p) => p.id !== e.pin.id);
        else pins = pins.map((p) => (p.id === e.pin.id ? e.pin : p));
      }
    }),
  );

  const carriesFiles = (e: DragEvent) => !!board?.can_edit && !draggingOwn && !!e.dataTransfer?.types.includes('Files');

  function onDragEnter(e: DragEvent) {
    if (!carriesFiles(e)) return;
    dragDepth++;
    dropping = true;
  }

  function onDragOver(e: DragEvent) {
    if (carriesFiles(e)) e.preventDefault();
  }

  function onDragLeave(e: DragEvent) {
    if (carriesFiles(e) && --dragDepth <= 0) stopDropping();
  }

  function stopDropping() {
    dragDepth = 0;
    dropping = false;
  }

  async function onDrop(e: DragEvent) {
    if (!carriesFiles(e)) return;
    e.preventDefault();
    stopDropping();
    importFiles(id, await droppedFiles(e.dataTransfer!));
  }

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

<svelte:window
  ondragstart={() => (draggingOwn = true)}
  ondragend={() => (draggingOwn = false)}
  ondragenter={onDragEnter}
  ondragover={onDragOver}
  ondragleave={onDragLeave}
  ondrop={onDrop}
/>
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
    <section class="hero">
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
      <div class="actions">
        {#if board.can_edit}
          <a class="btn primary add" href="/new?board={board.id}"><Plus size={18} /> <span>Add pin</span></a>
          <button class="btn" onclick={() => fileInput?.click()} title="Add lots of images at once, or drop them anywhere on this page">
            <Upload size={16} /> <span>Import</span>
          </button>
          <input
            bind:this={fileInput}
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            multiple
            hidden
            onchange={(e) => {
              if (e.currentTarget.files) importFiles(id, [...e.currentTarget.files]);
              e.currentTarget.value = '';
            }}
          />
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
        {/if}
        <PinSizeControl />
      </div>
    </section>

    {#if pins.length === 0}
      <div class="empty">
        <h2>This board is empty</h2>
        {#if board.can_edit}
          <a class="btn primary" href="/new?board={board.id}"><Plus size={18} /> Add the first pin</a>
          <p class="muted">Or drop a pile of images or a folder anywhere on this page.</p>
        {/if}
      </div>
    {:else}
      <Masonry {pins} sortable={board.can_edit} canEdit={board.can_edit} onreorder={reorder} />
    {/if}
  {/if}
</div>

{#if dropping}
  <div class="dropzone" aria-hidden="true">
    <Upload size={40} />
    <strong>Drop to add to {board?.name}</strong>
    <span>Each image becomes a pin. Folders work too.</span>
  </div>
{/if}

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
  .dropzone {
    position: fixed;
    inset: 12px;
    z-index: 120;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    border: 3px dashed var(--accent);
    border-radius: var(--radius-xl);
    background: var(--accent-soft);
    color: var(--text);
    text-align: center;
    /* the overlay must not become the drag target, or it flickers */
    pointer-events: none;
  }
  .dropzone :global(svg) {
    color: var(--accent);
  }
  .hero {
    display: flex;
    align-items: center;
    gap: 16px;
    margin: 0 0 22px;
    flex-wrap: wrap;
  }
  .text {
    flex: 1;
    min-width: 220px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  h1 {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 4px 10px;
    overflow-wrap: anywhere;
  }
  .desc {
    white-space: pre-line;
  }
  .meta {
    font-size: 0.8125rem;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
  }
  @media (max-width: 1180px) {
    .actions {
      width: 100%;
    }
  }
  @media (max-width: 640px) {
    .hero {
      gap: 12px;
    }
    .actions .btn:not(.add) span {
      display: none;
    }
    .actions .btn:not(.add) {
      width: 40px;
      padding: 0;
    }
    .actions .add {
      flex: 1;
    }
  }
</style>
