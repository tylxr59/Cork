<script lang="ts">
  import LayoutGrid from '@lucide/svelte/icons/layout-grid';
  import Images from '@lucide/svelte/icons/images';
  import Settings from '@lucide/svelte/icons/settings';
  import Shield from '@lucide/svelte/icons/shield';
  import Plus from '@lucide/svelte/icons/plus';
  import Lock from '@lucide/svelte/icons/lock';
  import LogOut from '@lucide/svelte/icons/log-out';
  import ChevronUp from '@lucide/svelte/icons/chevron-up';
  import Sun from '@lucide/svelte/icons/sun';
  import Moon from '@lucide/svelte/icons/moon';
  import Monitor from '@lucide/svelte/icons/monitor';
  import { api } from '../lib/api';
  import { navigate, router } from '../lib/router.svelte';
  import { myBoards, session } from '../lib/session.svelte';
  import { setTheme, theme, type ThemePref } from '../lib/theme.svelte';
  import BoardForm from './BoardForm.svelte';

  let { open = false, onclose }: { open?: boolean; onclose: () => void } = $props();

  let menuOpen = $state(false);
  let creating = $state(false);

  const path = $derived(router.current.bg ?? router.current.path);
  const activeBoard = $derived(path.startsWith('/b/') ? path.slice(3).split(/[?#]/)[0] : '');

  // close the drawer whenever the route changes
  $effect(() => {
    router.current.path;
    onclose();
    menuOpen = false;
  });

  async function logout() {
    menuOpen = false;
    await api.logout().catch(() => {});
    session.user = null;
    myBoards.loaded = false;
    myBoards.list = [];
    navigate('/login');
  }

  const themes: { value: ThemePref; label: string; icon: typeof Sun }[] = [
    { value: 'light', label: 'Light', icon: Sun },
    { value: 'dark', label: 'Dark', icon: Moon },
    { value: 'system', label: 'System', icon: Monitor },
  ];
</script>

<svelte:window onclick={() => (menuOpen = false)} />

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="scrim" onclick={onclose}></div>
{/if}

<aside class:open aria-label="Navigation">
  <div class="side">
    <a class="brand" href="/">
      <img src="/favicon.svg" alt="" width="26" height="26" />
      <span>Cork</span>
    </a>

    <nav class="nav">
      <a href="/" class:on={path === '/'}><LayoutGrid size={18} /> Boards</a>
      <a href="/pins" class:on={path.startsWith('/pins')}><Images size={18} /> All pins</a>
      <a href="/settings" class:on={path === '/settings'}><Settings size={18} /> Settings</a>
      {#if session.user?.is_admin}
        <a href="/admin" class:on={path === '/admin'}><Shield size={18} /> Admin</a>
      {/if}
    </nav>

    <div class="label">
      <span>Your boards</span>
      <button class="add" aria-label="New board" title="New board" onclick={() => (creating = true)}>
        <Plus size={16} />
      </button>
    </div>

    <div class="blist">
      {#each myBoards.list as b (b.id)}
        <a href="/b/{b.id}" class:on={activeBoard === b.id} title={b.name}>
          {#if b.covers[0]}
            <img src={b.covers[0].url} alt="" style:background={b.covers[0].color} loading="lazy" />
          {:else}
            <span class="ph"></span>
          {/if}
          <span class="name">{b.name}</span>
          {#if !b.is_public}<Lock size={12} class="lock" />{/if}
          <small>{b.pin_count}</small>
        </a>
      {:else}
        {#if myBoards.loaded}
          <button class="empty-board" onclick={() => (creating = true)}>
            <Plus size={16} /> Create your first board
          </button>
        {/if}
      {/each}
    </div>

    <div class="me-wrap">
      {#if menuOpen}
        <div class="menu" role="menu">
          <div class="themes" role="radiogroup" aria-label="Theme">
            {#each themes as t (t.value)}
              <button
                role="radio"
                aria-checked={theme.pref === t.value}
                class:on={theme.pref === t.value}
                onclick={(e) => {
                  e.stopPropagation();
                  setTheme(t.value);
                }}
              >
                <t.icon size={15} />
                {t.label}
              </button>
            {/each}
          </div>
          <a role="menuitem" href="/settings"><Settings size={16} /> Settings &amp; bookmarklet</a>
          <button role="menuitem" onclick={logout}><LogOut size={16} /> Sign out</button>
        </div>
      {/if}
      <button
        class="me"
        aria-expanded={menuOpen}
        aria-label="Account menu"
        onclick={(e) => {
          e.stopPropagation();
          menuOpen = !menuOpen;
        }}
      >
        <span class="avatar">{session.user?.username.slice(0, 1).toUpperCase()}</span>
        <span class="who">
          <strong>{session.user?.username}</strong>
          {#if session.user?.is_admin}<small>Admin</small>{/if}
        </span>
        <ChevronUp size={16} />
      </button>
    </div>
  </div>
</aside>

{#if creating}
  <BoardForm onclose={() => (creating = false)} onsaved={(b) => navigate(`/b/${b.id}`)} />
{/if}

<style>
  aside {
    position: sticky;
    top: 0;
    height: 100vh;
    height: 100dvh;
    width: var(--sidebar-w);
    flex-shrink: 0;
    z-index: 60;
  }
  .side {
    height: 100%;
    padding: 14px 10px 10px;
    display: flex;
    flex-direction: column;
    background: var(--side);
    border-right: 1px solid var(--border);
    overflow: hidden;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 9px;
    padding: 0 6px 16px;
    font-weight: 650;
    font-size: 1rem;
    letter-spacing: -0.015em;
  }
  .nav {
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .nav a {
    display: flex;
    align-items: center;
    gap: 9px;
    height: 32px;
    padding: 0 8px;
    border-radius: 7px;
    font-weight: 500;
    transition: background 0.12s;
  }
  .nav a :global(svg) {
    color: var(--accent);
  }
  .nav a:hover,
  .blist a:hover {
    background: var(--surface-2);
  }
  .nav a.on,
  .blist a.on {
    background: var(--surface-3);
    font-weight: 550;
  }
  .label {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 18px 4px 5px 8px;
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-2);
  }
  .add {
    width: 24px;
    height: 24px;
    border-radius: 6px;
    border: none;
    background: transparent;
    color: var(--text-2);
    display: grid;
    place-items: center;
    cursor: pointer;
  }
  .add:hover {
    background: var(--surface-2);
    color: var(--text);
  }
  .blist {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 1px;
    scrollbar-width: thin;
  }
  .blist a {
    display: flex;
    align-items: center;
    gap: 9px;
    flex-shrink: 0;
    height: 32px;
    padding: 0 8px 0 5px;
    border-radius: 7px;
    transition: background 0.12s;
  }
  .blist img,
  .ph {
    width: 24px;
    height: 24px;
    border-radius: 5px;
    object-fit: cover;
    flex-shrink: 0;
  }
  .ph {
    background: var(--surface-3);
  }
  .name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .blist :global(.lock) {
    color: var(--text-2);
    flex-shrink: 0;
  }
  .blist small {
    color: var(--text-2);
    font-size: 0.78rem;
    font-weight: 400;
  }
  .empty-board {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 4px 0;
    padding: 8px;
    border-radius: 7px;
    border: none;
    background: var(--surface-2);
    color: var(--text-2);
    font: inherit;
    font-weight: 550;
    cursor: pointer;
  }
  .me-wrap {
    position: relative;
    padding-top: 6px;
  }
  .me {
    width: 100%;
    height: 40px;
    display: flex;
    align-items: center;
    gap: 9px;
    padding: 0 8px 0 5px;
    border-radius: 7px;
    border: none;
    background: none;
    color: var(--text);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  .me:hover {
    background: var(--surface-2);
  }
  .me > :global(svg) {
    color: var(--text-2);
  }
  .avatar {
    width: 26px;
    height: 26px;
    border-radius: 50%;
    background: var(--accent);
    color: var(--accent-text);
    display: grid;
    place-items: center;
    font-size: 0.75rem;
    font-weight: 600;
    flex-shrink: 0;
  }
  .who {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    line-height: 1.2;
  }
  .who strong {
    font-size: 0.8125rem;
    font-weight: 550;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .who small {
    color: var(--text-2);
    font-size: 0.72rem;
  }
  .menu {
    position: absolute;
    left: 0;
    right: 0;
    bottom: calc(100% + 6px);
    padding: 5px;
    border-radius: var(--radius);
    background: var(--surface);
    box-shadow: var(--shadow);
    display: flex;
    flex-direction: column;
    gap: 1px;
    animation: pop 0.14s ease-out;
  }
  .menu a,
  .menu > button {
    display: flex;
    align-items: center;
    gap: 9px;
    height: 30px;
    padding: 0 8px;
    border-radius: 6px;
    border: none;
    background: none;
    color: inherit;
    font: inherit;
    cursor: pointer;
    text-align: left;
  }
  .menu a:hover,
  .menu > button:hover {
    background: var(--surface-2);
  }
  .themes {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 2px;
    padding: 2px;
    margin-bottom: 4px;
    border-radius: 7px;
    background: var(--surface-2);
  }
  .themes button {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1px;
    padding: 5px 2px;
    border: none;
    border-radius: 5px;
    background: none;
    color: var(--text-2);
    font: inherit;
    font-size: 0.72rem;
    cursor: pointer;
  }
  .themes button.on {
    background: var(--surface);
    color: var(--text);
    box-shadow: var(--shadow-sm);
  }
  .scrim {
    display: none;
  }
  @keyframes pop {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
  }

  /* tablet & mobile: off-canvas drawer */
  @media (max-width: 1023px) {
    aside {
      position: fixed;
      left: 0;
      top: 0;
      bottom: 0;
      height: auto;
      width: min(280px, 86vw);
      transform: translateX(-105%);
      transition: transform 0.22s cubic-bezier(0.2, 0.8, 0.2, 1);
      z-index: 90;
    }
    aside.open {
      transform: none;
    }
    aside.open .side {
      box-shadow: var(--shadow-lg);
    }
    .scrim {
      display: block;
      position: fixed;
      inset: 0;
      z-index: 89;
      background: var(--scrim);
      animation: fade 0.18s ease-out;
    }
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
</style>
