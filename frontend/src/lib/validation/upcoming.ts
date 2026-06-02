/**
 * Input validation for upcoming meeting text fields.
 *
 * Defense-in-depth for CWE-20 (Improper Input Validation).
 *
 * The dashboard XSS sink (frontend/src/routes/+page.svelte) is the
 * immediate fix; this module is the permanent fix. Even if the
 * client-side render path is hardened today, a future `{@html}` or
 * `innerHTML` regression on a stored title/description would resurrect
 * the same vulnerability. Validating at the API boundary means stored
 * data is always safe to render with any HTML-aware sink.
 *
 * Rules:
 *  - Length caps per field (see FIELD_LIMITS).
 *  - Reject ASCII control characters U+0000–U+001F except TAB, LF, CR.
 *  - Reject HTML tag-shaped content: any `<` followed by a
 *    non-whitespace character. Catches the full PoC suite
 *    (<script>, <img onerror>, <svg onload>, </script>) while leaving
 *    legitimate text like "Q&A discussion", "C++ deep-dive", and
 *    "1 < 2 is obvious" alone.
 *
 * Pure functions — no I/O, no side effects, deterministic. Used by
 * both POST /api/upcoming and PATCH /api/upcoming/[id].
 */

export interface FieldError {
  field: string;
  message: string;
}

export interface ParticipantInput {
  displayName?: unknown;
  email?: unknown;
  organization?: unknown;
}

export interface UpcomingPayload {
  title?: unknown;
  scheduledStart?: unknown;
  description?: unknown;
  clientOrOrganization?: unknown;
  participants?: unknown;
}

// Length caps per finding F5 of assessment t_793ea842.
export const FIELD_LIMITS = {
  title: 200,
  description: 5000,
  clientOrOrganization: 200,
  participantDisplayName: 100,
  participantOrganization: 200,
  participantEmail: 320, // RFC 5321
} as const;

// U+0000–U+001F except TAB (U+0009), LF (U+000A), CR (U+000D).
const CONTROL_CHAR_REGEX = /[\u0000-\u0008\u000B\u000C\u000E-\u001F]/;

// HTML tag detection. Any `<` followed by a non-whitespace character
// starts an HTML tag (or close-tag, e.g. `</script>`). Catches every
// stored-XSS vector in the PoC suite from assessment t_793ea842:
//   <script>alert(1)</script>
//   <img src=x onerror=alert(1)>
//   <svg onload=alert(1)>
//   </script>
//
// Legitimate text like "1 < 2 is obvious" passes because `<` is
// followed by whitespace. Titles that have no `<` at all — "Q&A
// discussion", "C++ deep-dive", "C++ vs JavaScript: a side-by-side" —
// pass naturally.
//
// We deliberately do NOT block the `javascript:` URI scheme as a
// substring. It would over-restrict legitimate sentences that mention
// "JavaScript:" as part of a phrase, and the stored-XSS vector is the
// HTML-tag form (which this regex catches), not a free-text mention.
// Any `javascript:` URI in a URL-bearing attribute (href, src) is a
// browser-side concern handled by the render path.
const XSS_MARKER_REGEX = /<\S/;

const STRING_LENGTH = (s: string): number => s.length;

/**
 * Validate a string field. Returns an error if the value is a string
 * that fails any of: type check, length cap, control character, or
 * XSS marker. Non-string types other than null/undefined are rejected.
 */
