# ADR-0008: Meeting Memory Processing — Observability & Privacy Architecture

> Status: Accepted

## Context

BRD-03 FR-8 (US-8) requires that operators observe processing lifecycle, retries, failures, and active version changes without exposing sensitive meeting content. The NFR on privacy explicitly bans raw transcript text, notes text, evidence snippets, participant PII, stakeholder note content, generated memory content, prior memory content, and resolution note text from appearing in logs or metric labels. US-8 AC-24 defines the specific fields that must appear in metrics and logs (correlation ID, meeting ID safe identifier, safe trigger type, failure class, retry count, duration, active version change reason, flag evaluation duration).

This ADR defines the observability layer: what events are emitted, what fields are allowed/prohibited, how structured logging and metrics are implemented, and how the privacy requirements are enforced architecturally. It also defines the authentication mechanism for memory API endpoints.

## Authentication for Memory Endpoints

All memory API endpoints (`GET /meetings/:id/memory`, `GET /meetings/:id/memory/versions`, `POST /meetings/:id/memory/reprocess`, `GET /meetings/:id/memory/state`, `GET /meetings/:id/memory/conflicts`, `POST /meetings/:id/memory/conflicts/:id/resolve`) require authenticated user sessions via the same BearerAuth or CookieAuth used by existing meeting endpoints.

**Authorization**: A user can only access memory for meetings they own or that are visible to them via standard meeting ACL. Memory processing jobs are server-side only (not user-scoped) — the queue worker operates with elevated server privileges but the resulting memory versions are subject to meeting-level authorization checks on read/write.

**Feature flag gate**: `FF_ENABLE_MEETING_MEMORY_PROCESSING` gates all memory write operations (reprocess, resolve conflict). Read operations (GET memory, GET versions, GET state, GET conflicts) also require the flag to be `true` — when `false`, the API returns 403 Forbidden with a safe error message, consistent with how other disabled features behave.

**Auth mechanism**:
- `Authorization: Bearer <jwt>` header or `session` cookie — same as existing meeting endpoints
- No separate auth scheme for memory endpoints — the same session context applies
- CORS: memory endpoints are subject to the same CORS policy as other API endpoints (same-origin only, credentials required)

## Decision

### Allowed and Prohibited Fields

**Allowed in logs and metrics** (per US-8 AC-24):
- `correlation_id` (UUID — opaque, no business meaning)
- `meeting_id` (UUID — safe identifier, no PII)
- `trigger_type` (enum: `import`, `reprocess`, `stale_reprocess`, `manual_retry`)
- `failure_class` (enum: `transient_provider_error`, `permanent_processing_error`, `validation_error`, `timeout`, `worker_unavailable`)
- `retry_count` (integer)
- `duration_ms` (integer — elapsed time in milliseconds)
- `active_version_change_reason` (enum: `new_version`, `conflict_resolved`, `user_reprocess`, `stale_reprocess`)
- `flag_evaluation_duration_ms` (integer)
- `processing_version_number` (integer — version number of the memory version produced)
- `category_count` (integer — number of categories in the output)
- `quality_distribution` (JSON object: `{"strong_evidence": N, "weak_evidence": N, "insufficient_evidence": N, "conflicting_evidence": N}`)

**Prohibited from all logs and metrics**:
- Raw transcript text
- Raw notes text
- Evidence snippets
- Participant names
- Participant emails
- Participant organizations
- Stakeholder note content
- Generated memory text (summary statements, decision statements, etc.)
- Prior memory content
- Resolution note text
- Source locations (character offsets, line references — these could be used to reconstruct snippets)
- `id` fields from `memory_evidence`, `memory_conflicts`, or `memory_prior_memory_inputs` that could be correlated with content

### Structured Logging

Logs are emitted as structured JSON (JSON Lines format) to stdout, collected by the container runtime and forwarded to the logging backend. Each log entry contains:

