<script lang="ts">
  import { goto } from '$app/navigation';
  import { Button, Card, Alert, Input } from '$lib/components/ui';
  import {
    createUpcomingMeeting,
    listUpcomingMeetings,
    type ParticipantInput,
  } from '$lib/api/upcoming';

  // ── Feature flag ──────────────────────────────────────────────
  const ffEnabled = import.meta.env.VITE_FF_ENABLE_UPCOMING_MEETINGS === 'true';

  // ── Types ─────────────────────────────────────────────────────
  interface Participant {
    displayName: string;
    email: string;
    organization: string;
  }

  // ── Form state ────────────────────────────────────────────────
  let title = $state('');
  let scheduledDate = $state('');
  let scheduledTime = $state('');
  let description = $state('');
  let clientOrOrganization = $state('');
  let participants = $state<Participant[]>([defaultParticipant()]);
  let showAdvanced = $state(false);
  let saving = $state(false);

  // ── Server-returned errors ────────────────────────────────────
  let serverErrors = $state<Record<string, string>>({});
  let serverErrorSummary = $state('');

  // ── Duplicate warning state (FR-24) ──────────────────────────
  let duplicateWarning = $state('');
  let checkingDuplicate = $state(false);

  function defaultParticipant(): Participant {
    return { displayName: '', email: '', organization: '' };
  }

  // ── Derived ──────────────────────────────────────────────────
  // Minimum date/time: 15 minutes from now
  const minDatetime = $derived(() => {
    const min = new Date(Date.now() + 15 * 60 * 1000);
    return min.toISOString().slice(0, 16);
  });

  const canSubmit = $derived(
    title.trim().length > 0 &&
    scheduledDate.length > 0 &&
    scheduledTime.length > 0 &&
    !saving
  );

  // Check for duplicate when title or scheduled time changes (FR-24)
  async function checkDuplicate() {
    if (!title.trim() || !scheduledDate || !scheduledTime) {
      duplicateWarning = '';
      return;
    }
    checkingDuplicate = true;
    try {
      const res = await listUpcomingMeetings(fetch, { status: 'scheduled' });
      if (res.ok) {
        const normalizedTitle = title.trim().toLowerCase().replace(/\s+/g, ' ');
        const proposedStart = new Date(`${scheduledDate}T${scheduledTime}`).toISOString();
        const proposedMinute = new Date(proposedStart).getTime();

        const dup = res.data.meetings.find(m => {
          const mNormalized = m.title.toLowerCase().replace(/\s+/g, ' ');
          const mMinute = new Date(m.scheduledStart).getTime();
          return mNormalized === normalizedTitle &&
            Math.abs(mMinute - proposedMinute) < 60 * 1000; // within 1 minute
        });
        if (dup) {
          duplicateWarning = `You already have a meeting "${dup.title}" scheduled at this time. You can still proceed if this is intentional.`;
        } else {
          duplicateWarning = '';
        }
      }
    } catch {
      // ignore duplicate check failures
    } finally {
      checkingDuplicate = false;
    }
  }

  // ── Duplicate detection triggers ───────────────────────────────
  $effect(() => {
    // React to title, date, or time changes for duplicate check
    const _ = title + scheduledDate + scheduledTime;
    scheduleDuplicateCheck();
  });

  // Debounced duplicate check when relevant fields change
  let duplicateTimer: ReturnType<typeof setTimeout>;
  function scheduleDuplicateCheck() {
    clearTimeout(duplicateTimer);
    duplicateTimer = setTimeout(checkDuplicate, 400);
  }

  // ── Participant management ─────────────────────────────────────
  function addParticipant() {
    participants = [...participants, defaultParticipant()];
  }

  function removeParticipant(index: number) {
    if (participants.length <= 1) return;
    participants = participants.filter((_, i) => i !== index);
  }

  // ── Form submission ───────────────────────────────────────────
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
      const res = await createUpcomingMeeting(payload, fetch);
      if (res.ok) {
        goto(`/upcoming/${res.data.id}`);
        return;
      }

      if (res.status === 400 && res.data.errors) {
        const errs: Record<string, string> = {};
        for (const err of res.data.errors) {
          errs[err.field] = err.message;
        }
        serverErrors = errs;
        if (res.data.message) serverErrorSummary = res.data.message;
      } else if (res.status === 403 || res.data.featureDisabled) {
        serverErrorSummary = 'Upcoming meetings are not enabled.';
      } else {
        serverErrorSummary = res.data.message ?? 'An unexpected error occurred. Please try again.';
      }
    } catch {
      serverErrorSummary = 'Network error. Please check your connection and try again.';
    } finally {
      saving = false;
    }
  }

  function cancel() {
    goto('/upcoming');
  }
</script>

<svelte:head>
  <title>Schedule Meeting — ContextPilot</title>
</svelte:head>

{#if !ffEnabled}
  <div class="page-container">
    <Alert variant="warning">
      Upcoming meetings are not enabled. Set <code>VITE_FF_ENABLE_UPCOMING_MEETINGS=true</code> to enable them.
    </Alert>
  </div>
{:else}
  <div class="page-container">
    <header class="page-header">
      <h1 class="page-title">Schedule Meeting</h1>
      <p class="page-subtitle">Create a future meeting record for pre-call preparation</p>
    </header>

    {#if serverErrorSummary}
      <Alert variant="error" class="top-alert">{serverErrorSummary}</Alert>
    {/if}

    <!-- Duplicate warning — FR-24 -->
    {#if duplicateWarning}
      <div role="alert" class="duplicate-warning">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
          <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
          <path d="M8 5v3M8 10.5v.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
        <span>{duplicateWarning}</span>
      </div>
    {/if}

    <Card>
      <form
        id="upcoming-meeting-form"
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
            min={minDatetime()}
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
          <label for="description" class="input-group__label">Description / agenda <span class="optional-label">(optional)</span></label>
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
          error={serverErrors.clientOrOrganization}
        />

        <!-- ── Participants ─────────────────────────────── -->
        <fieldset class="participants-section">
          <legend class="section-legend">
            Participants <span class="optional-label">(optional)</span>
          </legend>
          <p class="section-hint">Add participants to improve briefing accuracy.</p>

          {#if serverErrors.participants}
            <p class="field-error" role="alert">{serverErrors.participants}</p>
          {/if}

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
          {:else}
            <p class="field-error" role="alert">Maximum of 50 participants allowed.</p>
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
            Schedule meeting
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
    margin-bottom: var(--space-6);
  }

  .page-title {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--color-text-primary);
    margin: 0 0 var(--space-1) 0;
  }

  .page-subtitle {
    font-size: var(--text-sm);
    color: var(--color-text-secondary);
    margin: 0;
  }

  .top-alert {
    margin-bottom: var(--space-3);
  }

  .duplicate-warning {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4);
    background: rgb(234 179 8 / 0.08);
    border: 1px solid rgb(234 179 8 / 0.3);
    border-radius: var(--radius-sm);
    font-size: var(--text-sm);
    color: #92400e;
    margin-bottom: var(--space-3);
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

  .input-group__label {
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

  /* ── Participants ──────────────────────────────────── */
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

  .section-hint {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    margin: 0;
  }

  .field-error {
    font-size: var(--text-xs);
    color: var(--color-danger);
    margin: 0;
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

  /* ── Actions ─────────────────────────────────── */
  .form-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: var(--space-3);
    padding-top: var(--space-2);
    border-top: 1px solid var(--color-border);
  }
</style>