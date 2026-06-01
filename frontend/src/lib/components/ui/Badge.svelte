<script lang="ts">
  interface Props {
    variant?: 'success' | 'warning' | 'danger' | 'neutral';
    children: import('svelte').Snippet;
  }

  let { variant = 'neutral', children }: Props = $props();
</script>

<span class="badge badge--{variant}">
  {@render children()}
</span>

<style>
  .badge {
    display: inline-flex;
    align-items: center;
    padding: 0.125rem var(--space-2);
    border-radius: var(--radius-full);
    font-size: var(--text-xs);
    font-weight: 600;
    line-height: 1.5;
    white-space: nowrap;
  }

  .badge--success {
    background: rgb(5 150 105 / 0.1);
    color: var(--color-success);
  }

  .badge--warning {
    background: rgb(217 119 6 / 0.1);
    color: var(--color-warning);
  }

  .badge--danger {
    background: rgb(220 38 38 / 0.1);
    color: var(--color-danger);
  }

  .badge--neutral {
    background: var(--color-background-subtle);
    color: var(--color-text-secondary);
  }
</style>