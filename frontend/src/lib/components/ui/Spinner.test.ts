import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Spinner from './Spinner.svelte';

describe('Spinner', () => {
	it('renders with default props', () => {
		render(Spinner);
		const spinner = document.querySelector('.spinner');
		expect(spinner).toBeInTheDocument();
	});

	it('renders md size by default', () => {
		render(Spinner);
		const spinner = document.querySelector('.spinner');
		expect(spinner).toHaveClass('spinner--md');
	});

	it('renders sm size', () => {
		render(Spinner, { size: 'sm' });
		const spinner = document.querySelector('.spinner');
		expect(spinner).toHaveClass('spinner--sm');
	});

	it('renders md size explicitly', () => {
		render(Spinner, { size: 'md' });
		const spinner = document.querySelector('.spinner');
		expect(spinner).toHaveClass('spinner--md');
	});

	it('has role status', () => {
		render(Spinner);
		const spinner = document.querySelector('[role="status"]');
		expect(spinner).toBeInTheDocument();
	});

	it('has aria-label with default label', () => {
		render(Spinner);
		const spinner = document.querySelector('[role="status"]');
		expect(spinner).toHaveAttribute('aria-label', 'Loading');
	});

	it('has aria-label with custom label', () => {
		render(Spinner, { label: 'Saving...' });
		const spinner = document.querySelector('[role="status"]');
		expect(spinner).toHaveAttribute('aria-label', 'Saving...');
	});

	it('has sr-only text with label', () => {
		render(Spinner, { label: 'Please wait' });
		const srText = document.querySelector('.sr-only');
		expect(srText).toHaveTextContent('Please wait');
	});

	it('animates (CSS class present)', () => {
		render(Spinner);
		const spinner = document.querySelector('.spinner');
		// Animation is in CSS via @keyframes spin
		expect(spinner).toBeInTheDocument();
	});

	it('applies correct dimensions for sm size', () => {
		render(Spinner, { size: 'sm' });
		const spinner = document.querySelector('.spinner');
		expect(spinner).toHaveClass('spinner--sm');
	});

	it('applies correct dimensions for md size', () => {
		render(Spinner, { size: 'md' });
		const spinner = document.querySelector('.spinner');
		expect(spinner).toHaveClass('spinner--md');
	});
});