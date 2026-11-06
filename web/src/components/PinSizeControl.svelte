<script lang="ts">
  import { PIN_SIZES, pinSize, setPinSize } from '../lib/pinsize.svelte';
</script>

<div class="sizes" role="radiogroup" aria-label="Pin size">
  {#each PIN_SIZES as s (s.value)}
    <button
      type="button"
      role="radio"
      aria-checked={pinSize.value === s.value}
      aria-label={s.name}
      title={s.name}
      class:on={pinSize.value === s.value}
      onclick={() => setPinSize(s.value)}
    >
      {s.label}
    </button>
  {/each}
</div>

<style>
  .sizes {
    display: inline-flex;
    flex-shrink: 0;
    padding: 2px;
    border-radius: var(--radius-sm);
    background: var(--surface-2);
  }
  button {
    width: 30px;
    height: 30px;
    border: none;
    border-radius: 6px;
    background: none;
    color: var(--text-2);
    font: inherit;
    font-size: 0.75rem;
    font-weight: 550;
    cursor: pointer;
  }
  button.on {
    background: var(--surface);
    color: var(--text);
    box-shadow: var(--shadow-sm);
  }
  /* phones always get two columns, so there's nothing to choose */
  @media (max-width: 640px) {
    .sizes {
      display: none;
    }
  }
</style>