function validateStringField(
  fieldName: string,
  value: unknown,
  options: { maxLength: number; checkXss: boolean; required: boolean }
): FieldError[] {
  const errors: FieldError[] = [];

  if (value === undefined || value === null) {
    if (options.required) {
      errors.push({ field: fieldName, message: `${fieldName} is required.` });
    }
    return errors;
  }

  if (typeof value !== 'string') {
    errors.push({ field: fieldName, message: `${fieldName} must be a string.` });
    return errors;
  }

  if (options.required && value.trim().length === 0) {
    errors.push({ field: fieldName, message: `${fieldName} is required.` });
  }

  if (STRING_LENGTH(value) > options.maxLength) {
    errors.push({
      field: fieldName,
      message: `${fieldName} must be ${options.maxLength} characters or fewer.`,
    });
  }

  if (CONTROL_CHAR_REGEX.test(value)) {
    errors.push({
      field: fieldName,
      message: `${fieldName} contains forbidden control characters.`,
    });
  }

  if (options.checkXss && XSS_MARKER_REGEX.test(value)) {
    errors.push({
      field: fieldName,
      message: `${fieldName} contains forbidden content (HTML tag).`,
    });
  }

  return errors;
}

/**
 * Validate an email field. Allows empty (email is optional) but
 * enforces length, control characters, and a basic format check.
 */
function validateEmailField(
  fieldName: string,
  value: unknown
): FieldError[] {
  const errors: FieldError[] = [];

  if (value === undefined || value === null) {
    return errors; // email is optional on participants
  }

  if (typeof value !== 'string') {
    errors.push({ field: fieldName, message: `${fieldName} must be a string.` });
    return errors;
  }

  // Treat empty string as "not provided" — not an error.
  if (value.trim().length === 0) {
    return errors;
  }

  if (STRING_LENGTH(value) > FIELD_LIMITS.participantEmail) {
    errors.push({
      field: fieldName,
      message: `${fieldName} must be ${FIELD_LIMITS.participantEmail} characters or fewer.`,
    });
  }

  if (CONTROL_CHAR_REGEX.test(value)) {
    errors.push({
      field: fieldName,
      message: `${fieldName} contains forbidden control characters.`,
    });
  }

  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
    errors.push({ field: fieldName, message: 'Invalid email format.' });
  }

  return errors;
}

/**
 * Validate a single participant entry. Returns errors with field
 * names like `participants[0].displayName`.
 */
function validateParticipant(
  index: number,
  participant: unknown
): FieldError[] {
  const errors: FieldError[] = [];

  if (participant === null || typeof participant !== 'object') {
    errors.push({
      field: `participants[${index}]`,
      message: 'Participant must be an object.',
    });
    return errors;
  }

  const p = participant as ParticipantInput;
  const base = `participants[${index}]`;

  // displayName: XSS marker check applies because we render this text.
  errors.push(
    ...validateStringField(`${base}.displayName`, p.displayName, {
      maxLength: FIELD_LIMITS.participantDisplayName,
      checkXss: true,
      required: false,
    })
  );

  // organization: free text — XSS check applies.
  errors.push(
    ...validateStringField(`${base}.organization`, p.organization, {
      maxLength: FIELD_LIMITS.participantOrganization,
      checkXss: true,
      required: false,
    })
  );

  // email: separate validator (format check, optional).
  errors.push(...validateEmailField(`${base}.email`, p.email));

  // Identity requirement: must have a non-blank displayName OR email.
  const hasName = typeof p.displayName === 'string' && p.displayName.trim().length > 0;
  const hasEmail = typeof p.email === 'string' && p.email.trim().length > 0;
  if (!hasName && !hasEmail) {
    errors.push({
      field: base,
      message: 'Participant must have a display name or email.',
    });
  }

  return errors;
}

/**
 * Validate the full POST /api/upcoming payload.
 *
 * Returns a list of field errors. Empty list means valid.
 */
