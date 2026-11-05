<script lang="ts">
  import ImagePlus from '@lucide/svelte/icons/image-plus';
  import Globe from '@lucide/svelte/icons/globe';
  import X from '@lucide/svelte/icons/x';
  import Check from '@lucide/svelte/icons/check';
  import { api } from '../lib/api';
  import { domain } from '../lib/format';
  import { currencyHints, parsePrice, tidyPrice } from '../lib/currency';
  import { navigate, router } from '../lib/router.svelte';
  import { emitPin, loadBoards, myBoards, toast, toastError } from '../lib/session.svelte';
  import type { Board, ScrapedPage } from '../lib/types';
  import BoardForm from '../components/BoardForm.svelte';
  import Bookmarklet from '../components/Bookmarklet.svelte';

  const LAST_BOARD = 'cork:last-board';
  const params = router.current.query;
  const popup = params.get('popup') === '1';

  // image source: uploaded files, or remote image URLs
  let files = $state<{ file: File; preview: string }[]>([]);
  // images ticked in the bookmarklet, which arrive in the URL hash. Bookmarklets
  // dragged out before the picker existed send a single ?image= instead.
  const pickedImages = new URLSearchParams(location.hash.slice(1)).getAll('image');
  const legacyImage = params.get('image');
  let imageUrls = $state(pickedImages.length ? pickedImages : legacyImage ? [legacyImage] : []);

  let pageUrl = $state(params.get('url') ?? '');
  let scraped = $state<ScrapedPage | null>(null);
  let scraping = $state(false);
  let scrapeError = $state('');
  let hiddenCandidates = $state<Record<string, boolean>>({});

  let boardId = $state(params.get('board') ?? '');
  let title = $state(params.get('title') ?? '');
  let description = $state('');
  let sourceUrl = $state(params.get('url') ?? '');
  // price as seen on the product page right now (from the bookmarklet or the scraper).
  // The bookmarklet may send raw page text like "$20.23"; the page's language and
  // the site's country help decide which dollar (or yen, or krona) that is.
  const pageLang = params.get('lang') ?? '';
  const fromPage = parsePrice(params.get('price') ?? '', params.get('currency') ?? '', currencyHints(params.get('url') ?? '', pageLang));
  let price = $state(fromPage.amount);
  let currency = $state(fromPage.currency);
  // used when a price has no currency: the site's, else the user's own
  const fallbackCurrency = $derived(currencyHints(sourceUrl, pageLang)[0]);

  function cleanPrice() {
    const t = tidyPrice(price, currency, currencyHints(sourceUrl, pageLang));
    if (t) ({ price, currency } = t);
  }

  let creatingBoard = $state(false);
  let saving = $state(false);
  let progress = $state('');
  let saved = $state(false);
  let dragging = $state(false);
  let fileInput: HTMLInputElement | undefined = $state();

  loadBoards(true)
    .then((list) => {
      if (!boardId || !list.some((b) => b.id === boardId)) {
        const last = localStorage.getItem(LAST_BOARD);
        boardId = list.find((b) => b.id === last)?.id ?? list[0]?.id ?? '';
      }
    })
    .catch(toastError);

  // images were already chosen on the page, so only look for its description
  if (params.get('url')) scrape(pickedImages.length > 0);

  const hasImage = $derived(files.length > 0 || imageUrls.length > 0);
  const pinCount = $derived(imageUrls.length || files.length);
  const candidates = $derived.by(() => {
    const list = scraped?.images ?? [];
    const all = legacyImage && !list.includes(legacyImage) ? [legacyImage, ...list] : list;
    return all.filter((u) => !hiddenCandidates[u]);
  });

  function addFiles(list: FileList | File[]) {
    const imgs = [...list].filter((f) => f.type.startsWith('image/'));
    if (imgs.length === 0) {
      toast('That doesn’t look like an image', 'error');
      return;
    }
    imageUrls = [];
    files = [...files, ...imgs.map((file) => ({ file, preview: URL.createObjectURL(file) }))];
  }

  function removeFile(i: number) {
    URL.revokeObjectURL(files[i].preview);
    files = files.filter((_, j) => j !== i);
  }

  function clearImage() {
    files.forEach((f) => URL.revokeObjectURL(f.preview));
    files = [];
    imageUrls = [];
  }

  async function scrape(detailsOnly = false) {
    const url = pageUrl.trim();
    if (!url) return;
    scraping = true;
    scrapeError = '';
    scraped = null;
    hiddenCandidates = {};
    try {
      const page = await api.scrape(url);
      if (!detailsOnly) scraped = page;
      if (!sourceUrl) sourceUrl = page.url;
      if (!title) title = page.title;
      if (!description) description = page.description;
      if (!price && page.price) {
        price = page.price;
        currency = page.currency;
      }
      // a direct image link: select it straight away
      if (!imageUrls.length && page.images.length === 1 && page.images[0] === page.url) pick(page.url);
    } catch (e) {
      if (!detailsOnly) scrapeError = (e as Error).message;
    } finally {
      scraping = false;
    }
  }

  function pick(url: string) {
    if (files.length > 0) clearImage();
    imageUrls = imageUrls.includes(url) ? imageUrls.filter((u) => u !== url) : [...imageUrls, url];
  }

  function onCandidateLoad(e: Event, url: string) {
    const img = e.currentTarget as HTMLImageElement;
    // hide icons, spacers and tracking pixels
    if (img.naturalWidth < 120 || img.naturalHeight < 80) hiddenCandidates[url] = true;
  }

  function onPaste(e: ClipboardEvent) {
    const items = e.clipboardData?.files;
    if (items && items.length > 0) {
      e.preventDefault();
      addFiles(items);
      return;
    }
    const target = e.target as HTMLElement;
    if (target.closest('input, textarea')) return;
    const text = e.clipboardData?.getData('text')?.trim() ?? '';
    if (/^https?:\/\//i.test(text)) {
      pageUrl = text;
      scrape();
    }
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    dragging = false;
    const dt = e.dataTransfer;
    if (!dt) return;
    if (dt.files.length > 0) {
      addFiles(dt.files);
      return;
    }
    const url = (dt.getData('text/uri-list') || dt.getData('text/plain')).split('\n')[0].trim();
    if (/^https?:\/\//i.test(url)) {
      pageUrl = url;
      scrape();
    }
  }

  function boardCreated(b: Board) {
    creatingBoard = false;
    boardId = b.id;
  }

  async function save(e: SubmitEvent) {
    e.preventDefault();
    if (!hasImage || !boardId) return;
    saving = true;
    localStorage.setItem(LAST_BOARD, boardId);
    cleanPrice();
    if (price.trim() && !currency.trim()) currency = fallbackCurrency;
    const base = () => {
      const fd = new FormData();
      fd.append('board_id', boardId);
      fd.append('title', title);
      fd.append('description', description);
      fd.append('source_url', sourceUrl.trim());
      fd.append('price', price.trim());
      fd.append('currency', currency.trim());
      return fd;
    };
    try {
      if (imageUrls.length > 0) {
        const urls = [...imageUrls];
        for (let i = 0; i < urls.length; i++) {
          if (urls.length > 1) progress = `Saving ${i + 1} of ${urls.length}…`;
          const fd = base();
          fd.append('image_url', urls[i]);
          emitPin({ type: 'created', pin: await api.createPin(fd) });
          // if a later image fails, retrying must not pin this one twice
          imageUrls = imageUrls.filter((u) => u !== urls[i]);
        }
      } else {
        for (let i = 0; i < files.length; i++) {
          if (files.length > 1) progress = `Uploading ${i + 1} of ${files.length}…`;
          const fd = base();
          fd.append('image', files[i].file);
          emitPin({ type: 'created', pin: await api.createPin(fd) });
        }
      }
      const board = myBoards.list.find((b) => b.id === boardId);
      if (popup) {
        saved = true;
        setTimeout(() => window.close(), 1200);
      } else {
        toast(`Saved to ${board?.name ?? 'board'}`);
        clearImage();
        navigate(`/b/${boardId}`);
      }
    } catch (err) {
      toastError(err);
    } finally {
      saving = false;
      progress = '';
    }
  }
