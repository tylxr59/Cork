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
  <div class="side glass">
    <a class="brand" href="/">
      <img src="/favicon.svg" alt="" width="34" height="34" />
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
        <div class="menu glass" role="menu">
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
    padding: 18px 0 18px 18px;
    flex-shrink: 0;
    z-index: 60;
  }
  .side {
    height: 100%;
    border-radius: 28px;
    padding: 18px 12px 12px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    box-shadow: var(--shadow);
    overflow: hidden;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 2px 10px 18px;
    font-weight: 800;
    font-size: 1.3rem;
    letter-spacing: -0.03em;
  }
  .brand img {
    border-radius: 11px;
    box-shadow: 0 6px 16px rgb(109 93 252 / 35%);
  }
  .nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .nav a {
    display: flex;
    align-items: center;
    gap: 11px;
    padding: 10px 12px;
    border-radius: 14px;
    color: var(--text-2);
    font-weight: 650;
    transition: background 0.15s;
  }
  .nav a:hover {
    background: var(--glass-strong);
    color: var(--text);
  }
  .nav a.on {
    background: var(--surface);
    color: var(--text);
    box-shadow: var(--shadow-sm);
  }
  .label {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 18px 8px 6px 12px;
    font-size: 0.7rem;
    font-weight: 750;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--text-2);
  }
  .add {
    width: 28px;
    height: 28px;
    border-radius: 9px;
    border: none;
    background: transparent;
    color: var(--text-2);
    display: grid;
    place-items: center;
    cursor: pointer;
  }
  .add:hover {
    background: var(--glass-strong);
    color: var(--text);
  }
  .blist {
    flex: 1;
    overflow-y: auto;
    margin: 0 -4px;
    padding: 0 4px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    scrollbar-width: thin;
  }
  .blist a {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 10px 6px 6px;
    border-radius: 14px;
    font-weight: 600;
    font-size: 0.9rem;
  }
  .blist a:hover {
    background: var(--glass-strong);
  }
  .blist a.on {
    background: var(--surface);
    box-shadow: var(--shadow-sm);
  }
  .blist img,
  .ph {
    width: 36px;
    height: 36px;
    border-radius: 11px;
    object-fit: cover;
    flex-shrink: 0;
  }
  .ph {
    background: var(--brand-gradient);
    opacity: 0.35;
  }
  .name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .blist :global(.lock) {
    color: var(--text-3);
    flex-shrink: 0;
  }
  .blist small {
    color: var(--text-3);
    font-weight: 650;
    font-size: 0.78rem;
  }
  .empty-board {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 4px 6px;
    padding: 12px;
    border-radius: 14px;
    border: 1.5px dashed var(--border);
    background: none;
    color: var(--text-2);
    font: inherit;
    font-weight: 650;
    cursor: pointer;
  }
  .me-wrap {
    position: relative;
    padding-top: 10px;
  }
  .me {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px 8px 8px;
    border-radius: 16px;
    border: none;
    background: var(--glass-strong);
    color: var(--text);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  .avatar {
    width: 34px;
    height: 34px;
    border-radius: 50%;
    background: linear-gradient(135deg, #ffc48c, var(--accent-2));
    color: #fff;
    display: grid;
    place-items: center;
    font-weight: 800;
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
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .who small {
    color: var(--text-2);
    font-size: 0.75rem;
  }
  .menu {
    position: absolute;
    left: 0;
    right: 0;
    bottom: calc(100% + 6px);
    padding: 8px;
    border-radius: 20px;
    background: var(--glass-strong);
    box-shadow: var(--shadow-lg);
    display: flex;
    flex-direction: column;
    gap: 2px;
    animation: pop 0.15s ease-out;
  }
  .menu a,
  .menu > button {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    border-radius: 12px;
    border: none;
    background: none;
    color: inherit;
    font: inherit;
    font-weight: 650;
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
    gap: 4px;
    padding: 4px;
    margin-bottom: 4px;
    border-radius: 14px;
    background: var(--surface-2);
  }
  .themes button {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    padding: 8px 4px;
    border: none;
    border-radius: 10px;
    background: none;
    color: var(--text-2);
    font: inherit;
    font-size: 0.75rem;
    font-weight: 650;
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
      transform: translateY(6px);
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
      padding: 10px;
      width: min(300px, 86vw);
      transform: translateX(-105%);
      transition: transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
      z-index: 90;
    }
    aside.open {
      transform: none;
    }
    .side {
      background: var(--glass-strong);
      box-shadow: var(--shadow-lg);
    }
    .scrim {
      display: block;
      position: fixed;
      inset: 0;
      z-index: 89;
      background: var(--scrim);
      -webkit-backdrop-filter: blur(3px);
      backdrop-filter: blur(3px);
      animation: fade 0.2s ease-out;
    }
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
</style>
