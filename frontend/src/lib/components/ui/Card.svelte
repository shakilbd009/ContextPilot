<script lang="ts">
  import type { Snippet } from 'svelte';

  interface CardProps {
    variant?: 'default' | 'meeting' | 'briefing';
    children: Snippet;
  }

  let { variant = 'default', children }: CardProps = $props();
</script>

<div class="card card--{variant}">
  {@render children()}
</div>

<style>
  .card {
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: var(--space-4);
    box-shadow: var(--shadow-sm);
    transition: box-shadow var(--transition-fast), border-color var(--transition-fast);
  }

  .card:hover {
    box-shadow: var(--shadow-md);
    border-color: var(--color-text-muted);
  }

  .card--meeting {
    background: color-mix(in oklch, var(--color-primary) 4%, var(--color-background));
    border-color: color-mix(in oklch, var(--color-primary) 30%, var(--color-border));
  }

  .card--briefing {
    background: color-mix(in oklch, var(--color-success) 4%, var(--color-background));
    border-color: color-mix(in oklch, var(--color-success) 30%, var(--color-border));
  }
</style>