import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import type { Snippet } from 'svelte';
import Badge from './Badge.svelte';

// Snippet helper: components declare `children: Snippet` but Svelte 5's
// unique-symbol return type can't be expressed from a plain function literal.
// Cast through `unknown` to bridge the gap (runtime works correctly).
const children = (text: string): Snippet =>
	(() => text) as unknown as Snippet;

describe('Badge', () => {
	it('renders with neutral variant by default', () => {
		render(Badge, { props: { children: children('Badge') } });
		const badge = document.querySelector('.badge');
		expect(badge).toBeInTheDocument();
		expect(badge).toHaveClass('badge--neutral');
	});

	it('renders success variant', () => {
		render(Badge, { props: { variant: 'success', children: children('Active') } });
		const badge = document.querySelector('.badge');
		expect(badge).toBeInTheDocument();
		expect(badge).toHaveClass('badge--success');
	});

	it('renders warning variant', () => {
		render(Badge, { props: { variant: 'warning', children: children('Pending') } });
		const badge = document.querySelector('.badge');
		expect(badge).toBeInTheDocument();
		expect(badge).toHaveClass('badge--warning');
	});

	it('renders danger variant', () => {
		render(Badge, { props: { variant: 'danger', children: children('Error') } });
		const badge = document.querySelector('.badge');
		expect(badge).toBeInTheDocument();
		expect(badge).toHaveClass('badge--danger');
	});

	it('renders neutral variant explicitly', () => {
		render(Badge, { props: { variant: 'neutral', children: children('Neutral') } });
		const badge = document.querySelector('.badge');
		expect(badge).toBeInTheDocument();
		expect(badge).toHaveClass('badge--neutral');
	});

	it('renders children content', () => {
		render(Badge, { props: { children: children('Done') } });
		const badge = document.querySelector('.badge');
		expect(badge).toBeInTheDocument();
	});
});
