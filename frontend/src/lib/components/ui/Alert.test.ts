import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Alert from './Alert.svelte';

describe('Alert', () => {
	it('renders with info variant by default', () => {
		render(Alert, { children: () => 'Information message' });
		const alert = document.querySelector('.alert');
		expect(alert).toHaveClass('alert--info');
	});

	it('renders info variant explicitly', () => {
		render(Alert, { variant: 'info', children: () => 'Info' });
		expect(document.querySelector('.alert--info')).toBeInTheDocument();
	});

	it('renders success variant', () => {
		render(Alert, { variant: 'success', children: () => 'Success!' });
		expect(document.querySelector('.alert--success')).toBeInTheDocument();
	});

	it('renders warning variant', () => {
		render(Alert, { variant: 'warning', children: () => 'Warning!' });
		expect(document.querySelector('.alert--warning')).toBeInTheDocument();
	});

	it('renders error variant', () => {
		render(Alert, { variant: 'error', children: () => 'Error occurred' });
		expect(document.querySelector('.alert--error')).toBeInTheDocument();
	});

	it('renders children content', () => {
		render(Alert, { children: () => 'Alert content here' });
		const content = document.querySelector('.alert__content');
		expect(content).toBeInTheDocument();
	});

	it('has role alert', () => {
		render(Alert, { children: () => 'Important!' });
		expect(document.querySelector('[role="alert"]')).toBeInTheDocument();
	});

	it('shows dismiss button when dismissible is true', () => {
		render(Alert, { dismissible: true, children: () => 'Dismissible alert' });
		const dismissBtn = document.querySelector('.alert__dismiss');
		expect(dismissBtn).toBeInTheDocument();
	});

	it('hides alert after dismiss button is clicked', async () => {
		render(Alert, { dismissible: true, children: () => 'Will be dismissed' });
		const dismissBtn = document.querySelector('.alert__dismiss') as HTMLButtonElement;
		await fireEvent.click(dismissBtn);
		const alert = document.querySelector('.alert');
		expect(alert).not.toBeInTheDocument();
	});

	it('does not show dismiss button when dismissible is false', () => {
		render(Alert, { dismissible: false, children: () => 'Not dismissible' });
		expect(document.querySelector('.alert__dismiss')).not.toBeInTheDocument();
	});

	it('renders with custom class', () => {
		render(Alert, { class: 'my-custom-alert', children: () => 'Custom' });
		const alert = document.querySelector('.alert');
		expect(alert).toHaveClass('my-custom-alert');
	});

	it('displays icon for info variant', () => {
		render(Alert, { variant: 'info', children: () => 'Info' });
		const icon = document.querySelector('.alert__icon');
		expect(icon).toBeInTheDocument();
	});

	it('displays icon for success variant', () => {
		render(Alert, { variant: 'success', children: () => 'Success' });
		const icon = document.querySelector('.alert__icon');
		expect(icon).toBeInTheDocument();
	});

	it('displays icon for warning variant', () => {
		render(Alert, { variant: 'warning', children: () => 'Warning' });
		const icon = document.querySelector('.alert__icon');
		expect(icon).toBeInTheDocument();
	});

	it('displays icon for error variant', () => {
		render(Alert, { variant: 'error', children: () => 'Error' });
		const icon = document.querySelector('.alert__icon');
		expect(icon).toBeInTheDocument();
	});

	it('has accessible dismiss button with aria-label', () => {
		render(Alert, { dismissible: true, children: () => 'Dismissible' });
		const dismissBtn = document.querySelector('.alert__dismiss');
		expect(dismissBtn).toHaveAttribute('aria-label', 'Dismiss');
	});
});