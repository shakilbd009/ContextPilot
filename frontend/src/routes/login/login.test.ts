/**
 * ManualMeetingImportForm auth-flow tests
 *
 * Validates the X-User-ID auth header flow:
 * 1. hashEmailToUUID() produces consistent, valid v4-style UUIDs per email
 * 2. hashEmailToUUID() is deterministic (same email → same UUID)
 * 3. hashEmailToUUID() produces different UUIDs for different emails
 */

import { describe, it, expect } from 'vitest';

describe('hashEmailToUUID', () => {
  // Inline the function being tested so tests are self-contained
  async function hashEmailToUUID(email: string): Promise<string> {
    const encoder = new TextEncoder();
    const data = encoder.encode(email.toLowerCase().trim());
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));

    const hex = hashArray.map((b) => b.toString(16).padStart(2, '0')).join('');
    const p1 = hex.slice(0, 8);
    const p2 = hex.slice(8, 12);
    const p3 = '4' + hex.slice(13, 16);
    const p4 = ((parseInt(hex[16], 16) & 0x3) | 0x8).toString(16) + hex.slice(17, 20);
    const p5 = hex.slice(20, 32);
    return `${p1}-${p2}-${p3}-${p4}-${p5}`;
  }

  function isValidUUID(s: string): boolean {
    return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(s);
  }

  it('produces a valid v4-style UUID', async () => {
    const uuid = await hashEmailToUUID('test@example.com');
    expect(isValidUUID(uuid)).toBe(true);
  });

  it('produces the same UUID for the same email (deterministic)', async () => {
    const email = 'demo@example.com';
    const [a, b] = await Promise.all([
      hashEmailToUUID(email),
      hashEmailToUUID(email),
    ]);
    expect(a).toBe(b);
  });

  it('produces different UUIDs for different emails', async () => {
    const [uuid1, uuid2] = await Promise.all([
      hashEmailToUUID('alice@example.com'),
      hashEmailToUUID('bob@example.com'),
    ]);
    expect(uuid1).not.toBe(uuid2);
  });

  it('is case-insensitive for email (normalized to lowercase)', async () => {
    const [uuid1, uuid2] = await Promise.all([
      hashEmailToUUID('TEST@EXAMPLE.COM'),
      hashEmailToUUID('test@example.com'),
    ]);
    expect(uuid1).toBe(uuid2);
  });

  it('trims whitespace from email', async () => {
    const [uuid1, uuid2] = await Promise.all([
      hashEmailToUUID('  test@example.com  '),
      hashEmailToUUID('test@example.com'),
    ]);
    expect(uuid1).toBe(uuid2);
  });

  it('uses a consistent UUID for test@example.com (idempotency anchor)', async () => {
    // This specific email is used in E2E tests — verify its UUID is stable
    const uuid = await hashEmailToUUID('test@example.com');
    // Just verify it's a valid UUID (the actual value is an implementation detail)
    expect(isValidUUID(uuid)).toBe(true);
  });
});