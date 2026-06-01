<script lang="ts">
  import { page } from '$app/stores';
  import { Button, Card } from '$lib/components/ui';
  import { isAuthenticated } from '$lib/stores/auth';
</script>

<svelte:head>
  <title>Page not found — ContextPilot</title>
</svelte:head>

<div class="page-container">
  <div class="not-found">
    <div class="not-found__code" aria-hidden="true">404</div>
    <h1 class="not-found__title">Page not found</h1>
    <p class="not-found__desc">
      The page <code>{$page.url.pathname}</code> doesn't exist or has been moved.
    </p>
    <div class="not-found__actions">
      {#if $isAuthenticated}
        <Button variant="primary" onclick={() => window.location.href = '/'}>
          Go to dashboard
        </Button>
      {:else}
        <Button variant="primary" onclick={() => window.location.href = '/'}>
          Go to landing
        </Button>
      {/if}
      <Button variant="secondary" onclick={() => window.history.back()}>
        Go back
      </Button>
    </div>
  </div>
</div>

<style>
  .page-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: var(--space-8) var(--space-6);
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: calc(100vh - 56px);
  }

  .not-found {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-4);
    text-align: center;
    max-width: 480px;
  }

  .not-found__code {
    font-size: 6rem;
    font-weight: 800;
    color: var(--color-border);
    line-height: 1;
    font-family: var(--font-mono);
  }

  .not-found__title {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--color-text-primary);
    margin: 0;
  }

  .not-found__desc {
    font-size: var(--text-base);
    color: var(--color-text-secondary);
    margin: 0;
    line-height: 1.6;
  }

  .not-found__desc code {
    background: var(--color-background-subtle);
    padding: 0.1em 0.4em;
    border-radius: var(--radius-sm);
    font-family: var(--font-mono);
    font-size: var(--text-sm);
  }

  .not-found__actions {
    display: flex;
    gap: var(--space-3);
    flex-wrap: wrap;
    justify-content: center;
    margin-top: var(--space-2);
  }
</style>