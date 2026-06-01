import { describe, it, expect } from 'vitest';
import { escapeHtml } from './html';

describe('escapeHtml', () => {
  it('returns empty string for null', () => {
    expect(escapeHtml(null)).toBe('');
  });

  it('returns empty string for undefined', () => {
    expect(escapeHtml(undefined)).toBe('');
  });

  it('returns empty string for null literal', () => {
    expect(escapeHtml(null as unknown as string)).toBe('');
  });

  it('leaves plain text unchanged', () => {
    expect(escapeHtml('Hello world')).toBe('Hello world');
    expect(escapeHtml('Meeting with team')).toBe('Meeting with team');
  });

  it('escapes ampersand', () => {
    expect(escapeHtml('Tom & Jerry')).toBe('Tom &amp; Jerry');
  });

  it('escapes less-than sign', () => {
    expect(escapeHtml('<script>')).toBe('&lt;script&gt;');
  });

  it('escapes greater-than sign', () => {
    expect(escapeHtml('1 > 0')).toBe('1 &gt; 0');
  });

  it('escapes double quotes', () => {
    expect(escapeHtml('say "hello"')).toBe('say &quot;hello&quot;');
  });

  it('escapes single quotes', () => {
    expect(escapeHtml("it's fine")).toBe('it&#39;s fine');
  });

  it('escapes all special characters together', () => {
    expect(escapeHtml('<script>alert("XSS")</script>')).toBe(
      '&lt;script&gt;alert(&quot;XSS&quot;)&lt;/script&gt;'
    );
  });

  it('escapes evidence snippet with HTML-like content', () => {
    expect(escapeHtml('We discussed <b>bold</b> decisions & "action items"')).toBe(
      'We discussed &lt;b&gt;bold&lt;/b&gt; decisions &amp; &quot;action items&quot;'
    );
  });

  it('handles XSS payload in resolution note', () => {
    expect(escapeHtml('<img src=x onerror=alert(1)>')).toBe(
      '&lt;img src=x onerror=alert(1)&gt;'
    );
  });

  it('handles multi-line text with special characters', () => {
    expect(escapeHtml('Line 1\nLine 2 <tag> & \'quote\'')).toBe(
      'Line 1\nLine 2 &lt;tag&gt; &amp; &#39;quote&#39;'
    );
  });

  it('converts non-string input to string', () => {
    expect(escapeHtml(123 as unknown as string)).toBe('123');
  });
});