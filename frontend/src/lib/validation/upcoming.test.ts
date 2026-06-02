/**
 * Unit tests for the upcoming meeting input validator.
 *
 * Covers finding F5 of assessment t_793ea842 (CWE-20, server-side
 * input validation on POST /api/upcoming text fields). These tests
 * document the security contract: any text field that ends up rendered
 * in the dashboard or detail page is bounded in length, free of
 * control characters, and free of HTML-tag-shaped content.
 *
 * Test groups:
 *  - Length caps per field
 *  - Control character rejection
 *  - HTML-tag rejection (catches <script, <img onerror, <svg onload)
 *  - Legitimate titles still pass (defense-in-depth, not over-blocking)
 *  - Email format / length / control chars
 *  - PATCH (update) treats every field as optional
 *  - Edge cases: wrong types, null, empty, whitespace-only
 */

import { describe, it, expect } from 'vitest';
import {
  validateCreateUpcoming,
  validateUpdateUpcoming,
  FIELD_LIMITS,
} from './upcoming';

const FUTURE = new Date(Date.now() + 60 * 60 * 1000).toISOString();

function validCreate(extra: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    title: 'Q3 Planning Review',
    scheduledStart: FUTURE,
    ...extra,
  };
}

// ── Length caps ───────────────────────────────────────────────────

