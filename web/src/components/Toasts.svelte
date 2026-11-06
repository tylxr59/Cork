<script lang="ts">
  import { toasts } from '../lib/session.svelte';
</script>

<div class="toasts" aria-live="polite">
  {#each toasts.list as t (t.id)}
    <div class="toast" class:error={t.kind === 'error'}>{t.text}</div>
  {/each}
</div>

<style>
  .toasts {
    position: fixed;
    bottom: 24px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 200;
    display: flex;
    flex-direction: column;
    gap: 8px;
    align-items: center;
    pointer-events: none;
  }
  .toast {
    padding: 9px 14px;
    border-radius: var(--radius);
    background: var(--text);
    color: var(--bg);
    font-weight: 550;
    box-shadow: var(--shadow);
    animation: pop 0.2s ease-out;
  }
  .toast.error {
    background: var(--danger);
    color: #fff;
  }
  @keyframes pop {
    from {
      opacity: 0;
      transform: translateY(8px);
    }
  }
</style>
