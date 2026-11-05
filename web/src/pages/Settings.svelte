<script lang="ts">
  import { api } from '../lib/api';
  import { formatDate } from '../lib/format';
  import { session, toast } from '../lib/session.svelte';
  import Bookmarklet from '../components/Bookmarklet.svelte';

  let current = $state('');
  let next = $state('');
  let confirmPw = $state('');
  let error = $state('');
  let busy = $state(false);

  async function changePassword(e: SubmitEvent) {
    e.preventDefault();
    error = '';
    if (next !== confirmPw) {
      error = 'New passwords don’t match';
      return;
    }
    busy = true;
    try {
      await api.changePassword(current, next);
      current = next = confirmPw = '';
      toast('Password changed. Other devices were signed out.');
    } catch (err) {
      error = (err as Error).message;
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head><title>Settings · Cork</title></svelte:head>

<div class="page narrow">
  <h1 class="page-title">Settings</h1>

  <section class="card">
    <h2>Bookmarklet</h2>
    <Bookmarklet />
    <p class="muted small">
      Bookmarks bar hidden? Press <kbd>Ctrl</kbd>/<kbd>⌘</kbd> + <kbd>Shift</kbd> + <kbd>B</kbd> to show it. On phones,
      share a page’s link and paste it into Cork’s “Create pin” page instead.
    </p>
  </section>

  <section class="card">
    <h2>Account</h2>
    <p class="muted">
      Signed in as <strong>{session.user?.username}</strong>{session.user?.is_admin ? ' (admin)' : ''} · member since
      {session.user ? formatDate(session.user.created_at) : ''}
    </p>
    <form onsubmit={changePassword}>
      <label class="field">
        <span>Current password</span>
        <input class="input" type="password" bind:value={current} autocomplete="current-password" required />
      </label>
      <label class="field">
        <span>New password</span>
        <input class="input" type="password" bind:value={next} autocomplete="new-password" minlength="8" required />
      </label>
      <label class="field">
        <span>Confirm new password</span>
        <input class="input" type="password" bind:value={confirmPw} autocomplete="new-password" required />
      </label>
      {#if error}<p class="error-text">{error}</p>{/if}
      <div><button class="btn primary" disabled={busy}>Change password</button></div>
    </form>
  </section>
</div>

<style>
  section {
    padding: 26px;
    margin-bottom: 20px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  form {
    display: flex;
    flex-direction: column;
    gap: 14px;
    max-width: 360px;
  }
  .small {
    font-size: 0.85rem;
  }
  kbd {
    font-family: inherit;
    font-size: 0.8em;
    padding: 1px 6px;
    border-radius: 5px;
    background: var(--surface-2);
    border: 1px solid var(--border);
  }
</style>
