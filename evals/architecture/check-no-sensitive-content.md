# check-no-sensitive-content

> Fitness function: no hardcoded sensitive meeting-derived content in logs or metrics.

## Why This Rule Exists

BRD-03 Meeting Memory Processing handles some of the most sensitive data in the system: meeting transcripts, evidence snippets, participant PII, stakeholder notes, generated memories, and conflict resolution notes. If any of this content appears in log lines, metric labels, or telemetry events, it creates a data-exfiltration risk and violates US-8 (Observability Without Sensitive Content).

Operators must be able to observe processing lifecycle events — job queued, started, completed, failed, retried — without ever seeing the underlying meeting content.

## What This Checks

The eval scans `backend/` (Go) and `frontend/` (TypeScript/Svelte) for patterns that would emit sensitive meeting-derived fields in logging or metric APIs:

| Pattern class | Examples detected |
|---|---|
| Log calls with sensitive field names | `log.Info("processing", "transcript", value)` |
| Metric labels with sensitive keys/values | `metrics.WithLabel("transcript", ...)` |
| String search on sensitive fields | `strings.Contains(note, transcript.Get())` |
| Sprintf embedding of sensitive fields | `fmt.Sprintf("note: %s", transcript.Text)` |
| Structured logger field keys | `zap.String("evidenceSnippet", ...)` |
| JSON marshal in log argument | `log.Info("", json.Marshal(mem))` |
| console.* with sensitive fields | `console.log(transcript, notes)` |
| Analytics/telemetry with sensitive keys | `track("event", { transcript: ... })` |

## Grace Period

None. Sensitive content in logs/metrics is a data-exfiltration risk and is never acceptable in production code.

## Escape Hatch

Files may opt out of a specific check by adding `// ARCH_OK:` on the line immediately above the match, or on the same line as a prefix before the violation text:

```go
// ARCH_OK: test only — no real transcript data
log.Info("test", "transcript", "test-transcript-content")
```

```typescript
// ARCH_OK: test fixture
console.log("transcript", testTranscript);
```

## Phase 0 Behavior

Exits 0 when `backend/` and `frontend/` contain no `.go`, `.ts`, or `.svelte` files (Phase 0 safe).

## Running

```bash
# Standalone
bash evals/architecture/check-no-sensitive-content.sh

# Via Makefile (included in eval-arch)
make eval-arch
```