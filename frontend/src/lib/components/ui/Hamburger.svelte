<script lang="ts">
  interface HamburgerProps {
    open: boolean;
    'aria-controls'?: string;
  }

  let {
    open = $bindable(false),
    'aria-controls': ariaControls,
  }: HamburgerProps = $props();
</script>

<button
  type="button"
  class="hamburger"
  aria-label={open ? 'Close menu' : 'Open menu'}
  aria-expanded={open}
  aria-controls={ariaControls}
  onclick={() => (open = !open)}
>
  <span class="hamburger__bar" class:hamburger__bar--open={open}></span>
  <span class="hamburger__bar" class:hamburger__bar--open={open}></span>
  <span class="hamburger__bar" class:hamburger__bar--open={open}></span>
</button>

<style>
  .hamburger {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 5px;
    width: 36px;
    height: 36px;
    background: none;
    border: none;
    cursor: pointer;
    padding: var(--space-1);
    border-radius: var(--radius-sm);
  }

  .hamburger:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  .hamburger__bar {
    display: block;
    width: 100%;
    height: 2px;
    background: var(--color-text-primary);
    border-radius: 2px;
    transition: transform var(--transition-fast), opacity var(--transition-fast);
    transform-origin: center;
  }

  .hamburger__bar--open:nth-child(1) {
    transform: translateY(7px) rotate(45deg);
  }

  .hamburger__bar--open:nth-child(2) {
    opacity: 0;
  }

  .hamburger__bar--open:nth-child(3) {
    transform: translateY(-7px) rotate(-45deg);
  }
</style>