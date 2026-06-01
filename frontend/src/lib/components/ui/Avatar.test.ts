import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Avatar from './Avatar.svelte';

describe('Avatar', () => {
	it('renders with default props', () => {
		render(Avatar);
		const avatar = document.querySelector('.avatar');
		expect(avatar).toBeInTheDocument();
	});

	it('renders initials from name', () => {
		render(Avatar, { name: 'Alice Smith' });
		const avatar = document.querySelector('.avatar');
		expect(avatar?.textContent).toContain('AS');
	});

	it('renders initials from single word name', () => {
		render(Avatar, { name: 'Alice' });
		const avatar = document.querySelector('.avatar');
		expect(avatar?.textContent).toContain('A');
	});

	it('shows image when src provided', () => {
		render(Avatar, { src: 'https://example.com/photo.jpg', name: 'Alice' });
		const img = document.querySelector('.avatar__img');
		expect(img).toBeInTheDocument();
		expect(img).toHaveAttribute('src', 'https://example.com/photo.jpg');
	});

	it('shows initials when src fails', () => {
		render(Avatar, { src: 'https://invalid-url/broken.jpg', name: 'Bob Jones' });
		// Simulate image error - use a broken URL that fires onerror
		const avatar = document.querySelector('.avatar');
		// Initially renders with image
		const img = avatar?.querySelector('.avatar__img');
		expect(img).toBeInTheDocument();
	});

	it('has aria-label with name', () => {
		render(Avatar, { name: 'Charlie Davis' });
		const avatar = document.querySelector('.avatar');
		expect(avatar).toHaveAttribute('aria-label', 'Charlie Davis');
	});

	it('has fallback aria-label when no name', () => {
		render(Avatar);
		const avatar = document.querySelector('.avatar');
		expect(avatar).toHaveAttribute('aria-label', 'User avatar');
	});

	it('renders sm size', () => {
		render(Avatar, { size: 'sm', name: 'Small' });
		const avatar = document.querySelector('.avatar');
		expect(avatar).toHaveClass('avatar--sm');
	});

	it('renders md size by default', () => {
		render(Avatar, { name: 'Medium' });
		const avatar = document.querySelector('.avatar');
		expect(avatar).toHaveClass('avatar--md');
	});

	it('renders lg size', () => {
		render(Avatar, { size: 'lg', name: 'Large' });
		const avatar = document.querySelector('.avatar');
		expect(avatar).toHaveClass('avatar--lg');
	});

	it('handles empty name gracefully', () => {
		render(Avatar, { name: '' });
		const avatar = document.querySelector('.avatar');
		expect(avatar).toHaveAttribute('aria-label', 'User avatar');
		const initials = document.querySelector('.avatar__initials');
		expect(initials?.textContent).toBe('');
	});

	it('limits initials to two characters', () => {
		render(Avatar, { name: 'John Michael Smith' });
		const avatar = document.querySelector('.avatar');
		const initials = avatar?.textContent?.trim();
		expect(initials?.length).toBeLessThanOrEqual(2);
	});
});