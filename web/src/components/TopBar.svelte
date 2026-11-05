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
  // Tint the bar only once content scrolls under it; at the top the page
  // gradient should show through untouched.
  let scrolled = $state(false);

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

<svelte:window onscroll={() => (scrolled = window.scrollY > 8)} />

<div class="topbar" class:public={!session.user} class:scrolled>
  {#if session.user}
    <button class="btn icon menu-btn" aria-label="Open menu" onclick={onmenu}><Menu size={20} /></button>
    <a class="logo" href="/" aria-label="Cork home"><img src="/favicon.svg" alt="" width="32" height="32" /></a>
    <form class="search glass" role="search" onsubmit={search}>
      <Search size={18} />
      <input type="search" placeholder="Search your pins" bind:value={q} aria-label="Search your pins" />
    </form>
    <a class="btn primary create" href={createHref}><Plus size={18} /> <span>Create pin</span></a>
  {:else}
    <a class="logo full" href="/"><img src="/favicon.svg" alt="" width="34" height="34" /> <span>Cork</span></a>
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
    gap: 12px;
    padding: 18px 28px 14px 18px;
  }
  /* Blur what scrolls underneath instead of painting a colour over the page
     gradient (a solid fill shows up as a hard-edged band). */
  .topbar::before {
    content: '';
    position: absolute;
    inset: 0 0 -12px;
    z-index: -1;
    pointer-events: none;
    -webkit-backdrop-filter: blur(14px);
    backdrop-filter: blur(14px);
    -webkit-mask-image: linear-gradient(to bottom, #000 55%, transparent);
    mask-image: linear-gradient(to bottom, #000 55%, transparent);
    opacity: 0;
    transition: opacity 0.2s;
  }
  .topbar.scrolled::before {
    opacity: 1;
    background: color-mix(in srgb, var(--bg) 82%, transparent);
  }
  .topbar.public {
    padding-left: 28px;
  }
  .search {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 10px;
    height: 48px;
    padding: 0 18px;
    border-radius: 18px;
    color: var(--text-2);
    background: var(--glass-strong);
    box-shadow: var(--shadow-sm);
    min-width: 0;
  }
  .search:focus-within {
    box-shadow: 0 0 0 4px var(--focus);
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
    color: var(--text-3);
  }
  .create {
    height: 48px;
    border-radius: 18px;
    flex-shrink: 0;
  }
  .logo {
    display: none;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
    font-weight: 800;
    font-size: 1.25rem;
    letter-spacing: -0.03em;
  }
  .logo img {
    border-radius: 10px;
  }
  .logo.full {
    display: flex;
  }
  .menu-btn {
    display: none;
    flex-shrink: 0;
  }
  .grow {
    flex: 1;
  }
  :global([data-theme='dark']) .topbar :global(.sun),
  :global([data-theme='light']) .topbar :global(.moon) {
    display: none;
  }

  @media (max-width: 1023px) {
    .topbar {
      padding: 12px 20px;
    }
    .menu-btn,
    .logo {
      display: inline-flex;
    }
  }
  @media (max-width: 640px) {
    .topbar {
      padding: 10px 12px;
      gap: 8px;
    }
    .create span {
      display: none;
    }
    .create {
      width: 48px;
      padding: 0;
    }
    .logo:not(.full) {
      display: none;
    }
  }
</style>
