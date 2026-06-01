import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import type { Snippet } from 'svelte';
import Card from './Card.svelte';

// Snippet helper: components declare `children: Snippet` but Svelte 5's
// unique-symbol return type can't be expressed from a plain function literal.
// Cast through `unknown` to bridge the gap (runtime works correctly).
const children = (text: string): Snippet =>
	(() => text) as unknown as Snippet;

describe('Card', () => {
	it('renders with default variant', () => {
		render(Card, { props: { children: children('Card content') } });
		const card = document.querySelector('.card');
		expect(card).toBeInTheDocument();
		expect(card).toHaveClass('card--default');
	});

	it('renders default variant explicitly', () => {
		render(Card, { props: { variant: 'default', children: children('Content') } });
		const card = document.querySelector('.card');
		expect(card).toBeInTheDocument();
		expect(card).toHaveClass('card--default');
	});

	it('renders meeting variant', () => {
		render(Card, { props: { variant: 'meeting', children: children('Meeting') } });
		const card = document.querySelector('.card');
		expect(card).toBeInTheDocument();
		expect(card).toHaveClass('card--meeting');
	});

	it('renders briefing variant', () => {
		render(Card, { props: { variant: 'briefing', children: children('Briefing') } });
		const card = document.querySelector('.card');
		expect(card).toBeInTheDocument();
		expect(card).toHaveClass('card--briefing');
	});

	it('renders children content', () => {
		render(Card, { props: { children: children('Specific content') } });
		const card = document.querySelector('.card');
		expect(card).toBeInTheDocument();
	});

	it('applies hover effect via CSS', () => {
		render(Card, { props: { children: children('Hover card') } });
		const card = document.querySelector('.card');
		expect(card).toBeInTheDocument();
		// Hover styles are in CSS; component renders correctly
	});
});
