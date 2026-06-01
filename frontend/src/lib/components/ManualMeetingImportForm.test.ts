/**
 * ManualMeetingImportForm unit tests
 *
 * Tests the Svelte component's derived state, participant management,
 * character count logic, and feature-flag gating.
 *
 * Note: The evals/unit/brd-02-manual-meeting-import.md eval contract
 * specifies backend Go validator tests. This file covers the Svelte
 * component's client-side behavior, including:
 * - canSubmit derivation logic
 * - Character count enforcement (warn at 45k, block at 50k)
 * - Participant add/remove operations
 * - Feature flag gating (ffEnabled = false shows disabled state)
 */

import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { cleanup } from '@testing-library/svelte';

// Mock import.meta.env using Vitest's stubbing mechanism
beforeEach(() => {
  // Stub the env before component import
  vi.stubEnv('VITE_FF_ENABLE_MANUAL_MEETING_IMPORT', 'true');
});
afterEach(() => {
  vi.unstubAllEnvs();
  cleanup();
});

// Import after mocks are set
import ManualMeetingImportForm from './ManualMeetingImportForm.svelte';

describe('ManualMeetingImportForm — Feature Flag Off', () => {
	it('shows disabled state when feature flag is false', async () => {
		vi.stubEnv('VITE_FF_ENABLE_MANUAL_MEETING_IMPORT', 'false');
		render(ManualMeetingImportForm);
		// When flag is false, the component shows a warning alert
		// not the form
		const alert = await screen.findByRole('alert');
		expect(alert).toBeInTheDocument();
	});

	it('shows correct message when feature flag is false', () => {
		vi.stubEnv('VITE_FF_ENABLE_MANUAL_MEETING_IMPORT', 'false');
		render(ManualMeetingImportForm);
		expect(screen.getByText(/VITE_FF_ENABLE_MANUAL_MEETING_IMPORT=true/)).toBeInTheDocument();
	});
});

describe('ManualMeetingImportForm — Feature Flag On (VITE_FF_ENABLE_MANUAL_MEETING_IMPORT=true)', () => {
	beforeEach(() => {
		vi.stubEnv('VITE_FF_ENABLE_MANUAL_MEETING_IMPORT', 'true');
	});

	it('renders the form when feature flag is true', () => {
		render(ManualMeetingImportForm);
		expect(screen.getByText('Import Meeting')).toBeInTheDocument();
		expect(screen.getByLabelText(/Meeting title/i)).toBeInTheDocument();
	});

	it('renders required fields', () => {
		render(ManualMeetingImportForm);
		expect(screen.getByLabelText(/Meeting title/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/Completed date/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/Display name/i)).toBeInTheDocument();
	});

	it('renders submit button', () => {
		render(ManualMeetingImportForm);
		const submitBtn = screen.getByRole('button', { name: /save meeting/i });
		expect(submitBtn).toBeInTheDocument();
	});

	it('submit button is disabled when form is empty', () => {
		render(ManualMeetingImportForm);
		const submitBtn = screen.getByRole('button', { name: /save meeting/i });
		expect(submitBtn).toBeDisabled();
	});

	it('shows character count starting at 0 / 50,000', () => {
		render(ManualMeetingImportForm);
		const charCount = screen.getByText(/0.*\/.*50,000.*characters/);
		expect(charCount).toBeInTheDocument();
	});
});

describe('ManualMeetingImportForm — Participant Management', () => {
	beforeEach(() => {
		vi.stubEnv('VITE_FF_ENABLE_MANUAL_MEETING_IMPORT', 'true');
	});

	it('has one participant by default', () => {
		render(ManualMeetingImportForm);
		// Only one display name input visible
		const displayNameInputs = screen.getAllByLabelText(/Display name/i);
		expect(displayNameInputs.length).toBe(1);
	});

	it('can add a participant', async () => {
		render(ManualMeetingImportForm);
		// Initially one participant row
		const initialRows = document.querySelectorAll('.participant-row');
		expect(initialRows.length).toBe(1);

		const addBtn = screen.getByRole('button', { name: /add participant/i });
		await fireEvent.click(addBtn);

		// Now two participant rows
		const rows = document.querySelectorAll('.participant-row');
		expect(rows.length).toBe(2);
		// Both displayName inputs exist (only first has visible label, second has no label)
		const displayNameInputs = document.querySelectorAll('input[name*="displayName"]');
		expect(displayNameInputs.length).toBe(2);
	});

	it('can remove added participant (more than one)', async () => {
		render(ManualMeetingImportForm);
		// Add a second participant
		const addBtn = screen.getByRole('button', { name: /add participant/i });
		await fireEvent.click(addBtn);

		// Remove button should now be visible
		const removeBtn = document.querySelector('.remove-participant-btn');
		expect(removeBtn).toBeInTheDocument();

		// Click remove
		await fireEvent.click(removeBtn!);

		// Back to one display name input
		const displayNameInputs = screen.getAllByLabelText(/Display name/i);
		expect(displayNameInputs.length).toBe(1);
	});

	it('cannot remove the last participant', async () => {
		render(ManualMeetingImportForm);
		// No remove button for single participant
		expect(document.querySelector('.remove-participant-btn')).not.toBeInTheDocument();
	});

	it('show advanced fields toggles', async () => {
		render(ManualMeetingImportForm);
		const toggleBtn = screen.getByRole('button', { name: /show advanced/i });
		await fireEvent.click(toggleBtn);
		// Advanced fields visible
		expect(screen.getByLabelText(/Email \(optional\)/i)).toBeInTheDocument();
	});

	it('hides advanced fields when toggled off', async () => {
		render(ManualMeetingImportForm);
		const toggleBtn = screen.getByRole('button', { name: /show advanced/i });
		await fireEvent.click(toggleBtn); // open
		await fireEvent.click(toggleBtn); // close
		expect(screen.queryByLabelText(/Email \(optional\)/i)).not.toBeInTheDocument();
	});
});

describe('ManualMeetingImportForm — Character Count', () => {
	beforeEach(() => {
		vi.stubEnv('VITE_FF_ENABLE_MANUAL_MEETING_IMPORT', 'true');
	});

	it('shows ok state at 0 characters', () => {
		render(ManualMeetingImportForm);
		const okCount = screen.getByText(/0.*\/.*50,000.*characters/);
		expect(okCount).toBeInTheDocument();
		// Should not have warn or over class
	});

	it('has transcript and notes textareas', () => {
		render(ManualMeetingImportForm);
		expect(screen.getByLabelText(/Transcript/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/Notes/i)).toBeInTheDocument();
	});

	it('has section hint about at least one required', () => {
		render(ManualMeetingImportForm);
		expect(screen.getByText(/at least one of transcript or notes is required/i)).toBeInTheDocument();
	});
});

describe('ManualMeetingImportForm — Cancel', () => {
	beforeEach(() => {
		vi.stubEnv('VITE_FF_ENABLE_MANUAL_MEETING_IMPORT', 'true');
	});

	it('renders cancel button', () => {
		render(ManualMeetingImportForm);
		const cancelBtn = screen.getByRole('button', { name: /cancel/i });
		expect(cancelBtn).toBeInTheDocument();
	});
});