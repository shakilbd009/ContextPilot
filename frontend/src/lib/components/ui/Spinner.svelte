<script lang="ts">
  interface Props {
    label?: string;
    size?: 'sm' | 'md';
  }

  let { label = 'Loading', size = 'md' }: Props = $props();
</script>

<div class="spinner spinner--{size}" role="status" aria-label={label}>
  <span class="sr-only">{label}</span>
</div>

<style>
  .spinner {
    border-radius: 50%;
    border-style: solid;
    border-color: var(--color-border);
    border-top-color: var(--color-primary);
    animation: spin 0.7s linear infinite;
  }

  .spinner--sm {
    width: 16px;
    height: 16px;
    border-width: 2px;
  }

  .spinner--md {
    width: 24px;
    height: 24px;
    border-width: 3px;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>