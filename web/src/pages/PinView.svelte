<script lang="ts">
  import ExternalLink from '@lucide/svelte/icons/external-link';
  import Pencil from '@lucide/svelte/icons/pencil';
  import Trash from '@lucide/svelte/icons/trash-2';
  import Download from '@lucide/svelte/icons/download';
  import ImageUp from '@lucide/svelte/icons/image-up';
  import LinkIcon from '@lucide/svelte/icons/link';
  import Tag from '@lucide/svelte/icons/tag';
  import Lightbox from '../components/Lightbox.svelte';
  import { api, ApiError } from '../lib/api';
  import { domain, formatDate, formatDateTime, formatPrice } from '../lib/format';
  import { currencyHints, tidyPrice } from '../lib/currency';
  import { closeModal, navigate } from '../lib/router.svelte';
  import { emitPin, loadBoards, myBoards, toast, toastError } from '../lib/session.svelte';
  import type { Pin } from '../lib/types';

  let { id, modal = false }: { id: string; modal?: boolean } = $props();

  let pin = $state<Pin | null>(null);
  let notFound = $state(false);
  let editing = $state(false);
  let expanded = $state(false);
  let busy = $state(false);
  let form = $state({ title: '', description: '', source_url: '', board_id: '', price: '', currency: '' });
  let fileInput: HTMLInputElement | undefined = $state();

  const price = $derived(pin ? formatPrice(pin.price, pin.currency) : '');

  $effect(() => {
    const pinId = id;
    pin = null;
    notFound = false;
    editing = false;
    expanded = false;
    api
      .pin(pinId)
      .then((p) => (pin = p))
      .catch((e) => {
        if (e instanceof ApiError && e.status === 404) notFound = true;
        else toastError(e);
      });
  });

  const fallbackCurrency = $derived(currencyHints(form.source_url)[0]);

  function cleanPrice() {
    const t = tidyPrice(form.price, form.currency, currencyHints(form.source_url));
    if (t) ({ price: form.price, currency: form.currency } = t);
  }

  function startEdit() {
    if (!pin) return;
    form = {
      title: pin.title,
      description: pin.description,
      source_url: pin.source_url,
      board_id: pin.board_id,
      price: pin.price,
      currency: pin.currency,
    };
    editing = true;
    loadBoards().catch(toastError);
  }

  async function save(e: SubmitEvent) {
    e.preventDefault();
    if (!pin) return;
    cleanPrice();
    if (form.price.trim() && !form.currency.trim()) form.currency = fallbackCurrency;
    busy = true;
    try {
      const moved = form.board_id !== pin.board_id;
      pin = await api.updatePin(pin.id, form);
      emitPin({ type: 'updated', pin });
      editing = false;
      toast(moved ? `Moved to ${pin.board_name}` : 'Pin saved');
    } catch (err) {
      toastError(err);
    } finally {
      busy = false;
    }
  }

  async function replaceImage(file: File) {
    if (!pin) return;
    const fd = new FormData();
    fd.append('image', file);
    busy = true;
    try {
      pin = await api.replacePinImage(pin.id, fd);
      emitPin({ type: 'updated', pin });
      toast('Image replaced');
    } catch (err) {
      toastError(err);
    } finally {
      busy = false;
    }
  }

  async function remove() {
    if (!pin || !confirm('Delete this pin? This can’t be undone.')) return;
    try {
      await api.deletePin(pin.id);
      emitPin({ type: 'deleted', id: pin.id });
      toast('Pin deleted');
      if (modal) closeModal();
      else navigate(`/b/${pin.board_id}`, { replace: true });
    } catch (err) {
      toastError(err);
    }
  }
</script>

<svelte:head><title>{pin?.title ? `${pin.title} · Cork` : 'Pin · Cork'}</title></svelte:head>