</script>

<svelte:window onpaste={onPaste} />
<svelte:head><title>Create pin · Cork</title></svelte:head>

<div class="page">
  {#if saved}
    <div class="empty">
      <Check size={48} />
      <h2>Pinned!</h2>
      <button class="btn" onclick={() => window.close()}>Close window</button>
    </div>
  {:else}
    <h1 class="page-title">Create pin</h1>

    <form class="layout" onsubmit={save}>
      <section class="left">
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="drop"
          class:dragging
          class:has={hasImage}
          ondragover={(e) => {
            e.preventDefault();
            dragging = true;
          }}
          ondragleave={() => (dragging = false)}
          ondrop={onDrop}
        >
          {#if imageUrls.length === 1}
            <div class="preview">
              <img src={imageUrls[0]} alt="Selected" referrerpolicy="no-referrer" />
              <button type="button" class="btn icon small remove" aria-label="Remove image" onclick={clearImage}>
                <X size={16} />
              </button>
            </div>
          {:else if imageUrls.length > 1}
            <div class="multi">
              {#each imageUrls as url (url)}
                <div class="thumb">
                  <img src={url} alt="" referrerpolicy="no-referrer" />
                  <button type="button" class="btn icon small remove" aria-label="Remove" onclick={() => pick(url)}>
                    <X size={14} />
                  </button>
                </div>
              {/each}
            </div>
            <p class="muted small">{imageUrls.length} images — each becomes its own pin with the details on the right.</p>
          {:else if files.length === 1}
            <div class="preview">
              <img src={files[0].preview} alt="Selected" />
              <button type="button" class="btn icon small remove" aria-label="Remove image" onclick={clearImage}>
                <X size={16} />
              </button>
            </div>
          {:else if files.length > 1}
            <div class="multi">
              {#each files as f, i (f.preview)}
                <div class="thumb">
                  <img src={f.preview} alt="" />
                  <button type="button" class="btn icon small remove" aria-label="Remove" onclick={() => removeFile(i)}>
                    <X size={14} />
                  </button>
                </div>
              {/each}
            </div>
            <p class="muted small">{files.length} images — each becomes its own pin with the details on the right.</p>
          {:else}
            <button type="button" class="pick" onclick={() => fileInput?.click()}>
              <ImagePlus size={40} />
              <strong>Choose images or drag them here</strong>
              <span class="muted">You can also paste an image or a link anywhere on this page</span>
              <span class="muted small">JPEG, PNG, GIF or WebP</span>
            </button>
          {/if}
          <input
            bind:this={fileInput}
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            multiple
            hidden
            onchange={(e) => {
              if (e.currentTarget.files) addFiles(e.currentTarget.files);
              e.currentTarget.value = '';
            }}
          />
        </div>
        {#if files.length > 0}
          <button type="button" class="btn small" onclick={() => fileInput?.click()}>Add more images</button>
        {/if}

        <div class="from-url">
          <label class="field">
            <span>Or save from a website</span>
            <div class="url-row">
              <input
                class="input"
                type="url"
                placeholder="https://example.com/some-page"
                bind:value={pageUrl}
                onkeydown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    scrape();
                  }
                }}
              />
              <button type="button" class="btn" disabled={scraping || !pageUrl.trim()} onclick={() => scrape()}>
                <Globe size={16} />
                {scraping ? 'Fetching…' : 'Fetch'}
              </button>
            </div>
          </label>
          {#if scrapeError}<p class="error-text">{scrapeError}</p>{/if}
          {#if scraped}
            {#if candidates.length === 0}
              <p class="muted small">No usable images found on {domain(scraped.url)}.</p>
            {:else}
              <p class="muted small">Pick images from {domain(scraped.url)}:</p>
              <div class="candidates">
                {#each candidates as url (url)}
                  <button type="button" class="cand" class:selected={imageUrls.includes(url)} onclick={() => pick(url)}>
                    <img
                      src={url}
                      alt=""
                      loading="lazy"
                      referrerpolicy="no-referrer"
                      onload={(e) => onCandidateLoad(e, url)}
                      onerror={() => (hiddenCandidates[url] = true)}
                    />
                    {#if imageUrls.includes(url)}<span class="tick"><Check size={16} /></span>{/if}
                  </button>
                {/each}
              </div>
            {/if}
          {/if}
        </div>
      </section>

      <section class="right glass panel">
        <label class="field">
          <span>Board</span>
          <div class="url-row">
            <select class="select" bind:value={boardId} required>
              {#if myBoards.list.length === 0}<option value="" disabled>Create a board first</option>{/if}
              {#each myBoards.list as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
            </select>
            <button type="button" class="btn" onclick={() => (creatingBoard = true)}>New</button>
          </div>
        </label>
        <label class="field">
          <span>Title</span>
          <input class="input" bind:value={title} maxlength="300" placeholder="Add a title" />
        </label>
        <label class="field">
          <span>Description</span>
          <textarea class="textarea" rows="5" bind:value={description} placeholder="Notes, recipe, measurements…"></textarea>
        </label>
        <label class="field">
          <span>Link</span>
          <input class="input" type="url" bind:value={sourceUrl} placeholder="Where is this from? https://…" />
        </label>
        <div class="price-row">
          <label class="field">
            <span>Price</span>
            <input class="input" bind:value={price} onblur={cleanPrice} placeholder="Optional, e.g. 24.99" />
          </label>
          <label class="field cur">
            <span>Currency</span>
            <input class="input" bind:value={currency} maxlength="4" placeholder={fallbackCurrency} title="A code like USD, or a symbol" />
          </label>
        </div>
        {#if price && scraped?.price}<p class="hint muted">Price found on {domain(scraped.url)}, saved as it is today.</p>{/if}
        <button class="btn primary save" disabled={saving || !hasImage || !boardId}>
          {progress || (saving ? 'Saving…' : pinCount > 1 ? `Save ${pinCount} pins` : 'Save pin')}
        </button>

        {#if !popup}
          <details class="tip">
            <summary>Save from any site with the bookmarklet</summary>
            <Bookmarklet />
          </details>
        {/if}
      </section>
    </form>
  {/if}
</div>

{#if creatingBoard}
  <BoardForm onclose={() => (creatingBoard = false)} onsaved={boardCreated} />
{/if}

<style>
  .layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(280px, 400px);
    gap: 32px;
    align-items: start;
  }
  .left,
  .right {
    display: flex;
    flex-direction: column;
    gap: 18px;
  }
  .right {
    position: sticky;
    top: calc(var(--topbar-h) + 8px);
    padding: 22px;
  }
  .price-row {
    display: flex;
    gap: 10px;
  }
  .price-row .field {
    flex: 1;
    min-width: 0;
  }
  .price-row .cur {
    flex: 0 0 104px;
  }
  .hint {
    font-size: 0.8rem;
    margin-top: -8px;
  }
  .drop {
    border: 2px dashed var(--border);
    border-radius: var(--radius-xl);
    background: var(--glass);
    min-height: 320px;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    justify-content: center;
    transition:
      border-color 0.15s,
      background 0.15s;
  }
  .drop.dragging {
    border-color: var(--accent);
    background: var(--accent-soft);
  }
  .pick :global(svg) {
    color: var(--accent);
  }
  .drop.has {
    border-style: solid;
    padding: 12px;
  }
  .pick {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 40px 20px;
    border: none;
    background: none;
    color: var(--text);
    font: inherit;
    cursor: pointer;
    text-align: center;
  }
  .small {
    font-size: 0.82rem;
  }
  .preview {
    position: relative;
    align-self: center;
  }
  .preview img {
    max-height: 60vh;
    border-radius: var(--radius);
  }
  .remove {
    position: absolute;
    top: 8px;
    right: 8px;
    background: var(--surface);
    box-shadow: var(--shadow);
  }
  .multi {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
    gap: 8px;
  }
  .thumb {
    position: relative;
    aspect-ratio: 1;
  }
  .thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: var(--radius-sm);
  }
  .thumb .remove {
    top: 4px;
    right: 4px;
    width: 26px;
    height: 26px;
  }
  .multi + p {
    margin-top: 10px;
  }
  .url-row {
    display: flex;
    gap: 8px;
  }
  .url-row .input,
  .url-row .select {
    flex: 1;
    min-width: 0;
  }
  .from-url {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .candidates {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    gap: 8px;
  }
  .cand {
    position: relative;
    aspect-ratio: 1;
    padding: 0;
    border: 3px solid transparent;
    border-radius: var(--radius-sm);
    overflow: hidden;
    background: var(--surface-2);
    cursor: pointer;
  }
  .cand.selected {
    border-color: var(--accent);
  }
  .cand img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  .tick {
    position: absolute;
    top: 6px;
    right: 6px;
    width: 26px;
    height: 26px;
    border-radius: 50%;
    background: var(--accent);
    color: var(--accent-text);
    display: grid;
    place-items: center;
  }
  .save {
    height: 48px;
    font-size: 1rem;
  }
  .tip summary {
    cursor: pointer;
    font-weight: 600;
    font-size: 0.9rem;
    color: var(--text-2);
    margin-bottom: 12px;
  }
  @media (max-width: 820px) {
    .layout {
      grid-template-columns: 1fr;
    }
    .right {
      position: static;
    }
  }
</style>
