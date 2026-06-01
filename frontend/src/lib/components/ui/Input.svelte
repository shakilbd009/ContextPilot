<script lang="ts">
  interface Props {
    label?: string;
    type?: 'text' | 'email' | 'password' | 'date' | 'time';
    placeholder?: string;
    value?: string;
    error?: string;
    disabled?: boolean;
    id?: string;
    name?: string;
    required?: boolean;
    min?: string;
    max?: string;
  }

  let {
    label,
    type = 'text',
    placeholder = '',
    value = $bindable(''),
    error = '',
    disabled = false,
    id,
    name,
    required = false,
    min,
    max,
  }: Props = $props();

  const inputId = $derived(id ?? `input-${Math.random().toString(36).slice(2, 8)}`);
  const errorId = $derived(`${inputId}-error`);
</script>

<div class="input-group" class:input-group--error={!!error}>
  {#if label}
    <label for={inputId} class="input-group__label">
      {label}
      {#if required}<span aria-hidden="true">*</span>{/if}
    </label>
  {/if}

  <input
    {type}
    id={inputId}
    {name}
    {placeholder}
    {disabled}
    {required}
    {min}
    {max}
    bind:value
    class="input-group__input"
    aria-invalid={!!error}
    aria-describedby={error ? errorId : undefined}
    autocomplete="off"
  />

  {#if error}
    <span id={errorId} class="input-group__error" role="alert">
      {error}
    </span>
  {/if}
</div>

<style>
  .input-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .input-group__label {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-text-primary);
  }

  .input-group__input {
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    font-family: var(--font-sans);
    font-size: var(--text-base);
    color: var(--color-text-primary);
    background: var(--color-background);
    transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
    width: 100%;
  }

  .input-group__input::placeholder {
    color: var(--color-text-muted);
  }

  .input-group__input:focus {
    outline: none;
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px rgb(26 86 219 / 0.15);
  }

  .input-group__input:disabled {
    background: var(--color-background-subtle);
    cursor: not-allowed;
    opacity: 0.6;
  }

  .input-group--error .input-group__input {
    border-color: var(--color-danger);
  }

  .input-group--error .input-group__input:focus {
    box-shadow: 0 0 0 3px rgb(220 38 38 / 0.15);
  }

  .input-group__error {
    font-size: var(--text-xs);
    color: var(--color-danger);
  }
</style>