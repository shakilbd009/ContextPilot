import { describe, it, expect } from 'vitest';
import { safeRedirect } from './safeRedirect';

describe('safeRedirect', () => {
  // --- rejects absolute URLs (attacker-controlled hosts) ---
  it('rejects https://evil.com', () => expect(safeRedirect('https://evil.com')).toBe('/'));
  it('rejects http://evil.com', () => expect(safeRedirect('http://evil.com')).toBe('/'));
  it('rejects https://evil.com/fake-help', () => expect(safeRedirect('https://evil.com/fake-help')).toBe('/'));
  it('rejects https://evil.com?param=x', () => expect(safeRedirect('https://evil.com?param=x')).toBe('/'));
  it('rejects https://evil.com#anchor', () => expect(safeRedirect('https://evil.com#anchor')).toBe('/'));
  it('rejects https://contextpilot.com/login', () => expect(safeRedirect('https://contextpilot.com/login')).toBe('/'));

  // --- rejects protocol-relative URLs ---
  it('rejects //evil.com (protocol-relative)', () => expect(safeRedirect('//evil.com')).toBe('/'));
  it('rejects //evil.com/login-capture', () => expect(safeRedirect('//evil.com/login-capture')).toBe('/'));
  it('rejects //evil.com?redirect=x', () => expect(safeRedirect('//evil.com?redirect=x')).toBe('/'));

  // --- rejects backslash protocol-relative bypasses ---
  it('rejects /\\evil.com (backslash protocol-relative)', () => expect(safeRedirect('/\\evil.com')).toBe('/'));
  it('rejects /\\evil.com/path', () => expect(safeRedirect('/\\evil.com/path')).toBe('/'));

  // --- rejects javascript: and data: URIs ---
  it('rejects javascript:alert(1)', () => expect(safeRedirect('javascript:alert(1)')).toBe('/'));
  it('rejects data:text/html,...', () => expect(safeRedirect('data:text/html,<h1>x</h1>')).toBe('/'));

  // --- rejects bare hostnames ---
  it('rejects evil.com (bare hostname)', () => expect(safeRedirect('evil.com')).toBe('/'));
  it('rejects login.contextpilot.com (bare hostname)', () => expect(safeRedirect('login.contextpilot.com')).toBe('/'));

  // --- rejects non-string inputs ---
  it('returns / for null', () => expect(safeRedirect(null)).toBe('/'));
  it('returns / for undefined', () => expect(safeRedirect(undefined)).toBe('/'));
  it('returns / for 0', () => expect(safeRedirect(0 as unknown as string)).toBe('/'));
  it('returns / for false', () => expect(safeRedirect(false as unknown as string)).toBe('/'));

  // --- accepts legitimate same-origin paths ---
  it('accepts / (root)', () => expect(safeRedirect('/')).toBe('/'));
  it('accepts /meetings', () => expect(safeRedirect('/meetings')).toBe('/meetings'));
  it('accepts /meetings/abc123', () => expect(safeRedirect('/meetings/abc123')).toBe('/meetings/abc123'));
  it('accepts /meetings with query string', () => expect(safeRedirect('/meetings?id=1')).toBe('/meetings?id=1'));
  it('accepts /meetings with hash', () => expect(safeRedirect('/meetings#section')).toBe('/meetings#section'));
  it('accepts /meetings?q=1#anchor', () => expect(safeRedirect('/meetings?q=1#anchor')).toBe('/meetings?q=1#anchor'));
  it('accepts /upcoming with deep path', () => expect(safeRedirect('/upcoming/new')).toBe('/upcoming/new'));

  // --- default when no input ---
  it('returns / when no param provided', () => expect(safeRedirect('')).toBe('/'));
});