```json
{
  "timestamp": "2026-05-21T12:00:00.000Z",
  "level": "INFO",
  "service": "memory-processor",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "meeting_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "event": "memory.job.completed",
  "trigger_type": "import",
  "retry_count": 0,
  "duration_ms": 3421,
  "processing_version_number": 1,
  "category_count": 7,
  "quality_distribution": {"strong_evidence": 12, "weak_evidence": 5, "insufficient_evidence": 1, "conflicting_evidence": 0},
  "active_version_change_reason": "new_version"
}
```

Log levels:
- `DEBUG`: detailed step-level tracing (extraction, matching, conflict detection) — filtered out in production
- `INFO`: job lifecycle events (queued, started, completed, failed, retrying)
- `WARN`: non-fatal conditions (retry approaching exhaustion, uncertain match included by user)
- `ERROR`: permanent failures, unrecoverable states

### Metrics

Metrics are exposed via the Go server's `/metrics` endpoint (Prometheus format) and should be forwarded to the metrics backend. The following metrics are defined:

```
# Counter: total jobs by terminal status
memory_jobs_total{status="completed", trigger_type="import"} 142
memory_jobs_total{status="completed", trigger_type="manual_retry"} 23
memory_jobs_total{status="failed", trigger_type="import"} 7
memory_jobs_total{status="retry_exhausted", trigger_type="reprocess"} 2

# Histogram: job processing duration
memory_job_duration_seconds_bucket{le="1"} 45
memory_job_duration_seconds_bucket{le="5"} 120
memory_job_duration_seconds_bucket{le="30"} 158
memory_job_duration_seconds_bucket{le="+Inf"} 172

# Gauge: jobs in each state
memory_jobs_in_progress 3
memory_jobs_queued 12

# Counter: conflict events
memory_conflicts_total{meeting_id="..."} 5  # meeting_id is the safe UUID, not the content

# Histogram: flag evaluation duration
memory_flag_evaluation_ms_bucket{le="5"} 180
memory_flag_evaluation_ms_bucket{le="50"} 195

# Histogram: quality distribution per completed job (tags are prohibited — distribution stored in content, not labels)
# Note: quality distribution is NOT a metric label — it's emitted as a log event field
```

### Privacy Enforcement

**Architectural enforcement**: The `MemoryProcessor` interface and its implementations are the enforcement point. The processor receives the meeting content (transcript, notes) and prior memories as inputs, but the structured logger and metrics emitter only receive a `ProcessingResult` struct that contains the allowed fields listed above.

```go
// ProcessingResult contains only safe, non-sensitive fields
type ProcessingResult struct {
    CorrelationID             uuid.UUID
    MeetingID                 uuid.UUID
    TriggerType               string
    DurationMs                int64
    RetryCount                int
    Status                    JobStatus
    FailureClass              string  // never raw error messages
    ActiveVersionChangeReason string
    ProcessingVersionNumber   int
    CategoryCount             int
    QualityDistribution       map[string]int
    FlagEvaluationDurationMs int64
}
```

The rule: **the concrete `MemoryProcessor` implementation must not have access to the logger's JSON encoder or metrics registry directly**. It returns a `ProcessingResult` and an error; the queue worker handles the emission. This prevents accidental logging of sensitive content from within the processor logic.

**Content redaction utility**: A `Redactor` utility is provided that accepts any string (transcript, notes, evidence snippet, generated text) and returns a stable one-way hash of that content for the sole purpose of log correlation. This allows operators to correlate two log entries about the same content item without ever seeing the content itself.

```go
// Redactor returns a SHA-256 hash of the input string.
// The hash is stable for the lifetime of the process and can be used
// to correlate log entries about the same content without exposing the content.
func Redactor(content string) string {
    h := sha256.Sum256([]byte(content))
    return hex.EncodeToString(h[:8])  // first 8 hex chars for readability
}
```

Note: The redactor hash is NOT used as a safe content identifier in the data model — it is only for log correlation.

### Observability Events

