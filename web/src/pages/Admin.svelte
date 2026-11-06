<script lang="ts">
  import Trash from '@lucide/svelte/icons/trash-2';
  import KeyRound from '@lucide/svelte/icons/key-round';
  import { api } from '../lib/api';
  import { formatDate } from '../lib/format';
  import { session, toast, toastError } from '../lib/session.svelte';
  import type { AdminUser } from '../lib/types';

  let users = $state<AdminUser[]>([]);
  let registrationOpen = $state(false);
  let loading = $state(true);

  let newUser = $state({ username: '', password: '', is_admin: false });
  let createError = $state('');

  async function load() {
    try {
      [users, { registration_open: registrationOpen }] = await Promise.all([api.adminUsers(), api.adminSettings()]);
    } catch (e) {
      toastError(e);
    } finally {
      loading = false;
    }
  }
  load();

  async function toggleRegistration() {
    try {
      const s = await api.adminSaveSettings({ registration_open: !registrationOpen });
      registrationOpen = s.registration_open;
      session.registrationOpen = s.registration_open;
    } catch (e) {
      toastError(e);
    }
  }

  async function create(e: SubmitEvent) {
    e.preventDefault();
    createError = '';
    try {
      await api.adminCreateUser(newUser);
      toast(`Created ${newUser.username}`);
      newUser = { username: '', password: '', is_admin: false };
      load();
    } catch (err) {
      createError = (err as Error).message;
    }
  }

  async function toggleAdmin(u: AdminUser) {
    try {
      await api.adminUpdateUser(u.id, { is_admin: !u.is_admin });
      load();
    } catch (e) {
      toastError(e);
    }
  }

  async function resetPassword(u: AdminUser) {
    const pw = prompt(`New password for ${u.username} (at least 8 characters):`);
    if (!pw) return;
    try {
      await api.adminUpdateUser(u.id, { password: pw });
      toast(`Password reset for ${u.username}`);
    } catch (e) {
      toastError(e);
    }
  }

  async function remove(u: AdminUser) {
    if (!confirm(`Delete ${u.username} with all ${u.board_count} boards and ${u.pin_count} pins? This can't be undone.`))
      return;
    try {
      await api.adminDeleteUser(u.id);
      toast(`Deleted ${u.username}`);
      load();
    } catch (e) {
      toastError(e);
    }
  }
</script>

<svelte:head><title>Admin · Cork</title></svelte:head>

<div class="page narrow">
  <h1 class="page-title">Admin</h1>

  {#if loading}
    <div class="spinner"></div>
  {:else}
    <section class="card">
      <h2>Registration</h2>
      <label class="check">
        <input type="checkbox" checked={registrationOpen} onchange={toggleRegistration} />
        <span>
          <strong>Allow anyone to create an account</strong><br />
          <small class="muted">When off, only admins can add users below.</small>
        </span>
      </label>
    </section>

    <section class="card">
      <h2>Users</h2>
      <table>
        <thead>
          <tr><th>User</th><th>Boards</th><th>Pins</th><th>Joined</th><th></th></tr>
        </thead>
        <tbody>
          {#each users as u (u.id)}
            <tr>
              <td>
                <strong>{u.username}</strong>
                {#if u.is_admin}<span class="badge">admin</span>{/if}
              </td>
              <td>{u.board_count}</td>
              <td>{u.pin_count}</td>
              <td>{formatDate(u.created_at)}</td>
              <td class="row-actions">
                {#if u.id !== session.user?.id}
                  <button class="btn small" onclick={() => toggleAdmin(u)}>
                    {u.is_admin ? 'Remove admin' : 'Make admin'}
                  </button>
                {/if}
                <button class="btn small icon" title="Reset password" aria-label="Reset password" onclick={() => resetPassword(u)}>
                  <KeyRound size={15} />
                </button>
                {#if u.id !== session.user?.id}
                  <button class="btn small icon danger" title="Delete user" aria-label="Delete user" onclick={() => remove(u)}>
                    <Trash size={15} />
                  </button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>

      <form onsubmit={create}>
        <h3>Add a user</h3>
        <div class="new-user">
          <input class="input" placeholder="Username" bind:value={newUser.username} required autocomplete="off" />
          <input
            class="input"
            type="password"
            placeholder="Password"
            bind:value={newUser.password}
            minlength="8"
            required
            autocomplete="new-password"
          />
          <button class="btn primary">Add</button>
        </div>
        <label class="check"><input type="checkbox" bind:checked={newUser.is_admin} /> Admin</label>
        {#if createError}<p class="error-text">{createError}</p>{/if}
      </form>
    </section>
  {/if}
</div>

<style>
  section {
    padding: 20px;
    margin-bottom: 20px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    overflow-x: auto;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.9rem;
  }
  th {
    text-align: left;
    color: var(--text-2);
    font-weight: 600;
    font-size: 0.8rem;
    padding: 6px 8px;
    border-bottom: 1px solid var(--border);
  }
  td {
    padding: 10px 8px;
    border-bottom: 1px solid var(--border);
    white-space: nowrap;
  }
  .row-actions {
    display: flex;
    gap: 6px;
    justify-content: flex-end;
  }
  .badge {
    margin-left: 6px;
    padding: 2px 8px;
    border-radius: 999px;
    background: var(--accent-soft);
    color: var(--accent);
    font-size: 0.72rem;
    font-weight: 700;
  }
  form {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding-top: 8px;
  }
  h3 {
    font-size: 1rem;
  }
  .new-user {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .new-user .input {
    flex: 1;
    min-width: 160px;
  }
</style>
