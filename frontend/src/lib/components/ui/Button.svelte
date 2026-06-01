<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
    size?: 'sm' | 'md' | 'lg';
    type?: 'button' | 'submit' | 'reset';
    disabled?: boolean;
    loading?: boolean;
    class?: string;
    'aria-label'?: string;
    onclick?: (e: MouseEvent) => void;
    'aria-pressed'?: boolean | 'true' | 'false';
    'aria-current'?: boolean | 'true' | 'false' | null;
    children: Snippet;
  }

  let {
    variant = 'primary',
    size = 'md',
    type = 'button',
    disabled = false,
    loading = false,
    class: className = '',
    'aria-label': ariaLabel = undefined,
    onclick,
    'aria-pressed': ariaPressed = undefined,
    'aria-current': ariaCurrent = undefined,
    children,
  }: Props = $props();

  const baseClass = 'btn';
  const variantClass = $derived(`btn--${variant}`);
  const sizeClass = $derived(`btn--${size}`);
</script>

<button
  {type}
  class="{baseClass} {variantClass} {sizeClass} {className}"
  disabled={disabled || loading}
  aria-busy={loading}
  aria-label={ariaLabel}
  aria-pressed={ariaPressed}
  aria-current={ariaCurrent}
  onclick={disabled || loading ? undefined : onclick}
>
  {#if loading}
    <span class="btn__spinner" aria-hidden="true"></span>
  {/if}
  <span class="btn__content" class:btn__content--hidden={loading}>
    {@render children()}
  </span>
</button>

<style>
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-4);
    border-radius: var(--radius-sm);
    border: 1px solid transparent;
    font-family: var(--font-sans);
    font-size: var(--text-sm);
    font-weight: 600;
    line-height: 1.5;
    cursor: pointer;
    transition: background-color var(--transition-fast), border-color var(--transition-fast), color var(--transition-fast);
    position: relative;
    white-space: nowrap;
  }

  .btn:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Primary */
  .btn--primary {
    background: var(--color-primary);
    color: white;
    border-color: var(--color-primary);
  }

  .btn--primary:hover:not(:disabled) {
    background: var(--color-primary-hover);
    border-color: var(--color-primary-hover);
  }

  .btn--primary:active:not(:disabled) {
    background: var(--color-primary-hover);
    border-color: var(--color-primary-hover);
    transform: translateY(1px);
  }

  /* Secondary */
  .btn--secondary {
    background: var(--color-background);
    color: var(--color-text-primary);
    border-color: var(--color-border);
  }

  .btn--secondary:hover:not(:disabled) {
    background: var(--color-background-subtle);
    border-color: var(--color-text-muted);
  }

  .btn--secondary:active:not(:disabled) {
    background: var(--color-background-subtle);
    transform: translateY(1px);
  }

  /* Sizes */
  .btn--sm {
    padding: var(--space-1) var(--space-3);
    font-size: var(--text-xs);
  }

  .btn--lg {
    padding: var(--space-3) var(--space-6);
    font-size: var(--text-base);
  }

  /* Ghost */
  .btn--ghost {
    background: transparent;
    color: var(--color-primary);
    border-color: transparent;
  }

  .btn--ghost:hover:not(:disabled) {
    background: var(--color-background-subtle);
  }

  .btn--ghost:active:not(:disabled) {
    background: var(--color-background-subtle);
    transform: translateY(1px);
  }

  /* Danger */
  .btn--danger {
    background: var(--color-danger);
    color: white;
    border-color: var(--color-danger);
  }

  .btn--danger:hover:not(:disabled) {
    background: #b91c1c;
    border-color: #b91c1c;
  }

  .btn--danger:active:not(:disabled) {
    background: #b91c1c;
    border-color: #b91c1c;
    transform: translateY(1px);
  }

  /* Spinner */
  .btn__spinner {
    position: absolute;
    width: 1em;
    height: 1em;
    border: 2px solid currentColor;
    border-right-color: transparent;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }

  .btn__content--hidden {
    visibility: hidden;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>