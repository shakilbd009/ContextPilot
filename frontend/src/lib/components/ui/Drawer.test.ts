import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Drawer from './Drawer.svelte';

describe('Drawer', () => {
	it('does not render when open is false', () => {
		render(Drawer, { open: false, onclose: () => {} });
		expect(document.querySelector('.drawer')).not.toBeInTheDocument();
		expect(document.querySelector('.drawer-backdrop')).not.toBeInTheDocument();
	});

	it('renders when open is true', () => {
		render(Drawer, { open: true, onclose: () => {}, children: () => {} });
		expect(document.querySelector('.drawer')).toBeInTheDocument();
	});

	it('renders backdrop when open', () => {
		render(Drawer, { open: true, onclose: () => {}, children: () => {} });
		expect(document.querySelector('.drawer-backdrop')).toBeInTheDocument();
	});

	it('calls onclose when backdrop is clicked', async () => {
		const handler = vi.fn();
		render(Drawer, { open: true, onclose: handler, children: () => {} });
		const backdrop = document.querySelector('.drawer-backdrop') as HTMLDivElement;
		await fireEvent.click(backdrop);
		expect(handler).toHaveBeenCalledTimes(1);
	});

	it('calls onclose when close button is clicked', async () => {
		const handler = vi.fn();
		render(Drawer, { open: true, onclose: handler, children: () => {} });
		const closeBtn = document.querySelector('.drawer__close') as HTMLButtonElement;
		await fireEvent.click(closeBtn);
		expect(handler).toHaveBeenCalledTimes(1);
	});

	it('closes on Escape key', async () => {
		const handler = vi.fn();
		render(Drawer, { open: true, onclose: handler, children: () => {} });
		await fireEvent(document, new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
		expect(handler).toHaveBeenCalledTimes(1);
	});

	it('has role dialog', () => {
		render(Drawer, { open: true, onclose: () => {}, children: () => {} });
		expect(document.querySelector('[role="dialog"]')).toBeInTheDocument();
	});

	it('has aria-modal true', () => {
		render(Drawer, { open: true, onclose: () => {}, children: () => {} });
		expect(document.querySelector('[role="dialog"]')).toHaveAttribute('aria-modal', 'true');
	});

	it('has aria-label on dialog', () => {
		render(Drawer, { open: true, onclose: () => {}, children: () => {} });
		expect(document.querySelector('[role="dialog"]')).toHaveAttribute('aria-label', 'Navigation menu');
	});

	it('renders with custom id', () => {
		render(Drawer, { open: true, onclose: () => {}, id: 'my-drawer', children: () => {} });
		expect(document.querySelector('#my-drawer')).toBeInTheDocument();
	});

	it('renders children content', () => {
		render(Drawer, { open: true, onclose: () => {}, children: () => 'Drawer content' });
		const nav = document.querySelector('.drawer__nav');
		expect(nav).toBeInTheDocument();
	});

	it('has close button with aria-label', () => {
		render(Drawer, { open: true, onclose: () => {}, children: () => {} });
		const closeBtn = document.querySelector('.drawer__close');
		expect(closeBtn).toHaveAttribute('aria-label', 'Close menu');
	});

	it('has nav with aria-label', () => {
		render(Drawer, { open: true, onclose: () => {}, children: () => {} });
		const nav = document.querySelector('[aria-label="Main navigation"]');
		expect(nav).toBeInTheDocument();
	});

	it('renders drawer header with title', () => {
		render(Drawer, { open: true, onclose: () => {}, children: () => {} });
		expect(screen.getByText('Menu')).toBeInTheDocument();
	});
});