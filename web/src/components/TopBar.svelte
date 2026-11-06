<script lang="ts">
  import Plus from '@lucide/svelte/icons/plus';
  import Search from '@lucide/svelte/icons/search';
  import Menu from '@lucide/svelte/icons/menu';
  import Sun from '@lucide/svelte/icons/sun';
  import Moon from '@lucide/svelte/icons/moon';
  import { navigate, router } from '../lib/router.svelte';
  import { session } from '../lib/session.svelte';
  import { setTheme } from '../lib/theme.svelte';

  let { onmenu }: { onmenu?: () => void } = $props();

  let q = $state(router.current.path === '/pins' ? (router.current.query.get('q') ?? '') : '');
  // The bar only gets its hairline once content scrolls under it.
  let scrolled = $state(false);
  let input: HTMLInputElement | undefined = $state();

  // "/" jumps to search unless the user is typing or a dialog is open.
  function onKey(e: KeyboardEvent) {
    if (e.key !== '/' || e.metaKey || e.ctrlKey || e.altKey || !input) return;
    const el = e.target as HTMLElement;
    if (el.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(el.tagName)) return;
    if (document.querySelector('[aria-modal="true"], dialog[open]')) return;
    e.preventDefault();
    input.focus();
    input.select();
  }

  $effect(() => {
    if (router.current.path === '/pins') q = router.current.query.get('q') ?? '';
  });

  function search(e: SubmitEvent) {
    e.preventDefault();
    const term = q.trim();
    navigate(term ? `/pins?q=${encodeURIComponent(term)}` : '/pins', { replace: router.current.path === '/pins' });
  }

  const createHref = $derived.by(() => {
    const p = router.current.bg ?? router.current.path;
    return p.startsWith('/b/') ? `/new?board=${p.slice(3)}` : '/new';
  });
</script>

<svelte:window onscroll={() => (scrolled = window.scrollY > 8)} onkeydown={onKey} />

<div class="topbar" class:public={!session.user} class:scrolled>
  {#if session.user}
    <button class="btn icon menu-btn" aria-label="Open menu" onclick={onmenu}><Menu size={20} /></button>
    <a class="logo" href="/" aria-label="Cork home"><img src="/favicon.svg" alt="" width="26" height="26" /></a>
    <form class="search" role="search" onsubmit={search}>
      <Search size={16} />
      <input bind:this={input} type="search" placeholder="Search your pins" bind:value={q} aria-label="Search your pins" />
      <kbd aria-hidden="true">/</kbd>
    </form>
    <a class="btn primary create" href={createHref}><Plus size={18} /> <span>Create pin</span></a>
  {:else}
    <a class="logo full" href="/"><img src="/favicon.svg" alt="" width="26" height="26" /> <span>Cork</span></a>
    <span class="grow"></span>
    <button
      class="btn icon"
      aria-label="Toggle dark mode"
      onclick={() => setTheme(document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark')}
    >
      <Sun size={18} class="sun" /><Moon size={18} class="moon" />
    </button>
    <a class="btn primary" href="/login?next={encodeURIComponent(router.current.path)}">Sign in</a>
  {/if}
</div>

<style>
  .topbar {
    position: sticky;
    top: 0;
    z-index: 40;
    display: flex;
    align-items: center;
    gap: 10px;
    height: var(--topbar-h);
    padding: 0 24px;
    background: var(--bg);
    transition: box-shadow 0.15s;
  }
  .topbar.scrolled {
    box-shadow: 0 1px 0 var(--border);
  }
  .search {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 34px;
    padding: 0 6px 0 10px;
    border-radius: var(--radius-sm);
    color: var(--text-2);
    background: var(--surface-2);
    min-width: 0;
    transition: box-shadow 0.12s;
  }
  .search:focus-within {
    box-shadow: 0 0 0 3px var(--focus);
  }
  .search input {
    flex: 1;
    min-width: 0;
    border: none;
    background: transparent;
    font: inherit;
    color: var(--text);
    outline: none;
  }
  .search input::placeholder {
    color: var(--text-2);
  }
  kbd {
    width: 20px;
    height: 20px;
    display: grid;
    place-items: center;
    flex-shrink: 0;
    border-radius: 5px;
    box-shadow: inset 0 0 0 1px var(--border);
    font: inherit;
    font-size: 0.72rem;
  }
  .search:focus-within kbd {
    display: none;
  }
  .create {
    flex-shrink: 0;
  }
  .logo {
    display: none;
    align-items: center;
    gap: 9px;
    flex-shrink: 0;
    font-weight: 650;
    font-size: 1rem;
    letter-spacing: -0.015em;
  }
  .logo.full {
    display: flex;
  }
  .menu-btn {
    display: none;
    flex-shrink: 0;
    background: none;
  }
  .grow {
    flex: 1;
  }
  :global([data-theme='dark']) .topbar :global(.sun),
  :global([data-theme='light']) .topbar :global(.moon) {
    display: none;
  }

  @media (hover: none) {
    kbd {
      display: none;
    }
  }
  @media (max-width: 1023px) {
    .topbar {
      padding: 0 14px;
    }
    .menu-btn,
    .logo {
      display: inline-flex;
    }
  }
  @media (max-width: 640px) {
    .topbar {
      padding: 0 10px;
      gap: 8px;
    }
    .create span {
      display: none;
    }
    .create {
      aspect-ratio: 1;
      padding: 0;
    }
    .logo:not(.full) {
      display: none;
    }
  }
</style>
