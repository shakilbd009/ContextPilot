import { describe, expect, it } from 'vitest';
import { isPublicRoute } from './hooks.server';

describe('hooks public route allowlist', () => {
  it('allows browser auth pages and same-origin auth API proxies before session checks', () => {
    expect(isPublicRoute('/login')).toBe(true);
    expect(isPublicRoute('/signup')).toBe(true);
    expect(isPublicRoute('/api/auth/login')).toBe(true);
    expect(isPublicRoute('/api/auth/signup')).toBe(true);
  });

  it('keeps application APIs protected by default', () => {
    expect(isPublicRoute('/api/upcoming')).toBe(false);
    expect(isPublicRoute('/meetings')).toBe(false);
    expect(isPublicRoute(undefined)).toBe(false);
  });
});