export function validateCreateUpcoming(body: unknown): FieldError[] {
  const errors: FieldError[] = [];

  if (body === null || typeof body !== 'object') {
    return [{ field: 'body', message: 'Request body must be a JSON object.' }];
  }

  const payload = body as UpcomingPayload;

  // title is required and free-text — apply XSS check.
  errors.push(
    ...validateStringField('title', payload.title, {
      maxLength: FIELD_LIMITS.title,
      checkXss: true,
      required: true,
    })
  );

  // description is optional free-text — apply XSS check.
  errors.push(
    ...validateStringField('description', payload.description, {
      maxLength: FIELD_LIMITS.description,
      checkXss: true,
      required: false,
    })
  );

  // clientOrOrganization is optional free-text — apply XSS check.
  errors.push(
    ...validateStringField('clientOrOrganization', payload.clientOrOrganization, {
      maxLength: FIELD_LIMITS.clientOrOrganization,
      checkXss: true,
      required: false,
    })
  );

  // scheduledStart: required, must be ISO date string (format and
  // "must be ≥15 min in future" checks are time-dependent and live in
  // the route handler; we only enforce the type and reject control
  // characters here).
  if (payload.scheduledStart === undefined || payload.scheduledStart === null) {
    errors.push({ field: 'scheduledStart', message: 'Scheduled start is required.' });
  } else if (typeof payload.scheduledStart !== 'string') {
    errors.push({ field: 'scheduledStart', message: 'Scheduled start must be an ISO 8601 string.' });
  } else if (CONTROL_CHAR_REGEX.test(payload.scheduledStart)) {
    errors.push({
      field: 'scheduledStart',
      message: 'Scheduled start contains forbidden control characters.',
    });
  }

  // participants: optional array.
  if (payload.participants !== undefined && payload.participants !== null) {
    if (!Array.isArray(payload.participants)) {
      errors.push({ field: 'participants', message: 'Participants must be an array.' });
    } else {
      if (payload.participants.length > 50) {
        errors.push({ field: 'participants', message: 'Too many participants. Maximum is 50.' });
      }
      for (let i = 0; i < payload.participants.length; i++) {
        errors.push(...validateParticipant(i, payload.participants[i]));
      }
    }
  }

  return errors;
}

/**
 * Validate the PATCH /api/upcoming/[id] payload. Unlike POST, every
 * field is optional on PATCH — we only validate fields the caller
 * actually provided. Required-for-non-empty rules (e.g. title cannot
 * be blank when supplied) still apply.
 */
export function validateUpdateUpcoming(body: unknown): FieldError[] {
  const errors: FieldError[] = [];

  if (body === null || typeof body !== 'object') {
    return [{ field: 'body', message: 'Request body must be a JSON object.' }];
  }

  const payload = body as UpcomingPayload;

  // title (optional, but if provided must not be blank).
  if (payload.title !== undefined) {
    errors.push(
      ...validateStringField('title', payload.title, {
        maxLength: FIELD_LIMITS.title,
        checkXss: true,
        required: true,
      })
    );
  }

  if (payload.description !== undefined) {
    errors.push(
      ...validateStringField('description', payload.description, {
        maxLength: FIELD_LIMITS.description,
        checkXss: true,
        required: false,
      })
    );
  }

  if (payload.clientOrOrganization !== undefined) {
    errors.push(
      ...validateStringField('clientOrOrganization', payload.clientOrOrganization, {
        maxLength: FIELD_LIMITS.clientOrOrganization,
        checkXss: true,
        required: false,
      })
    );
  }

  if (payload.scheduledStart !== undefined) {
    if (typeof payload.scheduledStart !== 'string') {
      errors.push({ field: 'scheduledStart', message: 'Scheduled start must be an ISO 8601 string.' });
    } else if (CONTROL_CHAR_REGEX.test(payload.scheduledStart)) {
      errors.push({
        field: 'scheduledStart',
        message: 'Scheduled start contains forbidden control characters.',
      });
    }
  }

  if (payload.participants !== undefined) {
    if (!Array.isArray(payload.participants)) {
      errors.push({ field: 'participants', message: 'Participants must be an array.' });
    } else {
      if (payload.participants.length > 50) {
        errors.push({ field: 'participants', message: 'Too many participants. Maximum is 50.' });
      }
      for (let i = 0; i < payload.participants.length; i++) {
        errors.push(...validateParticipant(i, payload.participants[i]));
      }
    }
  }

  return errors;
}
