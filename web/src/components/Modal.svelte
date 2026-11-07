<script lang="ts">
  import type { Snippet } from 'svelte';
  import { onMount } from 'svelte';
  import X from '@lucide/svelte/icons/x';

  let {
    onclose,
    children,
    label,
    panel = false,
  }: {
    onclose: () => void;
    children: Snippet;
    label: string;
    /** Wide, viewport-height-capped dialog that scrolls inside itself (full screen on phones). */
    panel?: boolean;
  } = $props();

  let box: HTMLDivElement;

  onMount(() => {
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    const prevFocus = document.activeElement as HTMLElement | null;
    box.focus({ preventScroll: true });
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onclose();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => {
      document.body.style.overflow = prevOverflow;
      window.removeEventListener('keydown', onKey);
      prevFocus?.focus?.({ preventScroll: true });
    };
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="backdrop" class:sheet={panel} onclick={(e) => e.target === e.currentTarget && onclose()}>
  <div class="box" bind:this={box} role="dialog" aria-modal="true" aria-label={label} tabindex="-1">
    <button class="close" aria-label="Close" onclick={onclose}><X size={17} /></button>
    {@render children()}
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    overflow-y: auto;
    padding: 64px 16px;
  }
  /* only the dimmer fades; the dialog itself stays opaque */
  .backdrop::before {
    content: '';
    position: fixed;
    inset: 0;
    background: var(--scrim);
    -webkit-backdrop-filter: blur(4px);
    backdrop-filter: blur(4px);
    animation: fade 0.16s ease-out;
  }
  .box {
    position: relative;
    width: 100%;
    max-width: 480px;
    background: var(--surface);
    border-radius: var(--radius-xl);
    box-shadow: var(--shadow-lg);
    padding: 24px;
    animation: rise 0.2s cubic-bezier(0.2, 0.9, 0.25, 1);
    outline: none;
  }
  .close {
    position: absolute;
    top: 10px;
    right: 10px;
    z-index: 3;
    width: 30px;
    height: 30px;
    border-radius: 50%;
    border: none;
    display: grid;
    place-items: center;
    background: var(--surface-2);
    color: var(--text);
    cursor: pointer;
  }
  /* sits on top of the picture */
  .sheet .close {
    background: rgb(0 0 0 / 55%);
    color: #fff;
  }

  /* centered panel */
  .sheet {
    padding: 14px;
    align-items: center;
    overflow: hidden;
  }
  .sheet .box {
    max-width: 560px;
    max-height: 100%;
    padding: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
    /* a scrollbar here would cut into the picture and the rounded corners */
    scrollbar-width: none;
  }

  @keyframes fade {
    from {
      opacity: 0;
    }
  }
  @keyframes rise {
    from {
      opacity: 0;
      transform: scale(0.97);
    }
  }

  @media (max-width: 640px) {
    .backdrop {
      padding: 0;
      align-items: stretch;
    }
    .box,
    .sheet .box {
      max-width: none;
      border-radius: 0;
      min-height: 100%;
    }
    .backdrop:not(.sheet) .box {
      padding: 64px 20px 28px;
    }
    .sheet .box {
      animation: rise-full 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
    }
  }
  @keyframes rise-full {
    from {
      transform: translateY(40px);
      opacity: 0;
    }
  }
</style>
