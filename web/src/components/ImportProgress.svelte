<script lang="ts">
  import X from '@lucide/svelte/icons/x';
  import { cancelImport, dismissImport, imports, retryFailed } from '../lib/imports.svelte';
  import { plural } from '../lib/format';

  const settled = $derived(imports.done + imports.failed.length);
  const active = $derived(settled < imports.total);
</script>

{#if imports.total > 0}
  <div class="import" role="status">
    <div class="head">
      <strong>
        {#if active}Importing {settled + 1} of {imports.total}…{:else}Imported {imports.done} of {imports.total}{/if}
      </strong>
      {#if active}
        <button class="btn small" onclick={cancelImport}>Cancel</button>
      {:else}
        <button class="btn icon small" aria-label="Dismiss" onclick={dismissImport}><X size={16} /></button>
      {/if}
    </div>
    <progress max={imports.total} value={settled}></progress>
    {#if imports.failed.length > 0}
      <details>
        <summary class="error-text">{plural(imports.failed.length, 'image')} failed</summary>
        <ul>
          {#each imports.failed as f, i (i)}
            <li><span class="name">{f.file.name}</span> <span class="muted">{f.error}</span></li>
          {/each}
        </ul>
      </details>
      {#if !active}<button class="btn small" onclick={retryFailed}>Retry failed</button>{/if}
    {/if}
  </div>
{/if}

<style>
  .import {
    position: fixed;
    right: 24px;
    bottom: 24px;
    z-index: 150;
    width: min(340px, calc(100vw - 32px));
    padding: 14px 16px;
    border-radius: var(--radius-lg);
    background: var(--surface);
    box-shadow: var(--shadow);
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  progress {
    width: 100%;
    height: 6px;
    accent-color: var(--accent);
  }
  summary {
    cursor: pointer;
    font-size: 0.875rem;
  }
  ul {
    margin: 8px 0 0;
    padding: 0;
    list-style: none;
    max-height: 180px;
    overflow-y: auto;
    font-size: 0.82rem;
  }
  li {
    padding: 3px 0;
    overflow-wrap: anywhere;
  }
  .name {
    font-weight: 600;
  }
  @media (max-width: 640px) {
    .import {
      right: 16px;
      bottom: 16px;
    }
  }
</style>
