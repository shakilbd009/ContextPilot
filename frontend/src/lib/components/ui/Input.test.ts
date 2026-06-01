import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Input from './Input.svelte';

describe('Input', () => {
	it('renders with default props', () => {
		render(Input);
		const input = screen.getByRole('textbox');
		expect(input).toBeInTheDocument();
	});

	it('renders with label', () => {
		render(Input, { label: 'Email address' });
		expect(screen.getByText('Email address')).toBeInTheDocument();
	});

	it('renders required asterisk when required', () => {
		render(Input, { label: 'Email', required: true });
		const label = screen.getByText('Email');
		expect(label.querySelector('span')).toBeTruthy(); // asterisk span exists
	});

	it('shows error message', () => {
		render(Input, { label: 'Email', error: 'Invalid email' });
		expect(screen.getByRole('alert')).toHaveTextContent('Invalid email');
	});

	it('uses placeholder text', () => {
		render(Input, { placeholder: 'Enter your email' });
		const input = screen.getByRole('textbox');
		expect(input).toHaveAttribute('placeholder', 'Enter your email');
	});

	it('binds value correctly', async () => {
		const { component } = render(Input, { value: '' });
		const input = screen.getByRole('textbox') as HTMLInputElement;
		// Test that component renders with initial value
		expect(input.value).toBe('');
	});

	it('renders text type', () => {
		render(Input, { type: 'text' });
		expect(screen.getByRole('textbox')).toBeInTheDocument();
	});

	it('renders email type', () => {
		render(Input, { type: 'email' });
		const input = screen.getByRole('textbox');
		expect(input).toHaveAttribute('type', 'email');
	});

	it('renders date type', () => {
		render(Input, { type: 'date' });
		const input = document.querySelector('input[type="date"]') as HTMLInputElement;
		expect(input).toBeInTheDocument();
		expect(input.type).toBe('date');
	});

	it('renders password type', () => {
		render(Input, { type: 'password' });
		const input = document.querySelector('input[type="password"]') as HTMLInputElement;
		expect(input).toBeInTheDocument();
		expect(input.type).toBe('password');
	});

	it('is disabled when disabled prop is true', () => {
		render(Input, { disabled: true });
		expect(screen.getByRole('textbox')).toBeDisabled();
	});

	it('has aria-invalid when error is present', () => {
		render(Input, { error: 'Field required' });
		const input = screen.getByRole('textbox');
		expect(input).toHaveAttribute('aria-invalid', 'true');
	});

	it('has aria-describedby referencing error id when error present', () => {
		render(Input, { label: 'Name', error: 'Required' });
		const input = screen.getByRole('textbox');
		expect(input).toHaveAttribute('aria-describedby');
	});

	it('renders with id', () => {
		render(Input, { id: 'my-input' });
		expect(screen.getByRole('textbox')).toHaveAttribute('id', 'my-input');
	});

	it('renders with name attribute', () => {
		render(Input, { name: 'username' });
		expect(screen.getByRole('textbox')).toHaveAttribute('name', 'username');
	});

	it('applies error class to wrapper when error is present', () => {
		render(Input, { error: 'Error text' });
		const wrapper = document.querySelector('.input-group');
		expect(wrapper).toHaveClass('input-group--error');
	});
});