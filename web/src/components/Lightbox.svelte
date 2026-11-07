<script lang="ts">
  import { onMount, tick } from 'svelte';
  import X from '@lucide/svelte/icons/x';

  let { src, alt, onclose }: { src: string; alt: string; onclose: () => void } = $props();

  let dialog: HTMLDialogElement;
  let img: HTMLImageElement;
  let natural = $state({ w: 0, h: 0 });
  let innerWidth = $state(window.innerWidth);
  let innerHeight = $state(window.innerHeight);
  let zoomed = $state(false);
  let panning = $state(false);

  // Mouse drag panning. A drag ends in a click, which mustn't also zoom or close.
  let drag: { x: number; y: number; left: number; top: number } | null = null;
  let dragged = false;

  // Matches the dialog's padding below.
  const pad = $derived(innerWidth <= 640 ? 8 : 24);
  const fit = $derived(
    natural.w ? Math.min((innerWidth - pad * 2) / natural.w, (innerHeight - pad * 2) / natural.h) : 1,
  );
  // The first view is the smaller of fit-to-screen and original size; the second is the larger.
  const scale = $derived(zoomed ? Math.max(fit, 1) : Math.min(fit, 1));
  const canZoom = $derived(Math.abs(fit - 1) > 0.01);

  function measure() {
    natural = { w: img.naturalWidth, h: img.naturalHeight };
  }

  onMount(() => {
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    dialog.showModal();
    if (img.complete) measure();
    return () => {
      document.body.style.overflow = prevOverflow;
    };
  });

  async function toggleZoom(e: MouseEvent) {
    // keep the clicked part of the picture under the pointer
    const before = img.getBoundingClientRect();
    const fx = (e.clientX - before.left) / before.width;
    const fy = (e.clientY - before.top) / before.height;
    zoomed = !zoomed;
    await tick();
    const after = img.getBoundingClientRect();
    dialog.scrollLeft += after.left + fx * after.width - e.clientX;
    dialog.scrollTop += after.top + fy * after.height - e.clientY;
  }

  function onclick(e: MouseEvent) {
    if (dragged) return;
    if (e.target === img && canZoom) toggleZoom(e);
    else dialog.close();
  }

  function onpointerdown(e: PointerEvent) {
    dragged = false;
    const scrolls = dialog.scrollWidth > dialog.clientWidth || dialog.scrollHeight > dialog.clientHeight;
    if (e.pointerType === 'mouse' && e.target === img && scrolls) {
      drag = { x: e.clientX, y: e.clientY, left: dialog.scrollLeft, top: dialog.scrollTop };
    }
  }

  function onpointermove(e: PointerEvent) {
    if (!drag) return;
    const dx = e.clientX - drag.x;
    const dy = e.clientY - drag.y;
    if (!dragged && Math.hypot(dx, dy) < 4) return;
    dragged = panning = true;
    dialog.scrollLeft = drag.left - dx;
    dialog.scrollTop = drag.top - dy;
  }

  function endDrag() {
    drag = null;
    panning = false;
  }
</script>

<svelte:window bind:innerWidth bind:innerHeight />

<!-- The native dialog handles Escape and focus; stopping keydown here keeps Escape from also closing the pin Modal behind it. -->
<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
<dialog
  bind:this={dialog}
  aria-label="Expanded image"
  class:panning
  {onclose}
  {onclick}
  {onpointerdown}
  {onpointermove}
  onpointerup={endDrag}
  onpointercancel={endDrag}
  onkeydown={(e) => e.key === 'Escape' && e.stopPropagation()}
>
  <button class="close" aria-label="Close"><X size={20} /></button>
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <img
    bind:this={img}
    {src}
    {alt}
    draggable="false"
    class:zoom-in={canZoom && !zoomed}
    class:zoom-out={canZoom && zoomed}
    style:width={natural.w ? `${natural.w * scale}px` : undefined}
    style:height={natural.w ? `${natural.h * scale}px` : undefined}
    onload={measure}
  />
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
    overflow: auto;
    overscroll-behavior: contain;
    scrollbar-width: none;
    outline: none;
    animation: fade 0.18s ease-out;
  }
  dialog[open] {
    display: flex;
  }
  dialog::backdrop {
    background: rgba(0, 0, 0, 0.88);
    animation: fade 0.18s ease-out;
  }
  /* auto margins center it, but unlike centering keep an oversized picture's edges reachable */
  img {
    display: block;
    flex-shrink: 0;
    max-width: none;
    margin: auto;
    user-select: none;
  }
  .zoom-in {
    cursor: zoom-in;
  }
  .zoom-out {
    cursor: zoom-out;
  }
  .panning,
  .panning img {
    cursor: grabbing;
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
