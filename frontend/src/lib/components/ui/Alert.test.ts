import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import type { Snippet } from 'svelte';
import Alert from './Alert.svelte';

// Snippet helper: components declare `children: Snippet` but Svelte 5's
// unique-symbol return type can't be expressed from a plain function literal.
// Cast through `unknown` to bridge the gap (runtime works correctly).
const children = (text: string): Snippet =>
	(() => text) as unknown as Snippet;

describe('Alert', () => {
	it('renders with info variant by default', () => {
		render(Alert, { props: { children: children('Information message') } });
		const alert = document.querySelector('.alert');
		expect(alert).toHaveClass('alert--info');
	});

	it('renders info variant explicitly', () => {
		render(Alert, { props: { variant: 'info', children: children('Info') } });
		expect(document.querySelector('.alert--info')).toBeInTheDocument();
	});

	it('renders success variant', () => {
		render(Alert, { props: { variant: 'success', children: children('Success!') } });
		expect(document.querySelector('.alert--success')).toBeInTheDocument();
	});

	it('renders warning variant', () => {
		render(Alert, { props: { variant: 'warning', children: children('Warning!') } });
		expect(document.querySelector('.alert--warning')).toBeInTheDocument();
	});

	it('renders error variant', () => {
		render(Alert, { props: { variant: 'error', children: children('Error occurred') } });
		expect(document.querySelector('.alert--error')).toBeInTheDocument();
	});

	it('renders children content', () => {
		render(Alert, { props: { children: children('Alert content here') } });
		const content = document.querySelector('.alert__content');
		expect(content).toBeInTheDocument();
	});

	it('has role alert', () => {
		render(Alert, { props: { children: children('Important!') } });
		expect(document.querySelector('[role="alert"]')).toBeInTheDocument();
	});

	it('shows dismiss button when dismissible is true', () => {
		render(Alert, { props: { dismissible: true, children: children('Dismissible alert') } });
		const dismissBtn = document.querySelector('.alert__dismiss');
		expect(dismissBtn).toBeInTheDocument();
	});

	it('hides alert after dismiss button is clicked', async () => {
		render(Alert, { props: { dismissible: true, children: children('Will be dismissed') } });
		const dismissBtn = document.querySelector('.alert__dismiss') as HTMLButtonElement;
		await fireEvent.click(dismissBtn);
		const alert = document.querySelector('.alert');
		expect(alert).not.toBeInTheDocument();
	});

	it('does not show dismiss button when dismissible is false', () => {
		render(Alert, { props: { dismissible: false, children: children('Not dismissible') } });
		expect(document.querySelector('.alert__dismiss')).not.toBeInTheDocument();
	});

	it('renders with custom class', () => {
		render(Alert, { props: { class: 'my-custom-alert', children: children('Custom') } });
		const alert = document.querySelector('.alert');
		expect(alert).toHaveClass('my-custom-alert');
	});

	it('displays icon for info variant', () => {
		render(Alert, { props: { variant: 'info', children: children('Info') } });
		const icon = document.querySelector('.alert__icon');
		expect(icon).toBeInTheDocument();
	});

	it('displays icon for success variant', () => {
		render(Alert, { props: { variant: 'success', children: children('Success') } });
		const icon = document.querySelector('.alert__icon');
		expect(icon).toBeInTheDocument();
	});

	it('displays icon for warning variant', () => {
		render(Alert, { props: { variant: 'warning', children: children('Warning') } });
		const icon = document.querySelector('.alert__icon');
		expect(icon).toBeInTheDocument();
	});

	it('displays icon for error variant', () => {
		render(Alert, { props: { variant: 'error', children: children('Error') } });
		const icon = document.querySelector('.alert__icon');
		expect(icon).toBeInTheDocument();
	});

	it('has accessible dismiss button with aria-label', () => {
		render(Alert, { props: { dismissible: true, children: children('Dismissible') } });
		const dismissBtn = document.querySelector('.alert__dismiss');
		expect(dismissBtn).toHaveAttribute('aria-label', 'Dismiss');
	});
});