| Event | When | Allowed Fields |
|-------|------|---------------|
| `memory.job.queued` | Job row inserted | correlation_id, meeting_id, trigger_type |
| `memory.job.started` | Worker picks up job | correlation_id, meeting_id, trigger_type, processing_version_number (if known) |
| `memory.job.completed` | Processing succeeds | All allowed fields |
| `memory.job.failed` | Processing returns permanent error | correlation_id, meeting_id, trigger_type, failure_class, retry_count, duration_ms |
| `memory.job.retrying` | Transient failure, retry scheduled | correlation_id, meeting_id, trigger_type, retry_count, duration_ms, next_retry_at |
| `memory.job.retry_exhausted` | Max retries reached | correlation_id, meeting_id, trigger_type, retry_count, failure_class |
| `memory.version.activated` | New active version set | correlation_id, meeting_id, processing_version_number, active_version_change_reason |
| `memory.conflict.detected` | Conflict placed in review queue | correlation_id, meeting_id, category, conflict_id |
| `memory.conflict.resolved` | User resolves conflict | correlation_id, meeting_id, conflict_id, resolved_by |
| `memory.flag.evaluated` | Feature flag evaluated for a job | correlation_id, meeting_id, flag_name, evaluation_duration_ms, result |

## Rationale

**Structured JSON logs over unstructured text**: JSON Lines format enables log aggregation systems (Loki, Elasticsearch, CloudWatch) to index and query specific fields without parsing free text. This is essential for the privacy redaction requirement — we need to be able to verify programmatically that prohibited fields never appear in logs.

**Metrics over direct log counts**: Job counts, durations, and quality distributions are best served as Prometheus metrics (counter, histogram, gauge) because they support aggregation, alerting, and SLO tracking. Log events are for traceability — metrics are for measurement.

**Correlation ID as the primary join key**: All events for a given processing job share the same `correlation_id`, which is generated when the job is queued and stored in `memory_processing_jobs.correlation_id`. This enables full job traces in the logging system without using `meeting_id` as a join key (which would expose meeting ID in every log line even when not needed for correlation).

**Redactor utility**: The only way to verify privacy requirements is to have a mechanical redaction step that converts any sensitive string to a fixed-length hash. This prevents the common failure mode where an engineer adds a "helpful" log field and inadvertently exposes meeting content.

**Quality distribution as a log field, not metric labels**: Quality distribution (strong/weak/insufficient/conflicting counts) is per-job data that belongs in the job completion event. Storing it as metric labels would create a high-cardinality label set (`quality_distribution{strong=N, weak=M}` would create N×M unique label combinations across all jobs). Emitting it as a JSON field in the log event achieves the same analytical purpose without metric label explosion.

## Trade-offs

| Aspect | What we give up |
|--------|-----------------|
| Real-time alerting on quality distribution | Quality distribution is a log field, not a metric — querying it requires log aggregation rather than Prometheus PromQL |
| Debug logs for processing internals | DEBUG level logs are filtered in production; some debugging information may be unavailable without log level changes |
| Direct log correlation by meeting_id without correlation_id | Operators must use correlation_id to follow a job; meeting_id alone is insufficient for tracing |
| Cross-job aggregation of conflict content | Conflicts are not aggregated in metrics; operators must use the review queue API to see conflict content |

## Consequences

1. The `ProcessingResult` struct is the only data passed from `MemoryProcessor` to the queue worker for emission.
2. The `Redactor` utility is used in the queue worker when constructing log events from processor output.
3. The Go server's `/metrics` endpoint exposes Prometheus metrics for job lifecycle and duration.
4. Logs are emitted as JSON Lines to stdout; the Docker Compose logging driver must be configured to forward them to the log aggregation backend.
5. Observability evals (AC-23) verify that no prohibited field appears in logs or metric labels by grep-ing log output and metric label names.
6. The `check-no-sensitive-content.sh` architecture eval (to be written) validates that `Redactor` is called on all string fields from `transcript`, `notes`, `evidence_snippet`, `stakeholder_note`, `generated_memory_content`, `prior_memory_content`, and `resolution_note` before those fields can be used in log or metric emission.

## Date

2026-05-21