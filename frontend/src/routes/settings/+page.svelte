<script lang="ts">
  import { Card, Input, Button, Alert } from '$lib/components/ui';

  let email = $state('');
  let saving = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!email.trim()) return;
    saving = true;
    // TODO: connect to real auth when backend ready
    await new Promise((r) => setTimeout(r, 500));
    window.location.href = '/';
  }
</script>

<svelte:head>
  <title>Settings — ContextPilot</title>
</svelte:head>

<div class="page-container">
  <header class="page-header">
    <h1 class="page-title">Settings</h1>
  </header>

  <div class="settings-sections">
    <Card>
      <h2 class="section-title">Profile</h2>
      <form onsubmit={handleSubmit} class="settings-form" novalidate>
        <Input
          label="Email address"
          type="email"
          placeholder="you@company.com"
          bind:value={email}
          required
        />
        <div class="form-actions">
          <Button variant="primary" type="submit" loading={saving}>
            Save changes
          </Button>
        </div>
      </form>
    </Card>
  </div>
</div>

<style>
  .page-container {
    max-width: 800px;
    margin: 0 auto;
    padding: var(--space-8) var(--space-6);
  }

  .page-header {
    margin-bottom: var(--space-6);
  }

  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--color-text-primary);
    margin: 0;
  }

  .settings-sections {
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .section-title {
    font-size: var(--text-lg);
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0 0 var(--space-4);
  }

  .settings-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
  }
</style>