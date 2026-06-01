<script lang="ts">
  interface Props {
    src?: string;
    name?: string;
    size?: 'sm' | 'md' | 'lg';
  }

  let { src, name = '', size = 'md' }: Props = $props();

  const initials = $derived(
    name
      .split(' ')
      .map((n) => n[0])
      .slice(0, 2)
      .join('')
      .toUpperCase()
  );

  let imgError = $state(false);
</script>

<div class="avatar avatar--{size}" aria-label={name || 'User avatar'}>
  {#if src && !imgError}
    <img
      {src}
      alt=""
      class="avatar__img"
      onerror={() => (imgError = true)}
    />
  {:else}
    <span class="avatar__initials" aria-hidden="true">{initials}</span>
  {/if}
</div>

<style>
  .avatar {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-full);
    background: var(--color-primary);
    color: white;
    font-weight: 600;
    overflow: hidden;
    flex-shrink: 0;
  }

  .avatar--sm {
    width: 24px;
    height: 24px;
    font-size: var(--text-xs);
  }

  .avatar--md {
    width: 32px;
    height: 32px;
    font-size: var(--text-sm);
  }

  .avatar--lg {
    width: 48px;
    height: 48px;
    font-size: var(--text-base);
  }

  .avatar__img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .avatar__initials {
    line-height: 1;
  }
</style>