{#if notFound}
  <div class="empty">
    <h2>Pin not found</h2>
    <p>It may be private or have been deleted.</p>
  </div>
{:else if !pin}
  <div class="spinner"></div>
{:else}
  <div class="pin-view" class:modal class:page-card={!modal}>
    <button type="button" class="image" aria-label="Expand image" title="Expand" style:background={pin.color || 'var(--surface-2)'} onclick={() => (expanded = true)}>
      <img src={pin.image_url} alt={pin.title} width={pin.width} height={pin.height} />
    </button>
    {#if expanded}
      <Lightbox src={pin.image_url} alt={pin.title} onclose={() => (expanded = false)} />
    {/if}

    <div class="body">
      {#if editing}
        <form onsubmit={save}>
          <h2>Edit pin</h2>
          <label class="field">
            <span>Board</span>
            <select class="select" bind:value={form.board_id}>
              {#each myBoards.list as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
              {#if !myBoards.loaded}<option value={pin.board_id}>{pin.board_name}</option>{/if}
            </select>
          </label>
          <label class="field">
            <span>Title</span>
            <input class="input" bind:value={form.title} maxlength="300" />
          </label>
          <label class="field">
            <span>Description</span>
            <textarea class="textarea" rows="5" bind:value={form.description}></textarea>
          </label>
          <label class="field">
            <span>Link</span>
            <input class="input" type="url" bind:value={form.source_url} placeholder="https://" />
          </label>
          <div class="price-row">
            <label class="field">
              <span>Price when saved</span>
              <input class="input" bind:value={form.price} onblur={cleanPrice} placeholder="e.g. 24.99" />
            </label>
            <label class="field cur">
              <span>Currency</span>
              <input class="input" bind:value={form.currency} maxlength="4" placeholder={fallbackCurrency} title="A code like USD, or a symbol" />
            </label>
          </div>
          <button type="button" class="btn small replace" disabled={busy} onclick={() => fileInput?.click()}>
            <ImageUp size={16} /> Replace image
          </button>
          <input
            bind:this={fileInput}
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            hidden
            onchange={(e) => {
              const f = e.currentTarget.files?.[0];
              if (f) replaceImage(f);
              e.currentTarget.value = '';
            }}
          />
          <div class="actions">
            <button type="button" class="btn danger" onclick={remove}><Trash size={16} /> Delete</button>
            <span class="grow"></span>
            <button type="button" class="btn" onclick={() => (editing = false)}>Cancel</button>
            <button class="btn primary" disabled={busy}>Save</button>
          </div>
        </form>
      {:else}
        <div class="toolbar">
          {#if pin.can_edit}
            <button class="btn icon" aria-label="Edit pin" title="Edit" onclick={startEdit}><Pencil size={18} /></button>
          {/if}
          <a class="btn icon" href={pin.image_url} download aria-label="Download image" title="Download">
            <Download size={18} />
          </a>
          <span class="grow"></span>
          {#if pin.source_url}
            <a class="btn primary" href={pin.source_url} target="_blank" rel="noopener noreferrer nofollow">
              Visit site <ExternalLink size={16} />
            </a>
          {/if}
        </div>

        {#if pin.title}<h1>{pin.title}</h1>{/if}
        {#if pin.description}<p class="desc">{pin.description}</p>{/if}

        {#if price}
          <div class="price">
            <Tag size={18} />
            <div>
              <strong>{price}</strong>
              <small>Price when saved on {formatDate(pin.created_at)}</small>
            </div>
          </div>
        {/if}

        {#if pin.source_url}
          <a class="src" href={pin.source_url} target="_blank" rel="noopener noreferrer nofollow">
            <span class="src-icon"><LinkIcon size={16} /></span>
            <span class="src-text">
              <strong>{domain(pin.source_url)}</strong>
              <small>{pin.source_url}</small>
            </span>
          </a>
        {/if}

        <div class="meta">
          <a href="/b/{pin.board_id}"><small>Board</small><b>{pin.board_name}</b></a>
          <div title={formatDateTime(pin.created_at)}><small>Added</small><b>{formatDate(pin.created_at)}</b></div>
          {#if !pin.can_edit}
            <div><small>Pinned by</small><b>{pin.owner}</b></div>
          {:else}
            <div><small>Size</small><b>{pin.width} × {pin.height}</b></div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .pin-view {
    display: flex;
    flex-direction: column;
  }
  .image {
    display: flex;
    justify-content: center;
    width: 100%;
    padding: 0;
    border: none;
    cursor: zoom-in;
  }
  .image img {
    width: 100%;
    height: auto;
    object-fit: contain;
  }
  .modal .image img {
    max-height: 70vh;
  }
  .body {
    padding: 22px 26px 30px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-width: 0;
  }

  /* full page: side by side card */
  .page-card {
    flex-direction: row;
    background: var(--surface);
    border-radius: var(--radius-xl);
    box-shadow: var(--shadow);
    overflow: hidden;
    margin-top: 4px;
  }
  .page-card .image {
    flex: 1.2;
    align-items: center;
    min-width: 0;
  }
  .page-card .image img {
    max-height: 85vh;
  }
  .page-card .body {
    flex: 1;
    padding: 28px;
  }

  .toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .modal .toolbar {
    padding-right: 0;
  }
  .grow {
    flex: 1;
  }
  h1 {
    font-size: 1.75rem;
    overflow-wrap: anywhere;
  }
  .desc {
    white-space: pre-line;
    overflow-wrap: anywhere;
    color: var(--text-2);
  }
  .price {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 14px 16px;
    border-radius: var(--radius);
    background: var(--accent-soft);
    color: var(--accent);
  }
  .price div {
    display: flex;
    flex-direction: column;
    line-height: 1.25;
  }
  .price strong {
    font-size: 1.35rem;
    font-weight: 800;
    color: var(--text);
    font-variant-numeric: tabular-nums;
  }
  .price small {
    color: var(--text-2);
    font-size: 0.8rem;
  }
  .src {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 14px;
    border-radius: var(--radius);
    background: var(--surface-2);
    min-width: 0;
  }
  .src:hover {
    background: var(--surface-3);
  }
  .src-icon {
    width: 36px;
    height: 36px;
    border-radius: 12px;
    display: grid;
    place-items: center;
    background: var(--surface);
    flex-shrink: 0;
  }
  .src-text {
    display: flex;
    flex-direction: column;
    min-width: 0;
    line-height: 1.3;
  }
  .src-text small {
    color: var(--text-2);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .meta {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 10px;
  }
  .meta > * {
    padding: 10px 12px;
    border-radius: 16px;
    background: var(--surface-2);
    min-width: 0;
  }
  .meta a:hover {
    background: var(--surface-3);
  }
  .meta small {
    display: block;
    font-size: 0.68rem;
    font-weight: 750;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--text-2);
  }
  .meta b {
    display: block;
    font-size: 0.875rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  form {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .price-row {
    display: flex;
    gap: 10px;
  }
  .price-row .field {
    flex: 1;
  }
  .price-row .cur {
    flex: 0 0 110px;
  }
  .replace {
    align-self: flex-start;
  }
  .actions {
    display: flex;
    gap: 8px;
    align-items: center;
    flex-wrap: wrap;
  }

  @media (max-width: 860px) {
    .page-card {
      flex-direction: column;
    }
    .page-card .image img {
      max-height: 70vh;
    }
  }
  @media (max-width: 480px) {
    .body,
    .page-card .body {
      padding: 18px 16px 28px;
    }
    .meta {
      grid-template-columns: 1fr 1fr;
    }
  }
</style>
