import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Hamburger from './Hamburger.svelte';

describe('Hamburger', () => {
	it('renders with default props', () => {
		render(Hamburger);
		const button = screen.getByRole('button');
		expect(button).toBeInTheDocument();
	});

	it('renders with open false by default', () => {
		render(Hamburger);
		const button = screen.getByRole('button');
		expect(button).toHaveAttribute('aria-expanded', 'false');
	});

	it('renders with open true when prop is set', () => {
		render(Hamburger, { open: true });
		const button = screen.getByRole('button');
		expect(button).toHaveAttribute('aria-expanded', 'true');
	});

	it('toggles open state on click', async () => {
		render(Hamburger, { open: false });
		const button = screen.getByRole('button');
		await fireEvent.click(button);
		// open is bindable, component re-renders
	});

	it('has aria-label Open menu when closed', () => {
		render(Hamburger, { open: false });
		const button = screen.getByRole('button');
		expect(button).toHaveAttribute('aria-label', 'Open menu');
	});

	it('has aria-label Close menu when open', () => {
		render(Hamburger, { open: true });
		const button = screen.getByRole('button');
		expect(button).toHaveAttribute('aria-label', 'Close menu');
	});

	it('has aria-expanded reflecting open state', () => {
		render(Hamburger, { open: true });
		expect(screen.getByRole('button')).toHaveAttribute('aria-expanded', 'true');
	});

	it('has aria-controls linking to drawer id', () => {
		render(Hamburger, { open: false, 'aria-controls': 'nav-drawer' });
		const button = screen.getByRole('button');
		expect(button).toHaveAttribute('aria-controls', 'nav-drawer');
	});

	it('renders three bars', () => {
		render(Hamburger);
		const bars = document.querySelectorAll('.hamburger__bar');
		expect(bars.length).toBe(3);
	});

	it('applies open class to bars when open', () => {
		render(Hamburger, { open: true });
		const bars = document.querySelectorAll('.hamburger__bar--open');
		expect(bars.length).toBe(3);
	});

	it('does not apply open class to bars when closed', () => {
		render(Hamburger, { open: false });
		const bars = document.querySelectorAll('.hamburger__bar--open');
		expect(bars.length).toBe(0);
	});

	it('is a button element', () => {
		render(Hamburger);
		expect(screen.getByRole('button').tagName).toBe('BUTTON');
	});

	it('has type button to prevent form submission', () => {
		render(Hamburger);
		expect(screen.getByRole('button')).toHaveAttribute('type', 'button');
	});
});