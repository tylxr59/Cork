<script lang="ts">
  // Pin grid. Uses CSS grid with tiny rows and per-item row spans so items
  // stay in left-to-right reading order (which keeps drag-to-reorder intuitive),
  // while still packing like a masonry layout.
  import Sortable from 'sortablejs';
  import type { Pin } from '../lib/types';
  import PinCard from './PinCard.svelte';

  let {
    pins,
    sortable = false,
    showBoard = false,
    onreorder,
  }: {
    pins: Pin[];
    sortable?: boolean;
    showBoard?: boolean;
    onreorder?: (pins: Pin[]) => void;
  } = $props();

  const ROW = 4; // px; must match grid-auto-rows
  const GAP = 18; // vertical gap between pins

  let grid: HTMLDivElement;

  // Sizes a cell to its content. Measures right away (so the first layout
  // doesn't depend on when ResizeObserver delivers), then on any resize.
  function span(node: HTMLElement) {
    const inner = node.firstElementChild as HTMLElement;
    const update = () => {
      const h = inner.getBoundingClientRect().height;
      if (h > 0) node.style.gridRowEnd = `span ${Math.ceil((h + GAP) / ROW)}`;
    };
    update();
    const ro = new ResizeObserver(update);
    ro.observe(inner);
    inner.addEventListener('load', update, true); // image loads (capture: load doesn't bubble)
    return {
      destroy() {
        ro.disconnect();
        inner.removeEventListener('load', update, true);
      },
    };
  }

  $effect(() => {
    if (!sortable) return;
    const s = Sortable.create(grid, {
      animation: 180,
      draggable: '.cell',
      delay: 180,
      delayOnTouchOnly: true,
      ghostClass: 'sortable-ghost',
      chosenClass: 'sortable-chosen',
      dragClass: 'sortable-drag',
      onEnd(evt) {
        const { oldDraggableIndex: from, newDraggableIndex: to, item } = evt;
        if (from === undefined || to === undefined || from === to) return;
        // Put the DOM back where Svelte expects it, then let Svelte re-render
        // from the reordered array.
        item.remove();
        grid.insertBefore(item, grid.children[from] ?? null);
        const next = [...pins];
        const [moved] = next.splice(from, 1);
        next.splice(to, 0, moved);
        onreorder?.(next);
      },
    });
    return () => s.destroy();
  });
</script>

<div class="grid" class:sortable bind:this={grid} style:--row="{ROW}px">
  {#each pins as pin (pin.id)}
    <div class="cell" use:span>
      <div class="inner">
        <PinCard {pin} {showBoard} />
      </div>
    </div>
  {/each}
</div>

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(min(230px, calc(50% - 9px)), 1fr));
    grid-auto-rows: var(--row);
    column-gap: 18px;
    align-items: start;
  }
  .sortable :global(.cell) {
    cursor: grab;
  }
  @media (max-width: 520px) {
    .grid {
      column-gap: 10px;
    }
  }
</style>
