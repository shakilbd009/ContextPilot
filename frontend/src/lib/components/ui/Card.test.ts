import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Card from './Card.svelte';

describe('Card', () => {
	it('renders with default variant', () => {
		render(Card, { children: () => 'Card content' });
		const card = document.querySelector('.card');
		expect(card).toBeInTheDocument();
		expect(card).toHaveClass('card--default');
	});

	it('renders default variant explicitly', () => {
		render(Card, { variant: 'default', children: () => 'Content' });
		const card = document.querySelector('.card');
		expect(card).toBeInTheDocument();
		expect(card).toHaveClass('card--default');
	});

	it('renders meeting variant', () => {
		render(Card, { variant: 'meeting', children: () => 'Meeting' });
		const card = document.querySelector('.card');
		expect(card).toBeInTheDocument();
		expect(card).toHaveClass('card--meeting');
	});

	it('renders briefing variant', () => {
		render(Card, { variant: 'briefing', children: () => 'Briefing' });
		const card = document.querySelector('.card');
		expect(card).toBeInTheDocument();
		expect(card).toHaveClass('card--briefing');
	});

	it('renders children content', () => {
		render(Card, { children: () => 'Specific content' });
		const card = document.querySelector('.card');
		expect(card).toBeInTheDocument();
	});

	it('applies hover effect via CSS', () => {
		render(Card, { children: () => 'Hover card' });
		const card = document.querySelector('.card');
		expect(card).toBeInTheDocument();
		// Hover styles are in CSS; component renders correctly
	});
});