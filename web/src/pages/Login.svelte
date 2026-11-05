<script lang="ts">
  import { api } from '../lib/api';
  import { navigate, router } from '../lib/router.svelte';
  import { refreshSession, session } from '../lib/session.svelte';

  let mode = $state<'login' | 'register'>(
    session.needsSetup || router.current.path === '/register' ? 'register' : 'login',
  );
  let username = $state('');
  let password = $state('');
  let error = $state('');
  let busy = $state(false);

  const next = $derived.by(() => {
    const n = router.current.query.get('next') ?? '/';
    return n.startsWith('/') && !n.startsWith('//') && n !== '/login' ? n : '/';
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    error = '';
    busy = true;
    // signing in redirects away from this page, which clears the ?next= query
    const target = next;
    try {
      if (mode === 'login') await api.login(username, password);
      else await api.register(username, password);
      await refreshSession();
      navigate(target, { replace: true });
    } catch (err) {
      error = (err as Error).message;
    } finally {
      busy = false;
    }
  }
</script>

<div class="wrap">
  <form class="glass panel" onsubmit={submit}>
    <img src="/favicon.svg" alt="" width="56" height="56" />
    {#if session.needsSetup}
      <h1>Welcome to Cork</h1>
      <p class="muted">Create the first account. It becomes the admin for this Cork.</p>
    {:else if mode === 'login'}
      <h1>Sign in</h1>
    {:else}
      <h1>Create an account</h1>
    {/if}

    <label class="field">
      <span>Username</span>
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input" bind:value={username} autocomplete="username" required autofocus />
    </label>
    <label class="field">
      <span>Password</span>
      <input
        class="input"
        type="password"
        bind:value={password}
        autocomplete={mode === 'login' ? 'current-password' : 'new-password'}
        minlength={mode === 'register' ? 8 : undefined}
        required
      />
    </label>
    {#if error}<p class="error-text">{error}</p>{/if}
    <button class="btn primary" disabled={busy}>
      {mode === 'login' ? 'Sign in' : session.needsSetup ? 'Create admin account' : 'Create account'}
    </button>

    {#if !session.needsSetup && session.registrationOpen}
      <p class="switch muted">
        {#if mode === 'login'}
          New here? <button type="button" onclick={() => (mode = 'register')}>Create an account</button>
        {:else}
          Already have an account? <button type="button" onclick={() => (mode = 'login')}>Sign in</button>
        {/if}
      </p>
    {/if}
  </form>
</div>

<style>
  .wrap {
    min-height: 100vh;
    min-height: 100dvh;
    display: grid;
    place-items: center;
    padding: 24px;
  }
  form {
    width: 100%;
    max-width: 400px;
    padding: 40px 32px 32px;
    box-shadow: var(--shadow-lg);
    display: flex;
    flex-direction: column;
    gap: 16px;
    text-align: center;
  }
  form img {
    margin: 0 auto 4px;
    border-radius: 16px;
    box-shadow: var(--shadow-accent);
  }
  .field {
    text-align: left;
  }
  .switch {
    font-size: 0.9rem;
  }
  .switch button {
    border: none;
    background: none;
    color: var(--accent);
    font: inherit;
    font-weight: 600;
    cursor: pointer;
    padding: 0;
  }
</style>
