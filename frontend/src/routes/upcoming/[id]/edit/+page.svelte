<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { Button, Card, Alert, Input, Spinner } from '$lib/components/ui';
  import {
    getUpcomingMeeting,
    updateUpcomingMeeting,
    type UpcomingMeeting,
  } from '$lib/api/upcoming';

  let { data }: { data: { meetingId: string } } = $props();

  const ffEnabled = import.meta.env.VITE_FF_ENABLE_UPCOMING_MEETINGS === 'true';
  const meetingId = $derived(data.meetingId ?? $page.params.id);

  let meeting = $state<UpcomingMeeting | null>(null);
  let loading = $state(true);
  let saving = $state(false);
  let serverErrors = $state<Record<string, string>>({});
  let serverErrorSummary = $state('');
  let notFound = $state(false);

  // ── Form state ────────────────────────────────────────────────
  let title = $state('');
  let scheduledDate = $state('');
  let scheduledTime = $state('');
  let description = $state('');
  let clientOrOrganization = $state('');
  let participants = $state<Array<{ displayName: string; email: string; organization: string }>>([]);

  let showAdvanced = $state(false);
  let timeRemaining = $state<number | null>(null);

  // ── Edit-window countdown (FR-9) ─────────────────────────────
  const editWindowRemaining = $derived(() => {
    if (!meeting) return null;
    const deadline = new Date(new Date(meeting.scheduledStart).getTime() + 15 * 60 * 1000);
    return Math.max(0, deadline.getTime() - Date.now());
  });

  const isEditable = $derived(() => editWindowRemaining() > 0);

  // Live countdown tick
  $effect(() => {
    if (!meeting || meeting.status === 'cancelled') {
      timeRemaining = null;
      return;
    }
    timeRemaining = editWindowRemaining();
    const id = setInterval(() => {
      timeRemaining = editWindowRemaining();
      if (timeRemaining === 0) clearInterval(id);
    }, 1000);
    return () => clearInterval(id);
  });

  function formatCountdown(ms: number): string {
    const totalSec = Math.floor(ms / 1000);
    const h = Math.floor(totalSec / 3600);
    const m = Math.floor((totalSec % 3600) / 60);
    const s = totalSec % 60;
    if (h > 0) return `${h}h ${m}m remaining`;
    if (m > 0) return `${m}m ${s}s remaining`;
    return `${s}s remaining`;
  }

  function defaultParticipant() {
    return { displayName: '', email: '', organization: '' };
  }

  async function loadMeeting() {
    loading = true;
    serverErrorSummary = '';
    const res = await getUpcomingMeeting(meetingId, fetch);
    loading = false;
    if (res.ok) {
      meeting = res.data;
      // Pre-fill form
      title = res.data.title;
      const start = new Date(res.data.scheduledStart);
      scheduledDate = start.toISOString().slice(0, 10);
      scheduledTime = start.toISOString().slice(11, 16);
      description = res.data.description ?? '';
      clientOrOrganization = res.data.clientOrOrganization ?? '';
      participants = res.data.participants?.length
        ? res.data.participants.map(p => ({
            displayName: p.displayName ?? '',
            email: p.email ?? '',
            organization: p.organization ?? '',
          }))
        : [defaultParticipant()];
      // isEditable derived from editWindowRemaining() — no manual assignment needed
    } else if (res.status === 404 || res.data.error === 'not_found') {
      notFound = true;
    } else {
      serverErrorSummary = res.data.message ?? 'Failed to load meeting.';
    }
  }

  const canSubmit = $derived(
    isEditable &&
    title.trim().length > 0 &&
    scheduledDate.length > 0 &&
    scheduledTime.length > 0 &&
    !saving
  );

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!canSubmit) return;

    saving = true;
    serverErrors = {};
    serverErrorSummary = '';

    const scheduledStart = `${scheduledDate}T${scheduledTime}:00Z`;

    const payload = {
      title: title.trim(),
      scheduledStart,
      description: description.trim() || undefined,
      clientOrOrganization: clientOrOrganization.trim() || undefined,
      participants: participants
        .filter(p => p.displayName.trim() || p.email.trim())
        .map(p => ({
          displayName: p.displayName.trim() || undefined,
          email: p.email.trim() || undefined,
          organization: p.organization.trim() || undefined,
        })),
    };

    try {
      const res = await updateUpcomingMeeting(meetingId, payload, fetch);
      if (res.ok) {
        goto(`/upcoming/${meetingId}`);
        return;
      }

      if (res.status === 422 && res.data.error === 'not_editable') {
          serverErrorSummary = 'Edit window has passed. This meeting is now read-only.';
        } else if (res.status === 400 && res.data.errors) {
        const errs: Record<string, string> = {};
        for (const err of res.data.errors) {
          errs[err.field] = err.message;
        }
        serverErrors = errs;
        if (res.data.message) serverErrorSummary = res.data.message;
      } else {
        serverErrorSummary = res.data.message ?? 'Failed to update meeting.';
      }
    } catch {
      serverErrorSummary = 'Network error. Please check your connection and try again.';
    } finally {
      saving = false;
    }
  }

  function addParticipant() {
    participants = [...participants, defaultParticipant()];
  }

  function removeParticipant(index: number) {
    if (participants.length <= 1) return;
    participants = participants.filter((_, i) => i !== index);
  }

  function cancel() {
    goto(`/upcoming/${meetingId}`);
  }

  $effect(() => {
    if (ffEnabled) loadMeeting();
  });
