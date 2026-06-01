<script lang="ts">
  import { Button, Input, Card, Alert } from '$lib/components/ui';
  import { login } from '$lib/stores/auth';

  let email = $state('');
  let password = $state('');
  let saving = $state(false);
  let error = $state('');

  // Derive a deterministic v4-style UUID from an email address
  async function hashEmailToUUID(email: string): Promise<string> {
    const encoder = new TextEncoder();
    const data = encoder.encode(email.toLowerCase().trim());
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));

    // Construct a v4 UUID: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
    // y in [8, 9, a, b] to satisfy variant bits
    const hex = hashArray.map((b) => b.toString(16).padStart(2, '0')).join('');
    const p1 = hex.slice(0, 8);
    const p2 = hex.slice(8, 12);
    const p3 = '4' + hex.slice(13, 16);
    const p4 = ((parseInt(hex[16], 16) & 0x3) | 0x8).toString(16) + hex.slice(17, 20);
    const p5 = hex.slice(20, 32);
    return `${p1}-${p2}-${p3}-${p4}-${p5}`;
  }

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!email.trim() || !password.trim()) return;
    saving = true;
    error = '';

    try {
      // TODO: replace with real auth API call when backend is ready
      // Simulate login by setting a session cookie and auth store
      await new Promise((r) => setTimeout(r, 500));

      // Set session cookie (simplified for demo; real auth uses server-side session)
      document.cookie = `session_id=demo-session; path=/; SameSite=Lax`;

      // Derive a consistent X-User-ID UUID from email for backend auth
      // Uses a simple hash to generate a deterministic v4-style UUID
      const emailHash = await hashEmailToUUID(email);
      document.cookie = `X-User-ID=${emailHash}; path=/; SameSite=Lax`;

      // Update auth store
      login('Demo User', email);

      // Redirect to dashboard or ?redirectTo param
      const params = new URLSearchParams(window.location.search);
      const redirectTo = params.get('redirectTo') || '/';
      window.location.href = redirectTo;
    } catch {
      error = 'Invalid email or password';
    } finally {
      saving = false;
    }
  }
</script>

<svelte:head>
  <title>Sign in — ContextPilot</title>
</svelte:head>

<div class="auth-container">
  <div class="auth-card">
    <div class="auth-header">
      <svg width="32" height="32" viewBox="0 0 28 28" fill="none" aria-hidden="true">
        <rect width="28" height="28" rx="6" fill="var(--color-primary)"/>
        <path d="M8 10h12M8 14h8M8 18h10" stroke="white" stroke-width="2" stroke-linecap="round"/>
      </svg>
      <h1 class="auth-title">Sign in to ContextPilot</h1>
    </div>

    {#if error}
      <Alert variant="error" dismissible>{error}</Alert>
    {/if}

    <form onsubmit={handleSubmit} class="auth-form" novalidate>
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
        Sign in
      </Button>
    </form>

    <p class="auth-footer">
      Don't have an account? <a href="/signup">Sign up</a>
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