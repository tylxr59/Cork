<script lang="ts">
  import Plus from '@lucide/svelte/icons/plus';
  import { api } from '../lib/api';
  import { router } from '../lib/router.svelte';
  import { onPinEvent, toastError } from '../lib/session.svelte';
  import type { Pin } from '../lib/types';
  import Masonry from '../components/Masonry.svelte';
  import PinSizeControl from '../components/PinSizeControl.svelte';

  let pins = $state<Pin[]>([]);
  let hasMore = $state(false);
  let loading = $state(false);
  let sentinel: HTMLDivElement | undefined = $state();
  let generation = 0;

  const q = $derived(router.current.query.get('q') ?? '');

  async function load(reset: boolean) {
    const gen = reset ? ++generation : generation;
    loading = true;
    try {
      const res = await api.pins(q, reset ? 0 : pins.length);
      if (gen !== generation) return; // a newer search started
      pins = reset ? res.pins : [...pins, ...res.pins];
      hasMore = res.has_more;
    } catch (e) {
      toastError(e);
    } finally {
      if (gen === generation) loading = false;
    }
  }

  $effect(() => {
    q;
    load(true);
  });

  $effect(() => {
    if (!sentinel) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loading) load(false);
      },
      { rootMargin: '800px' },
    );
    io.observe(sentinel);
    return () => io.disconnect();
  });

  $effect(() =>
    onPinEvent((e) => {
      if (e.type === 'deleted') pins = pins.filter((p) => p.id !== e.id);
      if (e.type === 'updated') pins = pins.map((p) => (p.id === e.pin.id ? e.pin : p));
    }),
  );
</script>

<div class="page">
  <div class="head">
    <h1>{q ? `Results for “${q}”` : 'All pins'}</h1>
    <PinSizeControl />
  </div>

  {#if pins.length === 0 && loading}
    <div class="spinner"></div>
  {:else if pins.length === 0}
    <div class="empty">
      {#if q}
        <h2>No pins match “{q}”</h2>
        <p>Search looks at titles, descriptions, links and board names.</p>
      {:else}
        <h2>Nothing pinned yet</h2>
        <a class="btn primary" href="/new"><Plus size={18} /> Create a pin</a>
      {/if}
    </div>
  {:else}
    <Masonry {pins} showBoard canEdit />
    {#if loading}<div class="spinner"></div>{/if}
  {/if}
  <div bind:this={sentinel}></div>
</div>

<style>
  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    margin: 0 0 20px;
  }
</style>