</script>

<svelte:head>
  <title>{meeting ? `Edit: ${meeting.title}` : 'Edit Meeting'} — ContextPilot</title>
</svelte:head>

{#if !ffEnabled}
  <div class="page-container">
    <Alert variant="warning">
      Upcoming meetings are not enabled. Set <code>VITE_FF_ENABLE_UPCOMING_MEETINGS=true</code> to enable them.
    </Alert>
  </div>
{:else if loading}
  <div class="page-container">
    <div class="loading-state" aria-label="Loading meeting" aria-busy="true">
      <Spinner />
      <span>Loading meeting&hellip;</span>
    </div>
  </div>
{:else if notFound}
  <div class="page-container">
    <Alert variant="error">Meeting not found.</Alert>
    <Button variant="ghost" onclick={() => goto('/upcoming')}>← Back to upcoming meetings</Button>
  </div>
{:else if meeting && !isEditable()}
  <div class="page-container">
    <Alert variant="warning">
      Edit window has passed. This meeting is now read-only.
    </Alert>
    <Button variant="secondary" onclick={() => goto(`/upcoming/${meetingId}`)}>View meeting details</Button>
  </div>
{:else if meeting}
  <div class="page-container">
    <header class="page-header">
      <h1 class="page-title">Edit Meeting</h1>
      {#if timeRemaining !== null && timeRemaining > 0}
        <span class="countdown-badge" aria-label="Edit window time remaining">
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
            <circle cx="6" cy="6" r="5" stroke="currentColor" stroke-width="1.5"/>
            <path d="M6 3.5v2.5l1.5 1.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
          {formatCountdown(timeRemaining)}
        </span>
      {/if}
    </header>

    {#if serverErrorSummary}
      <Alert variant="error" class="top-alert">{serverErrorSummary}</Alert>
    {/if}

    <Card>
      <form
        id="edit-upcoming-meeting-form"
        onsubmit={handleSubmit}
        novalidate
        class="meeting-form"
      >
        <!-- ── Title ─────────────────────────────────────── -->
        <Input
          id="title"
          name="title"
          label="Meeting title"
          type="text"
          placeholder="Q3 Planning Review"
          bind:value={title}
          error={serverErrors.title}
          required
        />

        <!-- ── Scheduled date & time ─────────────────────── -->
        <div class="date-time-row">
          <Input
            id="scheduledDate"
            name="scheduledDate"
            label="Date"
            type="date"
            bind:value={scheduledDate}
            error={serverErrors.scheduledStart}
            required
          />
          <Input
            id="scheduledTime"
            name="scheduledTime"
            label="Time"
            type="time"
            bind:value={scheduledTime}
            error={serverErrors.scheduledStart}
            required
          />
        </div>

        <!-- ── Description ────────────────────────────────── -->
        <div class="textarea-group">
          <label for="description" class="input-label">Description / agenda <span class="optional-label">(optional)</span></label>
          <textarea
            id="description"
            name="description"
            placeholder="What will this meeting cover?"
            bind:value={description}
            rows="4"
            class="textarea"
          ></textarea>
        </div>

        <!-- ── Client / Organization ───────────────────────── -->
        <Input
          id="clientOrOrganization"
          name="clientOrOrganization"
          label="Client or organization"
          type="text"
          placeholder="Acme Corp"
          bind:value={clientOrOrganization}
        />

        <!-- ── Participants ─────────────────────────────── -->
        <fieldset class="participants-section">
          <legend class="section-legend">
            Participants <span class="optional-label">(optional)</span>
          </legend>

          <div class="participants-list" aria-label="Participant list">
            {#each participants as participant, i}
              <div class="participant-row">
                <div class="participant-row__main">
                  <Input
                    id="participants[{i}].displayName"
                    name="participants[{i}].displayName"
                    label={i === 0 ? 'Display name' : undefined}
                    type="text"
                    placeholder="Alice Smith"
                    bind:value={participant.displayName}
                  />
                  {#if participants.length > 1}
                    <button
                      type="button"
                      class="remove-participant-btn"
                      onclick={() => removeParticipant(i)}
                      aria-label="Remove participant {i + 1}"
                    >
                      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
                        <path d="M4 8h8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                      </svg>
                    </button>
                  {/if}
                </div>

                {#if showAdvanced}
                  <div class="participant-row__advanced">
                    <Input
                      id="participants[{i}].email"
                      name="participants[{i}].email"
                      label="Email (optional)"
                      type="email"
                      placeholder="alice@example.com"
                      bind:value={participant.email}
                      error={serverErrors[`participants[${i}].email`]}
                    />
                    <Input
                      id="participants[{i}].organization"
                      name="participants[{i}].organization"
                      label="Organization (optional)"
                      type="text"
                      placeholder="Acme Corp"
                      bind:value={participant.organization}
                      error={serverErrors[`participants[${i}].organization`]}
                    />
                  </div>
                {/if}
              </div>
            {/each}
          </div>

          {#if participants.length < 50}
            <button type="button" class="add-participant-btn" onclick={addParticipant}>
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
                <path d="M7 2v10M2 7h10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
              Add participant
            </button>
          {/if}

          <button
            type="button"
            class="toggle-advanced-btn"
            onclick={() => (showAdvanced = !showAdvanced)}
            aria-expanded={showAdvanced}
          >
            {showAdvanced ? 'Hide' : 'Show'} email and organization
          </button>
        </fieldset>

        <!-- ── Actions ─────────────────────────────────── -->
        <div class="form-actions">
          <Button variant="secondary" type="button" onclick={cancel}>
            Cancel
          </Button>
          <Button
            variant="primary"
            type="submit"
            loading={saving}
            disabled={!canSubmit || saving}
          >
            Save changes
          </Button>
        </div>
      </form>
    </Card>
  </div>
{/if}

<style>
  .page-container {
    max-width: 640px;
    margin: 0 auto;
    padding: var(--space-8) var(--space-4);
  }

  .page-header {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    margin-bottom: var(--space-6);
  }

  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--color-text-primary);
    margin: 0;
  }

  .top-alert {
    margin-bottom: var(--space-3);
  }

  .loading-text {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }

  .countdown-badge {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: var(--text-xs);
    color: var(--color-warning);
    font-weight: 500;
  }

  .meeting-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }

  .date-time-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-4);
  }

  .textarea-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .input-label {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-text-primary);
  }

  .optional-label {
    font-weight: 400;
    color: var(--color-text-muted);
  }

  .textarea {
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    font-family: var(--font-sans);
    font-size: var(--text-base);
    color: var(--color-text-primary);
    background: var(--color-background);
    resize: vertical;
    width: 100%;
    transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
  }

  .textarea:focus {
    outline: none;
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px rgb(26 86 219 / 0.15);
  }

  /* ── Participants ─ */
  .participants-section {
    border: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .section-legend {
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-text-primary);
    padding: 0;
    margin-bottom: var(--space-1);
  }

  .participants-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .participant-row {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-3);
    background: var(--color-background-subtle);
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border);
  }

  .participant-row__main {
    display: flex;
    align-items: flex-end;
    gap: var(--space-2);
  }

  .participant-row__main :global(.input-group) {
    flex: 1;
  }

  .remove-participant-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-background);
    color: var(--color-text-secondary);
    cursor: pointer;
    flex-shrink: 0;
    transition: color var(--transition-fast), border-color var(--transition-fast);
    margin-bottom: 1px;
  }

  .remove-participant-btn:hover {
    color: var(--color-danger);
    border-color: var(--color-danger);
  }

  .remove-participant-btn:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }

  .participant-row__advanced {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-3);
    margin-top: var(--space-2);
    padding-top: var(--space-3);
    border-top: 1px solid var(--color-border);
  }

  .add-participant-btn {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-primary);
    background: none;
    border: none;
    cursor: pointer;
    padding: var(--space-1) 0;
    font-family: var(--font-sans);
  }

  .add-participant-btn:hover {
    text-decoration: underline;
  }

  .add-participant-btn:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
    border-radius: var(--radius-xs);
  }

  .toggle-advanced-btn {
    display: inline-flex;
    align-items: center;
    font-size: var(--text-sm);
    font-weight: 400;
    color: var(--color-text-muted);
    background: none;
    border: none;
    cursor: pointer;
    padding: var(--space-1) 0;
    font-family: var(--font-sans);
  }

  .toggle-advanced-btn:hover {
    color: var(--color-text-secondary);
  }

  .toggle-advanced-btn:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
    border-radius: var(--radius-xs);
  }

  /* ── Actions ── */
  .form-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: var(--space-3);
    padding-top: var(--space-2);
    border-top: 1px solid var(--color-border);
  }
</style>