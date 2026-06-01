<script lang="ts">
  import { enhance } from '$app/forms';
  import { Button, Input, Card, Alert } from '$lib/components/ui';

  // ── Feature flag ──────────────────────────────────────────────
  const ffEnabled = import.meta.env.VITE_FF_ENABLE_MANUAL_MEETING_IMPORT === 'true';

  // ── Types ─────────────────────────────────────────────────────
  interface Participant {
    displayName: string;
    email: string;
    organization: string;
    role: string;
  }

  // ── Idempotency token (generated once per page load) ──────────
  const idempotencyToken = crypto.randomUUID();

  // ── Form state ────────────────────────────────────────────────
  let title = $state('');
  let completedAt = $state('');
  let completedTime = $state('');
  let transcript = $state('');
  let notes = $state('');
  let participants = $state<Participant[]>([defaultParticipant()]);
  let showAdvanced = $state(false);
  let saving = $state(false);

  // ── Server-returned errors (field-level) ───────────────────────
  let serverErrors = $state<Record<string, string>>({});
  let serverErrorSummary = $state('');

  // ── Derived ────────────────────────────────────────────────────
  const CHAR_LIMIT = 50_000;
  const CHAR_WARN = 45_000;

  function defaultParticipant() {
    return { displayName: '', email: '', organization: '', role: '' };
  }

  const combinedChars = $derived((transcript ?? '').length + (notes ?? '').length);
  const isOverWarn = $derived(combinedChars >= CHAR_WARN);
  const isOverLimit = $derived(combinedChars > CHAR_LIMIT);
  const canSubmit = $derived(
    !isOverLimit &&
    title.trim().length > 0 &&
    completedAt.length > 0 &&
    participants.length > 0 &&
    participants.every((p) => p.displayName.trim().length > 0) &&
    (transcript.trim().length > 0 || notes.trim().length > 0)
  );

  // ── Actions ────────────────────────────────────────────────────
  function addParticipant() {
    participants = [...participants, defaultParticipant()];
  }

  function removeParticipant(index: number) {
    if (participants.length <= 1) return;
    participants = participants.filter((_, i) => i !== index);
  }

  // ── Form submission via fetch (not form action) ────────────────
  // We use a hidden native form submit that JS intercepts for the fetch logic.
  // use:enhance on the native form handles the redirect on success and
  // re-renders with errors on validation failure.
  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!canSubmit || saving) return;

    saving = true;
    serverErrors = {};
    serverErrorSummary = '';

    const completedAtValue = completedTime
      ? `${completedAt}T${completedTime}:00Z`
      : `${completedAt}T00:00:00Z`;

    const payload = {
      title: title.trim(),
      completedAt: completedAtValue,
      participants: participants.map((p) => ({
        displayName: p.displayName.trim(),
        email: p.email.trim() || undefined,
        organization: p.organization.trim() || undefined,
        role: p.role.trim() || undefined,
      })),
      transcript: transcript.trim() || undefined,
      notes: notes.trim() || undefined,
      idempotencyToken,
    };

    try {
      const res = await fetch('/api/meetings', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(payload),
      });

      if (res.ok || res.status === 201) {
        const data = await res.json();
        window.location.href = data.redirect ?? `/meetings/${data.id}`;
        return;
      }

      const data = await res.json().catch(() => ({}));
      if (res.status === 400 && data.errors) {
        // Field-level errors — re-render form (values preserved automatically)
        const errs: Record<string, string> = {};
        for (const err of data.errors) {
          errs[err.field] = err.message;
        }
        serverErrors = errs;
        if (data.message) serverErrorSummary = data.message;
      } else if (res.status === 409) {
        // Duplicate — redirect to original meeting
        window.location.href = `/meetings/${data.meetingId ?? ''}`;
        return;
      } else if (res.status === 403) {
        serverErrorSummary = 'Manual meeting import is not enabled.';
      } else {
        serverErrorSummary = 'An unexpected error occurred. Please try again.';
      }
    } catch {
      serverErrorSummary = 'Network error. Please check your connection and try again.';
    } finally {
      saving = false;
    }
  }

  function cancel() {
    window.location.href = '/meetings';
  }
