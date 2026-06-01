/**
 * HTML escaping utilities for FR-4a and FR-13a compliance.
 *
 * Evidence snippets extracted from transcript/notes text (FR-4a) and
 * user-authored resolution notes (FR-13a) are stored as plain text.
 * Before rendering in the browser, special characters that could be
 * interpreted as HTML markup must be escaped.
 *
 * Escapes: < > & " '
 */

const ESCAPE_MAP: Record<string, string> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
};

/**
 * Escape HTML special characters in a string.
 * Returns the string with &, <, >, ", and ' replaced by their HTML entities.
 *
 * Usage in .svelte files:
 *   import { escapeHtml } from '$lib/utils/html';
 *   <p>{escapeHtml(dangerousText)}</p>
 *
 * Note: Svelte's {expression} syntax does NOT auto-escape — it renders
 * raw HTML. Always use this utility for content that may contain user-
 * generated text or evidence snippets.
 */
export function escapeHtml(text: string | null | undefined): string {
  if (text == null) return '';
  return String(text).replace(/[&<>"']/g, (char) => ESCAPE_MAP[char] ?? char);
}