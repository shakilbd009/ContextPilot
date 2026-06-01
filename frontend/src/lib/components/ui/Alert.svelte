<script lang="ts">
  import type { Snippet } from 'svelte';

  interface AlertProps {
    variant?: 'info' | 'success' | 'warning' | 'error';
    class?: string;
    children: Snippet;
    dismissible?: boolean;
  }

  let { variant = 'info', class: className = '', children, dismissible = false }: AlertProps = $props();

  let visible = $state(true);

  const iconMap = {
    info: `<svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/><path d="M8 7v4M8 5.5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`,
    success: `<svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/><path d="M5.5 8.5l2 2 3-3.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
    warning: `<svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><path d="M8 2L14.5 13.5H1.5L8 2z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/><path d="M8 6v4M8 11.5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`,
    error: `<svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true"><circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/><path d="M5.5 5.5l5 5M10.5 5.5l-5 5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`,
  };
</script>

{#if visible}
  <div class="alert alert--{variant} {className}" role="alert">
    <span class="alert__icon" aria-hidden="true">{@html iconMap[variant]}</span>
    <div class="alert__content">
      {@render children()}
    </div>
    {#if dismissible}
      <button
        type="button"
        class="alert__dismiss"
        aria-label="Dismiss"
        onclick={() => (visible = false)}
      >
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
          <path d="M3 3l8 8M11 3l-8 8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      </button>
    {/if}
  </div>
{/if}

<style>
  .alert {
    display: flex;
    align-items: flex-start;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius-md);
    border: 1px solid;
  }

  .alert--info {
    background: rgb(26 86 219 / 0.05);
    border-color: rgb(26 86 219 / 0.2);
    color: var(--color-primary);
  }

  .alert--success {
    background: rgb(5 150 105 / 0.05);
    border-color: rgb(5 150 105 / 0.2);
    color: var(--color-success);
  }

  .alert--warning {
    background: rgb(217 119 6 / 0.05);
    border-color: rgb(217 119 6 / 0.2);
    color: var(--color-warning);
  }

  .alert--error {
    background: rgb(220 38 38 / 0.05);
    border-color: rgb(220 38 38 / 0.2);
    color: var(--color-danger);
  }

  .alert__icon {
    flex-shrink: 0;
    margin-top: 1px;
  }

  .alert__content {
    flex: 1;
    font-size: var(--text-sm);
    color: var(--color-text-primary);
  }

  .alert__dismiss {
    background: none;
    border: none;
    cursor: pointer;
    color: currentColor;
    opacity: 0.6;
    padding: 0;
    flex-shrink: 0;
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .alert__dismiss:hover {
    opacity: 1;
  }

  .alert__dismiss:focus-visible {
    outline: 2px solid currentColor;
    outline-offset: 2px;
  }
</style>