describe('validateCreateUpcoming — length caps', () => {
  it('rejects title longer than 200 characters', () => {
    const errors = validateCreateUpcoming(validCreate({ title: 'a'.repeat(FIELD_LIMITS.title + 1) }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('accepts title at exactly 200 characters', () => {
    const errors = validateCreateUpcoming(validCreate({ title: 'a'.repeat(FIELD_LIMITS.title) }));
    expect(errors.filter((e) => e.field === 'title')).toEqual([]);
  });

  it('rejects description longer than 5000 characters', () => {
    const errors = validateCreateUpcoming(validCreate({ description: 'a'.repeat(FIELD_LIMITS.description + 1) }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'description' }));
  });

  it('accepts description at exactly 5000 characters', () => {
    const errors = validateCreateUpcoming(validCreate({ description: 'a'.repeat(FIELD_LIMITS.description) }));
    expect(errors.filter((e) => e.field === 'description')).toEqual([]);
  });

  it('rejects clientOrOrganization longer than 200 characters', () => {
    const errors = validateCreateUpcoming(validCreate({ clientOrOrganization: 'a'.repeat(FIELD_LIMITS.clientOrOrganization + 1) }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'clientOrOrganization' }));
  });

  it('rejects participant displayName longer than 100 characters', () => {
    const errors = validateCreateUpcoming(validCreate({
      participants: [{ displayName: 'a'.repeat(FIELD_LIMITS.participantDisplayName + 1) }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0].displayName' }));
  });

  it('rejects participant organization longer than 200 characters', () => {
    const errors = validateCreateUpcoming(validCreate({
      participants: [{ displayName: 'Alice', organization: 'a'.repeat(FIELD_LIMITS.participantOrganization + 1) }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0].organization' }));
  });

  it('rejects participant email longer than 320 characters', () => {
    const longEmail = 'a'.repeat(310) + '@example.com';
    const errors = validateCreateUpcoming(validCreate({
      participants: [{ displayName: 'Alice', email: longEmail }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0].email' }));
  });
});

// ── Control characters ────────────────────────────────────────────

describe('validateCreateUpcoming — control characters', () => {
  it.each([
    ['NUL (\\u0000)', '\u0000'],
    ['SOH (\\u0001)', '\u0001'],
    ['BEL (\\u0007)', '\u0007'],
    ['BS (\\u0008)', '\u0008'],
    ['VT (\\u000B)', '\u000B'],
    ['FF (\\u000C)', '\u000C'],
    ['SO (\\u000E)', '\u000E'],
    ['US (\\u001F)', '\u001F'],
  ])('rejects control character %s in title', (_label, char) => {
    const errors = validateCreateUpcoming(validCreate({ title: `Hello${char}World` }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('allows TAB (\\u0009) in description', () => {
    const errors = validateCreateUpcoming(validCreate({ description: 'line1\tline2' }));
    expect(errors.filter((e) => e.field === 'description')).toEqual([]);
  });

  it('allows LF (\\u000A) in description', () => {
    const errors = validateCreateUpcoming(validCreate({ description: 'line1\nline2' }));
    expect(errors.filter((e) => e.field === 'description')).toEqual([]);
  });

  it('allows CR (\\u000D) in description', () => {
    const errors = validateCreateUpcoming(validCreate({ description: 'line1\rline2' }));
    expect(errors.filter((e) => e.field === 'description')).toEqual([]);
  });

  it('rejects NUL in clientOrOrganization', () => {
    const errors = validateCreateUpcoming(validCreate({ clientOrOrganization: 'Acme\u0000Corp' }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'clientOrOrganization' }));
  });

  it('rejects control char in participant displayName', () => {
    const errors = validateCreateUpcoming(validCreate({
      participants: [{ displayName: 'Alice\u0000' }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0].displayName' }));
  });

  it('rejects control char in email', () => {
    const errors = validateCreateUpcoming(validCreate({
      participants: [{ displayName: 'Alice', email: 'alice\u0000@example.com' }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0].email' }));
  });
});

// ── HTML tag rejection (XSS prevention) ───────────────────────────

describe('validateCreateUpcoming — HTML tag rejection', () => {
  it('rejects <script> in title (the original PoC payload)', () => {
    const errors = validateCreateUpcoming(validCreate({ title: '<script>alert(1)</script>' }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('rejects <img onerror=...> in title (the other PoC payload)', () => {
    const errors = validateCreateUpcoming(validCreate({ title: '<img src=x onerror=alert(1)>' }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('rejects <svg onload=...> in title', () => {
    const errors = validateCreateUpcoming(validCreate({ title: '<svg onload=alert(1)>' }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('rejects <script in title (opening tag only)', () => {
    const errors = validateCreateUpcoming(validCreate({ title: 'prefix<script>suffix' }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('rejects </script> in title (closing tag only)', () => {
    const errors = validateCreateUpcoming(validCreate({ title: 'prefix</script>' }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('rejects <SCRIPT> in title (case-insensitive)', () => {
    const errors = validateCreateUpcoming(validCreate({ title: '<SCRIPT>alert(1)</SCRIPT>' }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('rejects <Img (mixed case) in title', () => {
    const errors = validateCreateUpcoming(validCreate({ title: '<Img src=x OnError=alert(1)>' }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('rejects <svg onload=...> in description', () => {
    const errors = validateCreateUpcoming(validCreate({ description: '<svg onload=alert(1)>' }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'description' }));
  });

  it('rejects <a href=...> in clientOrOrganization', () => {
    const errors = validateCreateUpcoming(validCreate({ clientOrOrganization: '<a href=javascript:bad>click</a>' }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'clientOrOrganization' }));
  });

  it('rejects XSS marker in participant displayName', () => {
    const errors = validateCreateUpcoming(validCreate({
      participants: [{ displayName: '<script>alert(1)</script>' }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0].displayName' }));
  });

  it('rejects XSS marker in participant organization', () => {
    const errors = validateCreateUpcoming(validCreate({
      participants: [{ displayName: 'Alice', organization: '<script>oops</script>' }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0].organization' }));
  });

  it('rejects <img tag in description', () => {
    const errors = validateCreateUpcoming(validCreate({ description: 'Look: <img src=foo.png>' }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'description' }));
  });
});

// ── Legitimate content is accepted ───────────────────────────────

describe('validateCreateUpcoming — legitimate content is accepted', () => {
  it('accepts "Q3 planning review"', () => {
    expect(validateCreateUpcoming(validCreate({ title: 'Q3 planning review' }))).toEqual([]);
  });

  it('accepts "Q&A discussion" (BRD-cited legit example)', () => {
    expect(validateCreateUpcoming(validCreate({ title: 'Q&A discussion' }))).toEqual([]);
  });

  it('accepts "C++ deep-dive" (BRD-cited legit example)', () => {
    expect(validateCreateUpcoming(validCreate({ title: 'C++ deep-dive' }))).toEqual([]);
  });

  it('accepts "1 < 2 is obvious" (less-than with whitespace after)', () => {
    expect(validateCreateUpcoming(validCreate({ title: '1 < 2 is obvious' }))).toEqual([]);
  });

  it('accepts "C++ vs JavaScript: a side-by-side" (no <, no javascript: alone)', () => {
    // Lowercased: "c++ vs javascript: a side-by-side" — contains the
    // substring "javascript:" but no `<` and we're explicitly NOT
    // blocking the URI scheme as a substring.
    expect(validateCreateUpcoming(validCreate({ title: 'C++ vs JavaScript: a side-by-side' }))).toEqual([]);
  });

  it('accepts description with ampersand and quotes', () => {
    expect(validateCreateUpcoming(validCreate({ description: 'Discussed "Tom & Jerry" priorities' }))).toEqual([]);
  });

  it('accepts multi-line description with newlines', () => {
    expect(validateCreateUpcoming(validCreate({ description: 'Line 1\nLine 2\nLine 3' }))).toEqual([]);
  });

  it('accepts a participant with displayName only', () => {
    expect(validateCreateUpcoming(validCreate({ participants: [{ displayName: 'Alice Smith' }] }))).toEqual([]);
  });

  it('accepts a participant with email only', () => {
    expect(validateCreateUpcoming(validCreate({ participants: [{ email: 'alice@example.com' }] }))).toEqual([]);
  });

  it('accepts valid email with plus and subdomain', () => {
    expect(validateCreateUpcoming(validCreate({
      participants: [{ displayName: 'Alice', email: 'alice.smith+work@example.co.uk' }],
    }))).toEqual([]);
  });

  it('accepts full payload with all fields populated', () => {
    const errors = validateCreateUpcoming({
      title: 'Q3 Planning Review',
      scheduledStart: FUTURE,
      description: 'Review Q3 roadmap priorities\nand confirm resource allocation.',
      clientOrOrganization: 'Acme Corp',
      participants: [
        { displayName: 'Alice Smith', email: 'alice@example.com', organization: 'Acme Corp' },
        { displayName: 'Bob Jones', email: 'bob@example.com', organization: 'Beta LLC' },
      ],
    });
    expect(errors).toEqual([]);
  });
});

// ── Email format ──────────────────────────────────────────────────

describe('validateCreateUpcoming — email format', () => {
  it('rejects email missing @', () => {
    const errors = validateCreateUpcoming(validCreate({ participants: [{ displayName: 'A', email: 'aliceexample.com' }] }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0].email' }));
  });

  it('rejects email missing domain', () => {
    const errors = validateCreateUpcoming(validCreate({ participants: [{ displayName: 'A', email: 'alice@' }] }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0].email' }));
  });

  it('rejects email with internal whitespace', () => {
    const errors = validateCreateUpcoming(validCreate({ participants: [{ displayName: 'A', email: 'ali ce@example.com' }] }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0].email' }));
  });

  it('treats null email as optional', () => {
    const errors = validateCreateUpcoming(validCreate({ participants: [{ displayName: 'A', email: null }] }));
    expect(errors.filter((e) => e.field === 'participants[0].email')).toEqual([]);
  });

  it('treats empty-string email as optional', () => {
    const errors = validateCreateUpcoming(validCreate({ participants: [{ displayName: 'A', email: '' }] }));
    expect(errors.filter((e) => e.field === 'participants[0].email')).toEqual([]);
  });
});

// ── Participant identity requirement ──────────────────────────────

describe('validateCreateUpcoming — participant identity', () => {
  it('rejects participant with neither displayName nor email', () => {
    const errors = validateCreateUpcoming(validCreate({
      participants: [{ organization: 'Acme' }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0]' }));
  });

  it('rejects participant with both blank', () => {
    const errors = validateCreateUpcoming(validCreate({
      participants: [{ displayName: '', email: '' }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0]' }));
  });

  it('accepts participant with whitespace-only name but valid email', () => {
    const errors = validateCreateUpcoming(validCreate({
      participants: [{ displayName: '   ', email: 'alice@example.com' }],
    }));
    expect(errors.filter((e) => e.field.startsWith('participants[0]'))).toEqual([]);
  });
});

// ── Type / shape edge cases ───────────────────────────────────────

describe('validateCreateUpcoming — type edge cases', () => {
  it('rejects non-object body', () => {
    expect(validateCreateUpcoming('not an object')).toContainEqual(expect.objectContaining({ field: 'body' }));
  });

  it('rejects null body', () => {
    expect(validateCreateUpcoming(null)).toContainEqual(expect.objectContaining({ field: 'body' }));
  });

  it('rejects title as a number', () => {
    const errors = validateCreateUpcoming({ title: 42, scheduledStart: FUTURE });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('rejects missing title', () => {
    const errors = validateCreateUpcoming({ scheduledStart: FUTURE });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('rejects whitespace-only title', () => {
    const errors = validateCreateUpcoming({ title: '   ', scheduledStart: FUTURE });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('rejects missing scheduledStart', () => {
    const errors = validateCreateUpcoming({ title: 'Meeting' });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'scheduledStart' }));
  });

  it('rejects scheduledStart as a number', () => {
    const errors = validateCreateUpcoming({ title: 'Meeting', scheduledStart: 12345 });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'scheduledStart' }));
  });

  it('rejects participants that is not an array', () => {
    const errors = validateCreateUpcoming({ ...validCreate(), participants: 'not an array' });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants' }));
  });

  it('rejects participant that is not an object', () => {
    const errors = validateCreateUpcoming(validCreate({ participants: ['not an object'] }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0]' }));
  });

  it('rejects more than 50 participants', () => {
    const many = Array.from({ length: 51 }, (_, i) => ({ displayName: `P${i}` }));
    const errors = validateCreateUpcoming(validCreate({ participants: many }));
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants' }));
  });

  it('accepts exactly 50 participants', () => {
    const many = Array.from({ length: 50 }, (_, i) => ({ displayName: `P${i}` }));
    expect(validateCreateUpcoming(validCreate({ participants: many }))).toEqual([]);
  });
});

// ── PATCH: every field is optional ───────────────────────────────

describe('validateUpdateUpcoming — fields are optional on PATCH', () => {
  it('accepts an empty object (no-op PATCH)', () => {
    expect(validateUpdateUpcoming({})).toEqual([]);
  });

  it('validates title when provided', () => {
    const errors = validateUpdateUpcoming({ title: '<script>x</script>' });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('rejects blank title when provided', () => {
    const errors = validateUpdateUpcoming({ title: '   ' });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });

  it('validates description when provided', () => {
    const errors = validateUpdateUpcoming({ description: '<svg onload=alert(1)>' });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'description' }));
  });

  it('validates description length when provided', () => {
    const errors = validateUpdateUpcoming({ description: 'a'.repeat(FIELD_LIMITS.description + 1) });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'description' }));
  });

  it('accepts description explicitly set to null', () => {
    // null is treated as "not provided" for optional fields.
    expect(validateUpdateUpcoming({ description: null })).toEqual([]);
  });

  it('validates clientOrOrganization when provided', () => {
    const errors = validateUpdateUpcoming({ clientOrOrganization: '<script>x</script>' });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'clientOrOrganization' }));
  });

  it('validates scheduledStart type when provided', () => {
    const errors = validateUpdateUpcoming({ scheduledStart: 12345 });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'scheduledStart' }));
  });

  it('validates participants array when provided', () => {
    const errors = validateUpdateUpcoming({
      participants: [{ displayName: '<script>x</script>' }],
    });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants[0].displayName' }));
  });

  it('rejects non-array participants', () => {
    const errors = validateUpdateUpcoming({ participants: 'oops' });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'participants' }));
  });

  it('accepts a legit PATCH payload', () => {
    expect(validateUpdateUpcoming({
      title: 'Renamed Meeting',
      description: 'New description',
    })).toEqual([]);
  });

  it('rejects non-object PATCH body', () => {
    expect(validateUpdateUpcoming(null)).toContainEqual(expect.objectContaining({ field: 'body' }));
  });
});
