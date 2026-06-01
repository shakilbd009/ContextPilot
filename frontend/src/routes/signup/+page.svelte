<script lang="ts">
  import { Button, Input, Card, Alert } from '$lib/components/ui';

  let name = $state('');
  let email = $state('');
  let password = $state('');
  let saving = $state(false);
  let error = $state('');

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!name.trim() || !email.trim() || !password.trim()) return;
    saving = true;
    error = '';

    try {
      // TODO: replace with real auth API call when backend is ready
      await new Promise((r) => setTimeout(r, 500));
      document.cookie = `session_id=demo-session; path=/; SameSite=Lax`;
      window.location.href = '/';
    } catch {
      error = 'Something went wrong. Please try again.';
    } finally {
      saving = false;
    }
  }
</script>

<svelte:head>
  <title>Create account — ContextPilot</title>
</svelte:head>

<div class="auth-container">
  <div class="auth-card">
    <div class="auth-header">
      <svg width="32" height="32" viewBox="0 0 28 28" fill="none" aria-hidden="true">
        <rect width="28" height="28" rx="6" fill="var(--color-primary)"/>
        <path d="M8 10h12M8 14h8M8 18h10" stroke="white" stroke-width="2" stroke-linecap="round"/>
      </svg>
      <h1 class="auth-title">Create your account</h1>
    </div>

    {#if error}
      <Alert variant="error" dismissible>{error}</Alert>
    {/if}

    <form onsubmit={handleSubmit} class="auth-form" novalidate>
      <Input
        label="Full name"
        type="text"
        placeholder="Alex Johnson"
        bind:value={name}
        required
      />
      <Input
        label="Email"
        type="email"
        placeholder="you@company.com"
        bind:value={email}
        required
      />
      <Input
        label="Password"
        type="password"
        placeholder="••••••••"
        bind:value={password}
        required
      />
      <Button variant="primary" type="submit" loading={saving} class="auth-submit">
        Create account
      </Button>
    </form>

    <p class="auth-footer">
      Already have an account? <a href="/login">Sign in</a>
    </p>
  </div>
</div>

<style>
  .auth-container {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: calc(100vh - 56px);
    padding: var(--space-6);
  }

  .auth-card {
    width: 100%;
    max-width: 400px;
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .auth-header {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-3);
    text-align: center;
  }

  .auth-title {
    font-size: var(--text-xl);
    font-weight: 700;
    color: var(--color-text-primary);
    margin: 0;
  }

  .auth-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  :global(.auth-submit) {
    width: 100%;
    margin-top: var(--space-2);
  }

  .auth-footer {
    text-align: center;
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
    margin: 0;
  }

  .auth-footer a {
    color: var(--color-primary);
    text-decoration: none;
    font-weight: 500;
  }

  .auth-footer a:hover {
    text-decoration: underline;
  }

  .auth-footer a:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
    border-radius: var(--radius-sm);
  }
</style>