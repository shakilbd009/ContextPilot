<script lang="ts">
  import '../app.css';
  import type { Snippet } from 'svelte';
  import { page } from '$app/stores';
  import { Hamburger, Drawer } from '$lib/components/ui';
  import type { LayoutData } from './$types';

  interface Props {
    data: LayoutData;
    children: Snippet;
  }

  let { data, children }: Props = $props();

  let mobileMenuOpen = $state(false);

  const navLinks = [
    { href: '/', label: 'Dashboard' },
    { href: '/meetings', label: 'Meetings' },
    { href: '/upcoming', label: 'Upcoming', ffKey: 'VITE_FF_ENABLE_UPCOMING_MEETINGS' },
    { href: '/settings', label: 'Settings' },
  ];

  function isActive(href: string, current: string): boolean {
    if (href === '/') return current === '/';
    return current.startsWith(href);
  }

  function closeDrawer() {
    mobileMenuOpen = false;
  }
</script>

<svelte:head>
  <title>ContextPilot</title>
</svelte:head>

<!-- Skip navigation link (WCAG 2.1 AA) -->
<a href="#main-content" class="skip-nav">Skip to main content</a>

<!-- Mobile hamburger only shows below 640px -->
<div class="mobile-header">
  <Hamburger bind:open={mobileMenuOpen} aria-controls="mobile-drawer" />
</div>

<!-- Drawer for mobile nav -->
<Drawer bind:open={mobileMenuOpen} onclose={closeDrawer} id="mobile-drawer">
  {#each navLinks as link}
    {#if !link.ffKey || (typeof window !== 'undefined' && import.meta.env[link.ffKey] === 'true')}
      <a
        href={link.href}
        class="drawer-nav-link"
        class:drawer-nav-link--active={isActive(link.href, $page.url.pathname)}
        onclick={closeDrawer}
      >
        {link.label}
      </a>
    {/if}
  {/each}
</Drawer>

<!-- Desktop Header (always visible, hidden on mobile) -->
<header class="site-header">
  <div class="site-header__inner">
    <a href="/" class="site-header__logo" aria-label="ContextPilot home">
      <svg width="28" height="28" viewBox="0 0 28 28" fill="none" aria-hidden="true">
        <rect width="28" height="28" rx="6" fill="var(--color-primary)"/>
        <path d="M8 10h12M8 14h8M8 18h10" stroke="white" stroke-width="2" stroke-linecap="round"/>
      </svg>
      <span>ContextPilot</span>
    </a>

    <nav class="site-header__nav" aria-label="Main navigation">
      {#each navLinks as link}
        {#if !link.ffKey || (typeof window !== 'undefined' && import.meta.env[link.ffKey] === 'true')}
          <a
            href={link.href}
            class="site-header__nav-link"
            class:site-header__nav-link--active={isActive(link.href, $page.url.pathname)}
            aria-current={isActive(link.href, $page.url.pathname) ? 'page' : undefined}
          >
            {link.label}
          </a>
        {/if}
      {/each}
    </nav>

    <div class="site-header__user">
      <a href="/login" class="site-header__login-link">Sign in</a>
    </div>
  </div>
</header>

<!-- Main content area -->
<main id="main-content" class="main-content">
  {@render children()}
</main>

<style>
  /* ── Mobile header (hamburger only) ── */
  .mobile-header {
    display: none;
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: 56px;
    padding: var(--space-2) var(--space-4);
    background: var(--color-background);
    border-bottom: 1px solid var(--color-border);
    z-index: 30;
    align-items: center;
  }

  /* ── Desktop header ── */
  .site-header {
    display: flex;
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: 56px;
    background: var(--color-background);
    border-bottom: 1px solid var(--color-border);
    z-index: 20;
  }

  .site-header__inner {
    display: flex;
    align-items: center;
    gap: var(--space-6);
    max-width: 1200px;
    width: 100%;
    margin: 0 auto;
    padding: 0 var(--space-6);
  }

  .site-header__logo {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    text-decoration: none;
    color: var(--color-text-primary);
    font-size: var(--text-lg);
    font-weight: 700;
    flex-shrink: 0;
  }

  .site-header__logo:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 4px;
    border-radius: var(--radius-sm);
  }

  .site-header__nav {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    flex: 1;
  }

  .site-header__nav-link {
    padding: var(--space-2) var(--space-3);
    border-radius: var(--radius-sm);
    text-decoration: none;
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-text-secondary);
    transition: color var(--transition-fast), background var(--transition-fast);
  }

  .site-header__nav-link:hover {
    color: var(--color-text-primary);
    background: var(--color-background-subtle);
  }

  .site-header__nav-link--active {
    color: var(--color-primary);
  }

  .site-header__nav-link:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  .site-header__user {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    margin-left: auto;
  }

  .site-header__login-link {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-primary);
    text-decoration: none;
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-sm);
  }

  .site-header__login-link:hover {
    text-decoration: underline;
  }

  .site-header__login-link:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  /* ── Drawer nav links ── */
  :global(.drawer-nav-link) {
    display: block;
    padding: var(--space-3) var(--space-4);
    border-radius: var(--radius-md);
    text-decoration: none;
    font-size: var(--text-base);
    font-weight: 500;
    color: var(--color-text-primary);
    transition: background var(--transition-fast), color var(--transition-fast);
  }

  :global(.drawer-nav-link:hover) {
    background: var(--color-background-subtle);
  }

  :global(.drawer-nav-link--active) {
    color: var(--color-primary);
    background: rgb(26 86 219 / 0.08);
  }

  :global(.drawer-nav-link:focus-visible) {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  /* ── Main content ── */
  .main-content {
    padding-top: 56px; /* header height */
    min-height: 100vh;
  }

  /* ── Responsive ── */
  @media (max-width: 639px) {
    .mobile-header {
      display: flex;
    }

    .site-header {
      display: none;
    }

    .main-content {
      padding-top: 56px;
    }
  }

  @media (min-width: 640px) {
    .mobile-header {
      display: none;
    }

    .site-header {
      display: flex;
    }
  }
</style>