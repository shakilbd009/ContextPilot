import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import type { Snippet } from 'svelte';
import Button from './Button.svelte';

// Snippet helper: components declare `children: Snippet` but Svelte 5's
// unique-symbol return type can't be expressed from a plain function literal.
// Cast through `unknown` to bridge the gap (runtime works correctly).
const children = (text: string): Snippet =>
	(() => text) as unknown as Snippet;

describe('Button', () => {
	it('renders with default props', () => {
		render(Button, { props: { children: children('Click me') } });
		const btn = screen.getByRole('button');
		expect(btn).toBeInTheDocument();
		expect(btn).toHaveClass('btn');
		expect(btn).toHaveClass('btn--primary');
		expect(btn).toHaveClass('btn--md');
	});

	it('renders primary variant', () => {
		render(Button, { props: { variant: 'primary', children: children('Primary') } });
		expect(screen.getByRole('button')).toHaveClass('btn--primary');
	});

	it('renders secondary variant', () => {
		render(Button, { props: { variant: 'secondary', children: children('Secondary') } });
		expect(screen.getByRole('button')).toHaveClass('btn--secondary');
	});

	it('renders ghost variant', () => {
		render(Button, { props: { variant: 'ghost', children: children('Ghost') } });
		expect(screen.getByRole('button')).toHaveClass('btn--ghost');
	});

	it('renders danger variant', () => {
		render(Button, { props: { variant: 'danger', children: children('Danger') } });
		expect(screen.getByRole('button')).toHaveClass('btn--danger');
	});

	it('renders sm size', () => {
		render(Button, { props: { size: 'sm', children: children('Small') } });
		expect(screen.getByRole('button')).toHaveClass('btn--sm');
	});

	it('renders lg size', () => {
		render(Button, { props: { size: 'lg', children: children('Large') } });
		expect(screen.getByRole('button')).toHaveClass('btn--lg');
	});

	it('is disabled when disabled prop is true', () => {
		render(Button, { props: { disabled: true, children: children('Disabled') } });
		expect(screen.getByRole('button')).toBeDisabled();
	});

	it('is disabled when loading prop is true', () => {
		render(Button, { props: { loading: true, children: children('Loading') } });
		expect(screen.getByRole('button')).toBeDisabled();
	});

	it('shows spinner when loading', () => {
		render(Button, { props: { loading: true, children: children('Loading') } });
		const spinner = document.querySelector('.btn__spinner');
		expect(spinner).toBeInTheDocument();
	});

	it('hides content when loading', () => {
		render(Button, { props: { loading: true, children: children('Loading') } });
		const content = document.querySelector('.btn__content');
		expect(content).toHaveClass('btn__content--hidden');
	});

	it('calls onclick handler when clicked', async () => {
		const handler = vi.fn();
		render(Button, { props: { onclick: handler, children: children('Click me') } });
		await fireEvent.click(screen.getByRole('button'));
		expect(handler).toHaveBeenCalledTimes(1);
	});

	it('does not call onclick when disabled', async () => {
		const handler = vi.fn();
		render(Button, { props: { disabled: true, onclick: handler, children: children('Disabled') } });
		await fireEvent.click(screen.getByRole('button'));
		expect(handler).not.toHaveBeenCalled();
	});

	it('renders submit type', () => {
		render(Button, { props: { type: 'submit', children: children('Submit') } });
		expect(screen.getByRole('button')).toHaveAttribute('type', 'submit');
	});

	it('renders reset type', () => {
		render(Button, { props: { type: 'reset', children: children('Reset') } });
		expect(screen.getByRole('button')).toHaveAttribute('type', 'reset');
	});

	it('applies custom class', () => {
		render(Button, { props: { class: 'custom-class', children: children('Custom') } });
		expect(screen.getByRole('button')).toHaveClass('custom-class');
	});
});