</script>

{#if !ffEnabled}
  <div class="page-container">
    <Alert variant="warning">
      Manual meeting import is not available. Set <code>VITE_FF_ENABLE_MANUAL_MEETING_IMPORT=true</code> to enable it.
    </Alert>
  </div>
{:else}
  <div class="page-container">
    <header class="page-header">
      <h1 class="page-title">Import Meeting</h1>
      <p class="page-subtitle">Record a completed meeting from memory</p>
    </header>

    {#if serverErrorSummary}
      <Alert variant="error" class="top-alert">{serverErrorSummary}</Alert>
    {/if}

    <Card>
      <form
        id="manual-meeting-form"
        onsubmit={handleSubmit}
        novalidate
        class="meeting-form"
      >
        <input type="hidden" name="idempotencyToken" value={idempotencyToken} />

        <!-- ── Title ─────────────────────────────────────── -->
        <Input
          id="title"
          name="title"
          label="Meeting title"
          type="text"
          placeholder="Q3 planning review"
          bind:value={title}
          error={serverErrors.title}
          required
        />

        <!-- ── Completed date & time ─────────────────────── -->
        <div class="date-time-row">
          <Input
            id="completedAt"
            name="completedAt"
            label="Completed date"
            type="date"
            bind:value={completedAt}
            error={serverErrors.completedAt}
            required
          />
          <Input
            id="completedTime"
            name="completedTime"
            label="Completed time (optional)"
            type="time"
            bind:value={completedTime}
          />
        </div>

        <!-- ── Participants ─────────────────────────────── -->
        <fieldset class="participants-section">
          <legend class="section-legend">
            Participants
            <span class="required-badge" aria-hidden="true">*</span>
          </legend>

          {#if serverErrors.participants}
            <p class="field-error" role="alert">{serverErrors.participants}</p>
          {/if}

          <div class="participants-list" aria-label="Participant list">
            {#each participants as participant, i}
              <div class="participant-row" data-index={i}>
                <div class="participant-row__main">
                  <Input
                    id="participants[{i}].displayName"
                    name="participants[{i}].displayName"
                    label={i === 0 ? 'Display name' : undefined}
                    type="text"
                    placeholder="Alice Smith"
                    bind:value={participant.displayName}
                    error={serverErrors[`participants[${i}].displayName`]}
                    required
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
                    />
                    <Input
                      id="participants[{i}].organization"
                      name="participants[{i}].organization"
                      label="Organization (optional)"
                      type="text"
                      placeholder="Acme Corp"
                      bind:value={participant.organization}
                    />
                    <Input
                      id="participants[{i}].role"
                      name="participants[{i}].role"
                      label="Role / title (optional)"
                      type="text"
                      placeholder="Engineer"
                      bind:value={participant.role}
                    />
                  </div>
                {/if}
              </div>
            {/each}
          </div>

          <button type="button" class="add-participant-btn" onclick={addParticipant}>
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
              <path d="M7 2v10M2 7h10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
            </svg>
            Add participant
          </button>

          <button
            type="button"
            class="toggle-advanced-btn"
            onclick={() => (showAdvanced = !showAdvanced)}
            aria-expanded={showAdvanced}
          >
            {showAdvanced ? 'Hide' : 'Show'} advanced participant details
          </button>
        </fieldset>

        <!-- ── Transcript & Notes ────────────────────────── -->
        <fieldset class="content-section">
          <legend class="section-legend">
            Meeting content
            <span class="required-badge" aria-hidden="true">*</span>
          </legend>
          <p class="section-hint">At least one of transcript or notes is required.</p>

          {#if serverErrors.content}
            <p class="field-error" role="alert">{serverErrors.content}</p>
          {/if}

          <div class="textarea-wrapper">
            <label for="transcript" class="textarea-label">Transcript</label>
            <textarea
              id="transcript"
              name="transcript"
              placeholder="Paste or type the meeting transcript…"
              bind:value={transcript}
              rows="6"
              class:textarea--error={!!serverErrors.content}
            ></textarea>
          </div>

          <div class="textarea-wrapper">
            <label for="notes" class="textarea-label">Notes</label>
            <textarea
              id="notes"
              name="notes"
              placeholder="Or enter meeting notes…"
              bind:value={notes}
              rows="6"
              class:textarea--error={!!serverErrors.content}
            ></textarea>
          </div>

          <!-- Character count -->
          <div class="char-count-wrapper">
            <div class="char-count" aria-live="polite" aria-atomic="true">
              {#if isOverLimit}
                <span class="char-count__over" role="alert">
                  {combinedChars.toLocaleString()} / {CHAR_LIMIT.toLocaleString()} characters — limit exceeded
                </span>
              {:else if isOverWarn}
                <span class="char-count__warn">
                  {combinedChars.toLocaleString()} / {CHAR_LIMIT.toLocaleString()} characters — approaching limit
                </span>
              {:else}
                <span class="char-count__ok">
                  {combinedChars.toLocaleString()} / {CHAR_LIMIT.toLocaleString()} characters
                </span>
              {/if}
            </div>
          </div>
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
            Save meeting
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
    display: flex;
    align-items: center;
    gap: var(--space-1);
    padding: 0;
    margin-bottom: var(--space-1);
  }

  .required-badge {
    color: var(--color-danger);
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
    grid-template-columns: 1fr 1fr 1fr;
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
    text-decoration: underline;
    text-underline-offset: 2px;
  }

  .add-participant-btn:hover {
    color: var(--color-primary-hover);
  }

  .add-participant-btn:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
    border-radius: var(--radius-sm);
  }

  .toggle-advanced-btn {
    background: none;
    border: none;
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    cursor: pointer;
    padding: var(--space-1) 0;
    text-decoration: underline;
    text-underline-offset: 2px;
    transition: color var(--transition-fast);
  }

  .toggle-advanced-btn:hover {
    color: var(--color-text-secondary);
  }

  .toggle-advanced-btn:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
    border-radius: var(--radius-sm);
  }

  /* ── Content section ───────────────────────────────── */
  .content-section {
    border: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .textarea-wrapper {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .char-count-wrapper {
    display: flex;
    justify-content: flex-end;
  }

  .textarea-label {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--color-text-primary);
  }

  textarea {
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    font-family: var(--font-sans);
    font-size: var(--text-base);
    color: var(--color-text-primary);
    background: var(--color-background);
    resize: vertical;
    transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
    width: 100%;
  }

  textarea::placeholder {
    color: var(--color-text-muted);
  }

  textarea:focus {
    outline: none;
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px rgb(26 86 219 / 0.15);
  }

  textarea.textarea--error {
    border-color: var(--color-danger);
  }

  textarea.textarea--error:focus {
    box-shadow: 0 0 0 3px rgb(220 38 38 / 0.15);
  }

  .char-count {
    font-size: var(--text-xs);
  }

  .char-count__ok {
    color: var(--color-text-muted);
  }

  .char-count__warn {
    color: var(--color-warning);
    font-weight: 500;
  }

  .char-count__over {
    color: var(--color-danger);
    font-weight: 600;
  }

  /* ── Actions ────────────────────────────────────────── */
  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-3);
    padding-top: var(--space-2);
    border-top: 1px solid var(--color-border);
  }

  /* ── Responsive ────────────────────────────────────── */
  @media (max-width: 600px) {
    .date-time-row {
      grid-template-columns: 1fr;
    }

    .participant-row__advanced {
      grid-template-columns: 1fr;
    }
  }
</style>