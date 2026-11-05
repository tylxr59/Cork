<script lang="ts">
  import { api } from '../lib/api';
  import { loadBoards } from '../lib/session.svelte';
  import type { Board } from '../lib/types';
  import Modal from './Modal.svelte';

  let {
    board = null,
    onclose,
    onsaved,
  }: { board?: Board | null; onclose: () => void; onsaved: (b: Board) => void } = $props();

  // svelte-ignore state_referenced_locally
  let name = $state(board?.name ?? '');
  // svelte-ignore state_referenced_locally
  let description = $state(board?.description ?? '');
  // svelte-ignore state_referenced_locally
  let isPublic = $state(board?.is_public ?? false);
  let error = $state('');
  let busy = $state(false);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = '';
    try {
      const data = { name, description, is_public: isPublic };
      const saved = board ? await api.updateBoard(board.id, data) : await api.createBoard(data);
      await loadBoards(true);
      onsaved(saved);
    } catch (err) {
      error = (err as Error).message;
    } finally {
      busy = false;
    }
  }
</script>

<Modal {onclose} label={board ? 'Edit board' : 'Create board'}>
  <form onsubmit={submit}>
    <h2>{board ? 'Edit board' : 'Create board'}</h2>
    <label class="field">
      <span>Name</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input" bind:value={name} maxlength="100" placeholder="e.g. Kitchen ideas" required autofocus />
    </label>
    <label class="field">
      <span>Description</span>
      <textarea class="textarea" bind:value={description} maxlength="2000" placeholder="What's this board about?"></textarea>
    </label>
    <label class="check">
      <input type="checkbox" bind:checked={isPublic} />
      <span>
        <strong>Public</strong><br />
        <small class="muted">Anyone with the link can view this board, even without an account.</small>
      </span>
    </label>
    {#if error}<p class="error-text">{error}</p>{/if}
    <div class="actions">
      <button type="button" class="btn" onclick={onclose}>Cancel</button>
      <button class="btn primary" disabled={busy || !name.trim()}>{board ? 'Save' : 'Create'}</button>
    </div>
  </form>
</Modal>

<style>
  form {
    display: flex;
    flex-direction: column;
    gap: 18px;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }
</style>
