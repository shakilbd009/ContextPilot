<script lang="ts">
  import type { Snippet } from 'svelte';

  interface DrawerProps {
    open: boolean;
    onclose: () => void;
    id?: string;
    children: Snippet;
  }

  let { open = $bindable(false), onclose, id, children }: DrawerProps = $props();

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onclose();
    }
  }

  function handleBackdropClick() {
    onclose();
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <!-- Backdrop -->
  <div
    class="drawer-backdrop"
    aria-hidden="true"
    onclick={handleBackdropClick}
    role="presentation"
  ></div>

  <!-- Drawer panel -->
  <div
    class="drawer"
    role="dialog"
    aria-modal="true"
    aria-label="Navigation menu"
    {id}
  >
    <div class="drawer__header">
      <span class="drawer__title">Menu</span>
      <button
        type="button"
        class="drawer__close"
        aria-label="Close menu"
        onclick={onclose}
      >
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none" aria-hidden="true">
          <path d="M5 5l10 10M15 5L5 15" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      </button>
    </div>
    <nav class="drawer__nav" aria-label="Main navigation">
      {@render children()}
    </nav>
  </div>
{/if}

<style>
  .drawer-backdrop {
    position: fixed;
    inset: 0;
    background: rgb(0 0 0 / 0.4);
    z-index: 40;
  }

  .drawer {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: 280px;
    max-width: 80vw;
    background: var(--color-background);
    box-shadow: var(--shadow-lg);
    z-index: 50;
    display: flex;
    flex-direction: column;
    animation: slideIn var(--transition-base) ease;
  }

  @keyframes slideIn {
    from { transform: translateX(-100%); }
    to { transform: translateX(0); }
  }

  .drawer__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-4);
    border-bottom: 1px solid var(--color-border);
  }

  .drawer__title {
    font-size: var(--text-lg);
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .drawer__close {
    background: none;
    border: none;
    cursor: pointer;
    color: var(--color-text-secondary);
    padding: var(--space-1);
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .drawer__close:hover {
    background: var(--color-background-subtle);
    color: var(--color-text-primary);
  }

  .drawer__close:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  .drawer__nav {
    display: flex;
    flex-direction: column;
    padding: var(--space-4);
    gap: var(--space-1);
  }
</style>