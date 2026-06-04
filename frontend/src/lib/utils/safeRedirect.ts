/**
 * Safe redirect — prevents open-redirect (CWE-601) attacks.
 *
 * Only allows same-origin path strings (must start with / and not //).
 * Rejects: absolute URLs, protocol-relative //, javascript:, data:, bare hostnames.
 * Accepts: same-origin paths with optional query string and/or hash fragment.
 */
export function safeRedirect(input: string | null | undefined): string {
  if (!input) return '/';
  if (typeof input !== 'string') return '/';

  // Reject protocol-relative URLs (e.g. //evil.com or /\evil.com)
  if (input.startsWith('//') || input.startsWith('/\\')) return '/';

  // Reject paths that don't start with /
  if (!input.startsWith('/')) return '/';

  // Reject javascript: and data: URIs (shouldn't be possible after the above,
  // but defense-in-depth)
  if (input.startsWith('javascript:') || input.startsWith('data:')) return '/';

  return input;
}