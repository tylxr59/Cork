<script lang="ts">
  import { onMount } from 'svelte';
  import X from '@lucide/svelte/icons/x';

  let { src, alt, onclose }: { src: string; alt: string; onclose: () => void } = $props();

  let dialog: HTMLDialogElement;

  onMount(() => {
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    dialog.showModal();
    return () => {
      document.body.style.overflow = prevOverflow;
    };
  });
</script>

<!-- The native dialog handles Escape and focus; stopping keydown here keeps Escape from also closing the pin Modal behind it. -->
<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
<dialog
  bind:this={dialog}
  aria-label="Expanded image"
  {onclose}
  onclick={() => dialog.close()}
  onkeydown={(e) => e.key === 'Escape' && e.stopPropagation()}
>
  <button class="close" aria-label="Close"><X size={20} /></button>
  <img {src} {alt} />
</dialog>

<style>
  dialog {
    width: 100vw;
    height: 100dvh;
    max-width: none;
    max-height: none;
    margin: 0;
    padding: 24px;
    border: none;
    background: none;
    cursor: zoom-out;
    outline: none;
    animation: fade 0.18s ease-out;
  }
  dialog::backdrop {
    background: rgba(0, 0, 0, 0.88);
    animation: fade 0.18s ease-out;
  }
  img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: contain;
  }
  .close {
    position: fixed;
    top: 14px;
    right: 14px;
    width: 34px;
    height: 34px;
    border-radius: 50%;
    border: none;
    display: grid;
    place-items: center;
    background: rgb(255 255 255 / 16%);
    color: #fff;
    cursor: pointer;
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
  @media (max-width: 640px) {
    dialog {
      padding: 8px;
    }
  }
</style>
