<script lang="ts">
  import Pin from '@lucide/svelte/icons/pin';
  import { bookmarkletHref } from '../lib/bookmarklet';

  let { compact = false }: { compact?: boolean } = $props();

  let link: HTMLAnchorElement;

  // Svelte won't render javascript: URLs through attributes, so set it directly.
  $effect(() => {
    link.setAttribute('href', bookmarkletHref(location.origin));
  });
</script>

<div class="bookmarklet" class:compact>
  <!-- svelte-ignore a11y_invalid_attribute -->
  <a
    bind:this={link}
    href="#"
    class="btn primary"
    onclick={(e) => {
      e.preventDefault();
      alert('Drag this button to your bookmarks bar, then click it on any page to pin from it.');
    }}
    title="Drag me to your bookmarks bar"
  >
    <Pin size={16} /> Pin to Cork
  </a>
  {#if !compact}
    <p class="muted">
      Drag this button to your browser's bookmarks bar. On any website, click it to open Cork with that page ready to
      pin. It fills in the link and title, and on shopping sites it saves the product's current price too.
    </p>
  {/if}
</div>

<style>
  .bookmarklet {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  .bookmarklet a {
    cursor: grab;
  }
  p {
    font-size: 0.875rem;
  }
</style>
