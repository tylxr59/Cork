<script lang="ts">
  import { setUnauthorizedHandler } from './lib/api';
  import { closeModal, match, navigate, router } from './lib/router.svelte';
  import { loadBoards, onPinEvent, refreshSession, session } from './lib/session.svelte';
  import './lib/theme.svelte';
  import Sidebar from './components/Sidebar.svelte';
  import TopBar from './components/TopBar.svelte';
  import Modal from './components/Modal.svelte';
  import Toasts from './components/Toasts.svelte';
  import Login from './pages/Login.svelte';
  import Boards from './pages/Boards.svelte';
  import AllPins from './pages/AllPins.svelte';
  import Board from './pages/Board.svelte';
  import PinView from './pages/PinView.svelte';
  import NewPin from './pages/NewPin.svelte';
  import Settings from './pages/Settings.svelte';
  import Admin from './pages/Admin.svelte';

  let loadError = $state('');
  let drawerOpen = $state(false);

  refreshSession().catch((e) => (loadError = e.message));

  setUnauthorizedHandler(() => {
    if (session.user) {
      session.user = null;
      navigate(`/login?next=${encodeURIComponent(location.pathname + location.search)}`, { replace: true });
    }
  });

  // the sidebar lists the user's boards
  $effect(() => {
    if (session.user) loadBoards().catch(() => {});
  });

  // keep sidebar pin counts and covers fresh as pins change
  let refreshTimer: ReturnType<typeof setTimeout>;
  onPinEvent(() => {
    clearTimeout(refreshTimer);
    refreshTimer = setTimeout(() => loadBoards(true).catch(() => {}), 400);
  });

  type Route =
    | { page: 'login' }
    | { page: 'boards' }
    | { page: 'pins' }
    | { page: 'board'; id: string }
    | { page: 'pin'; id: string }
    | { page: 'new' }
    | { page: 'settings' }
    | { page: 'admin' }
    | { page: 'notfound' };

  const PUBLIC = new Set(['login', 'board', 'pin', 'notfound']);

  function resolve(path: string): Route {
    let m;
    if (match('/', path)) return { page: 'boards' };
    if (match('/login', path) || match('/register', path)) return { page: 'login' };
    if (match('/pins', path)) return { page: 'pins' };
    if ((m = match('/b/:id', path))) return { page: 'board', id: m.id };
    if ((m = match('/p/:id', path))) return { page: 'pin', id: m.id };
    if (match('/new', path)) return { page: 'new' };
    if (match('/settings', path)) return { page: 'settings' };
    if (match('/admin', path)) return { page: 'admin' };
    return { page: 'notfound' };
  }

  // A pin opened from a grid shows as a slide-over on top of the page it came from.
  const modalPin = $derived(router.current.bg ? match('/p/:id', router.current.path)?.id : undefined);
  // Derive from a plain string so the page underneath doesn't re-render (and
  // refetch) when a pin modal opens or closes over it.
  const pagePath = $derived(modalPin ? new URL(router.current.bg!, location.origin).pathname : router.current.path);
  const route = $derived(resolve(pagePath));
  const popup = $derived(route.page === 'new' && router.current.query.get('popup') === '1');
  const withShell = $derived(!!session.user && route.page !== 'login' && !popup);

  // guards
  $effect(() => {
    if (!session.ready) return;
    const here = location.pathname + location.search;
    if (!session.user && !PUBLIC.has(route.page)) {
      navigate(`/login?next=${encodeURIComponent(here)}`, { replace: true });
    } else if (session.user && route.page === 'login') {
      navigate('/', { replace: true });
    } else if (route.page === 'admin' && !session.user?.is_admin) {
      navigate('/', { replace: true });
    }
  });
</script>

{#if loadError}
  <div class="empty">
    <h2>Can’t reach Cork</h2>
    <p>{loadError}</p>
    <button class="btn" onclick={() => location.reload()}>Retry</button>
  </div>
{:else if !session.ready}
  <div class="spinner"></div>
{:else}
  <div class="shell" class:with-shell={withShell}>
    {#if withShell}
      <Sidebar open={drawerOpen} onclose={() => (drawerOpen = false)} />
    {/if}

    <div class="main">
      {#if route.page !== 'login' && !popup}
        <TopBar onmenu={() => (drawerOpen = true)} />
      {/if}

      <main>
        {#if route.page === 'login'}
          <Login />
        {:else if route.page === 'boards' && session.user}
          <Boards />
        {:else if route.page === 'pins' && session.user}
          <AllPins />
        {:else if route.page === 'board'}
          {#key route.id}<Board id={route.id} />{/key}
        {:else if route.page === 'pin'}
          <div class="page pin-page"><PinView id={route.id} /></div>
        {:else if route.page === 'new' && session.user}
          <NewPin />
        {:else if route.page === 'settings' && session.user}
          <Settings />
        {:else if route.page === 'admin' && session.user?.is_admin}
          <Admin />
        {:else if route.page === 'notfound'}
          <div class="empty">
            <h2>Page not found</h2>
            <a class="btn" href="/">Go home</a>
          </div>
        {/if}
      </main>
    </div>
  </div>

  {#if modalPin}
    <Modal onclose={closeModal} label="Pin" panel>
      <PinView id={modalPin} modal />
    </Modal>
  {/if}
{/if}

<Toasts />

<style>
  .shell {
    min-height: 100vh;
  }
  .shell.with-shell {
    display: flex;
    align-items: flex-start;
  }
  .main {
    flex: 1;
    min-width: 0;
  }
  .pin-page {
    max-width: 1100px;
  }
</style>
