import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import Button from './Button.svelte';

describe('Button', () => {
	it('renders with default props', () => {
		render(Button, { children: () => 'Click me' });
		const btn = screen.getByRole('button');
		expect(btn).toBeInTheDocument();
		expect(btn).toHaveClass('btn');
		expect(btn).toHaveClass('btn--primary');
		expect(btn).toHaveClass('btn--md');
	});

	it('renders primary variant', () => {
		render(Button, { variant: 'primary', children: () => 'Primary' });
		expect(screen.getByRole('button')).toHaveClass('btn--primary');
	});

	it('renders secondary variant', () => {
		render(Button, { variant: 'secondary', children: () => 'Secondary' });
		expect(screen.getByRole('button')).toHaveClass('btn--secondary');
	});

	it('renders ghost variant', () => {
		render(Button, { variant: 'ghost', children: () => 'Ghost' });
		expect(screen.getByRole('button')).toHaveClass('btn--ghost');
	});

	it('renders danger variant', () => {
		render(Button, { variant: 'danger', children: () => 'Danger' });
		expect(screen.getByRole('button')).toHaveClass('btn--danger');
	});

	it('renders sm size', () => {
		render(Button, { size: 'sm', children: () => 'Small' });
		expect(screen.getByRole('button')).toHaveClass('btn--sm');
	});

	it('renders lg size', () => {
		render(Button, { size: 'lg', children: () => 'Large' });
		expect(screen.getByRole('button')).toHaveClass('btn--lg');
	});

	it('is disabled when disabled prop is true', () => {
		render(Button, { disabled: true, children: () => 'Disabled' });
		expect(screen.getByRole('button')).toBeDisabled();
	});

	it('is disabled when loading prop is true', () => {
		render(Button, { loading: true, children: () => 'Loading' });
		expect(screen.getByRole('button')).toBeDisabled();
	});

	it('shows spinner when loading', () => {
		render(Button, { loading: true, children: () => 'Loading' });
		const spinner = document.querySelector('.btn__spinner');
		expect(spinner).toBeInTheDocument();
	});

	it('hides content when loading', () => {
		render(Button, { loading: true, children: () => 'Loading' });
		const content = document.querySelector('.btn__content');
		expect(content).toHaveClass('btn__content--hidden');
	});

	it('calls onclick handler when clicked', async () => {
		const handler = vi.fn();
		render(Button, { onclick: handler, children: () => 'Click me' });
		await fireEvent.click(screen.getByRole('button'));
		expect(handler).toHaveBeenCalledTimes(1);
	});

	it('does not call onclick when disabled', async () => {
		const handler = vi.fn();
		render(Button, { disabled: true, onclick: handler, children: () => 'Disabled' });
		await fireEvent.click(screen.getByRole('button'));
		expect(handler).not.toHaveBeenCalled();
	});

	it('renders submit type', () => {
		render(Button, { type: 'submit', children: () => 'Submit' });
		expect(screen.getByRole('button')).toHaveAttribute('type', 'submit');
	});

	it('renders reset type', () => {
		render(Button, { type: 'reset', children: () => 'Reset' });
		expect(screen.getByRole('button')).toHaveAttribute('type', 'reset');
	});

	it('applies custom class', () => {
		render(Button, { class: 'custom-class', children: () => 'Custom' });
		expect(screen.getByRole('button')).toHaveClass('custom-class');
	});
});