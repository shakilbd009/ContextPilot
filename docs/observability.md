# Observability

> Version 2.0.0 · Structured logs, metrics, traces, and error handling expectations

> **Current state (2026-06-01):** this document is the policy baseline. Implementation status is tracked separately — see [STATUS.md](../STATUS.md) → "Recovery state". No feature is "Production-Ready" yet. The contract below ("All three are required before a feature can be marked `Active`") is not yet satisfied for any feature; it is the bar to clear before flipping a flag to Active.

---

## Overview

ContextPilot must be observable in production without requiring engineers to SSH into boxes. The three pillars:
- **Structured logging** — machine-parseable, correlation-ID-linked
- **Metrics** — business and operational signals emitted continuously
- **Traces** — distributed request flow across services

All three are required before a feature can be marked `Active` in `specs/feature-flags.md`.

---

## Structured Logging

### Log Format

JSON to stdout (container-native). One event per line.

```json
{
  "level": "INFO",
  "timestamp": "2026-05-19T10:30:00Z",
  "message": "Briefing generated",
  "request_id": "req_abc123",
  "meeting_id": "mt_456",
  "duration_ms": 342,
  "user_id": "usr_789",
  "service": "contextpilot-backend"
}
```

### Log Levels

| Level | When to use |
|-------|-------------|
| DEBUG | Detailed diagnostic info (not in production) |
| INFO | Normal operational events — request received, briefing generated |
| WARN | Degraded state — retry, fallback, timeout |
| ERROR | Operation failed — returns 500 to client, logged with stack trace |
| FATAL | Process cannot continue — startup failure, OOM |

### Required Fields

| Field | Description |
|-------|-------------|
| `level` | Log level string |
| `timestamp` | ISO 8601 UTC |
| `message` | Human-readable description |
| `request_id` | `X-Request-ID` header or generated UUID |
| `service` | Always `contextpilot-backend` or `contextpilot-frontend` |

### Optional Fields

| Field | Description |
|-------|-------------|
| `meeting_id` | Present when operation is meeting-scoped |
| `user_id` | Present when operation is user-scoped |
| `duration_ms` | Present for async operations |
| `error` | Present on ERROR/FATAL, contains error message and stack |

### What to Log

- Request received (INFO) — method, path, request_id, user_id
- Request completed (INFO) — status code, duration_ms, request_id
- Briefing generated (INFO) — meeting_id, duration_ms, request_id
- Retry attempt (WARN) — operation, attempt number, reason
- Unhandled error (ERROR) — stack trace, request_id

### What NOT to Log

- Passwords or tokens (redact from `Authorization` header)
- Full request bodies for PII fields (attendee emails, meeting transcripts)
- Stack traces in client-facing responses (return error ID instead)

---

## Metrics

### Architecture

Metrics emitted via Prometheus client library (`github.com/prometheus/client_golang` in Go).
Exposed at `GET /metrics`.

### Naming Convention

```
cp_<feature>_<operation>_<type>
```

| Segment | Example | Notes |
|---------|---------|-------|
| `cp` | ContextPilot prefix | Always `cp` |
| `<feature>` | `meeting`, `briefing`, `app_shell` | From BRD |
| `<operation>` | `import`, `generate`, `render` | Action verb |
| `<type>` | `total`, `duration_ms`, `errors` | Counter or histogram |

### Required Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `cp_http_requests_total` | Counter | `method`, `path`, `status` | All HTTP requests |
| `cp_http_request_duration_ms` | Histogram | `method`, `path` | Request latency |
| `cp_meeting_import_total` | Counter | `status` | Meeting import attempts |
| `cp_meeting_import_duration_ms` | Histogram | — | Import processing time |
| `cp_briefing_generate_total` | Counter | `status` | Briefing generation |
| `cp_briefing_generate_duration_ms` | Histogram | — | Generation time |
| `cp_app_shell_paint_duration_ms` | Histogram | `metric` (fcp/ttir) | Frontend paint metrics |

### Health Endpoints

| Endpoint | Returns | When |
|----------|---------|------|
| `GET /live` | `{"status":"ok"}` | Process is alive |
| `GET /ready` | `{"status":"ok","postgres":"ok","redis":"ok"}` | Dependencies are reachable |

---

## Distributed Tracing

- Every request carries `X-Request-ID` from gateway or generates one
- `X-Request-ID` is added to all log events for the request
- Database queries include the request context
- Future: OpenTelemetry collector for trace propagation

---

## Frontend Error Handling

### Error Boundary

Every SvelteKit route wraps content in an error boundary component. Uncaught errors render a friendly error card (never a blank page or raw exception).

### Frontend Error Format

```json
{
  "level": "ERROR",
  "timestamp": "2026-05-19T10:30:00Z",
  "message": "Uncaught error in MeetingList",
  "request_id": "req_frontend_abc",
  "component": "MeetingList",
  "error": "TypeError: Cannot read properties of undefined",
  "url": "/meetings",
  "user_id": "usr_789"
}
```

### What Frontend Logs

- Uncaught exceptions in components
- API errors (4xx, 5xx responses)
- Feature flag evaluation failures

---

## Local Debugging

```bash
# View logs
docker compose logs -f backend

# Check health
curl http://localhost:8080/live
curl http://localhost:8080/ready

# View metrics
curl http://localhost:8080/metrics

# Trace a request
curl -H "X-Request-ID: test-123" http://localhost:8080/api/v1/meetings
grep "test-123" docker compose logs
```

---

## Alerts (Future)

When implementing, add alerts for:
- Error rate > 5% over 5 minutes
- P99 latency > 2s for `/api/v1/meetings`
- `cp_briefing_generate_errors_total` increasing
- Disk usage > 80% on postgres volume

Alert routing: PagerDuty (ops on-call) for critical; Slack #alerts for warning.