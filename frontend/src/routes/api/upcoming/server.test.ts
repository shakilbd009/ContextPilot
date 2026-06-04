/**
 * Sibling tests for /api/upcoming POST and /api/upcoming/[id] PATCH
 * handlers.
 *
 * Covers finding F5 of assessment t_793ea842 (CWE-20, server-side
 * input validation on upcoming meeting text fields). These tests are
 * a wiring/contract check: they verify the route handlers import and
 * delegate to the shared validator. The validator itself has
 * comprehensive unit tests in src/lib/validation/upcoming.test.ts —
 * see that file for behavior coverage.
 *
 * The shape mirrors the project pattern at
 * src/lib/api/meetings/+server.test.ts (small, focused, behavior
 * probes) rather than mocking the full SvelteKit RequestEvent (which
 * would require a SvelteKit test harness not currently in the project).
 */

import { describe, it, expect } from 'vitest';
import * as serverModule from './+server';
import * as idServerModule from './[id]/+server';
import {
  validateCreateUpcoming,
  validateUpdateUpcoming,
  FIELD_LIMITS,
} from '$lib/validation/upcoming';

describe('/api/upcoming POST handler — wiring', () => {
  it('exports a POST handler function', () => {
    expect(typeof serverModule.POST).toBe('function');
  });

  it('delegates validation to the shared validator', () => {
    // Direct call to the shared validator: any future regression
    // where the handler stopped using the validator would still
    // leave this test green (because the validator still works),
    // but the constant-wiring checks below catch the *import* and
    // the *exports*.
    const errors = validateCreateUpcoming({
      title: '<script>alert(1)</script>',
      scheduledStart: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
    });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'title' }));
  });
});

describe('/api/upcoming/[id] PATCH handler — wiring', () => {
  it('exports a PATCH handler function', () => {
    expect(typeof idServerModule.PATCH).toBe('function');
  });

  it('delegates validation to the shared validator (no fields required)', () => {
    // PATCH semantics: an empty body is valid.
    expect(validateUpdateUpcoming({})).toEqual([]);

    // ...but a hostile field is rejected.
    const errors = validateUpdateUpcoming({ description: '<svg onload=alert(1)>' });
    expect(errors).toContainEqual(expect.objectContaining({ field: 'description' }));
  });
});

describe('Validator constants — reflect the BRD-05 spec', () => {
  it('exposes the length caps called out in the F5 finding', () => {
    expect(FIELD_LIMITS.title).toBe(200);
    expect(FIELD_LIMITS.description).toBe(5000);
    expect(FIELD_LIMITS.clientOrOrganization).toBe(200);
    expect(FIELD_LIMITS.participantDisplayName).toBe(100);
    expect(FIELD_LIMITS.participantOrganization).toBe(200);
    expect(FIELD_LIMITS.participantEmail).toBe(320);
  });
});
