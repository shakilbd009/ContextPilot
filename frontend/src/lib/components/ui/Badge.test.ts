import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Badge from './Badge.svelte';

describe('Badge', () => {
	it('renders with neutral variant by default', () => {
		render(Badge, { children: () => 'Badge' });
		const badge = document.querySelector('.badge');
		expect(badge).toBeInTheDocument();
		expect(badge).toHaveClass('badge--neutral');
	});

	it('renders success variant', () => {
		render(Badge, { variant: 'success', children: () => 'Active' });
		const badge = document.querySelector('.badge');
		expect(badge).toBeInTheDocument();
		expect(badge).toHaveClass('badge--success');
	});

	it('renders warning variant', () => {
		render(Badge, { variant: 'warning', children: () => 'Pending' });
		const badge = document.querySelector('.badge');
		expect(badge).toBeInTheDocument();
		expect(badge).toHaveClass('badge--warning');
	});

	it('renders danger variant', () => {
		render(Badge, { variant: 'danger', children: () => 'Error' });
		const badge = document.querySelector('.badge');
		expect(badge).toBeInTheDocument();
		expect(badge).toHaveClass('badge--danger');
	});

	it('renders neutral variant explicitly', () => {
		render(Badge, { variant: 'neutral', children: () => 'Neutral' });
		const badge = document.querySelector('.badge');
		expect(badge).toBeInTheDocument();
		expect(badge).toHaveClass('badge--neutral');
	});

	it('renders children content', () => {
		render(Badge, { children: () => 'Done' });
		const badge = document.querySelector('.badge');
		expect(badge).toBeInTheDocument();
	});
});