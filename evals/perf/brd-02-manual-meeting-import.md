# Performance Eval: brd-02-manual-meeting-import

> 🟡 Known Failure — NFR save latency < 1s and form interactivity < 100ms cannot be measured end-to-end until t_a2c3d3d5 (frontend POST STUB repair) is resolved. Backend save latency confirmed < 200ms via handler tests; character count histogram populated.

## Scope

Performance benchmarks for Manual Meeting Import (BRD-02): form interactivity latency, save round-trip latency, character count responsiveness, and concurrent load handling.

---

## Form Interactivity

| Metric | Target | Method |
|--------|--------|--------|
| Field update response | < 100ms | Type in title field, measure DOM update |
| Participant row add/remove | < 100ms | Click add/remove participant, measure DOM |
| Advanced section expand/collapse | < 100ms | Toggle advanced participant details, measure DOM |
| Validation feedback display | < 100ms | Submit invalid form, measure error render |
| Character count update | < 100ms | Type in transcript, measure live count update at 50k chars |
| Save button state (enabled/disabled) | < 100ms | Fill required fields, measure button enabled state |

Test input: simulate typing in transcript field with content up to 50,000 combined characters to verify no jank.

---

## Save Latency

| Metric | Target | Method |
|--------|--------|--------|
| Save round-trip (happy path) | < 1s | Click Save → receive redirect; measured client-side |
| Save round-trip at character limit | < 1s | Submit with 50,000 combined chars, measure redirect time |
| Save round-trip with 3 participants | < 1s | Submit with 3 full participant records, measure redirect time |
| Time to first byte (TTFB) | < 200ms | Measure from Save click to first response byte |

---

## Character Count Benchmarks

| Scenario | Behavior | Target |
|----------|----------|--------|
| Warning threshold (45,000 chars) | Live count turns warning color | Count updates < 100ms on keystroke |
| Block threshold (50,001 chars) | Save button disabled | Button disabled < 100ms |
| At limit (50,000 chars) | Warning color, save still enabled | Count display accurate |
| Above limit (50,001+) | Save blocked, error message | Blocking < 100ms |

---

## Load Handling

| Metric | Target | Method |
|--------|--------|--------|
| Concurrent form submissions | 10 simultaneous users | All complete within 5s; no duplicate meetings created |
| POST /meetings under load | 50 requests/s | p95 latency < 2s |
| GET /meetings under load | 100 requests/s | p95 latency < 500ms |
| Database connection pool | No pool exhaustion at 20 concurrent saves | All succeed within timeout |

---

## Bundle Size

| Asset | Target | Method |
|-------|--------|--------|
| Initial JS bundle | < 150KB gzipped | Measure via Lighthouse or webpack stats |
| Initial CSS bundle | < 30KB gzipped | Measure via Lighthouse or webpack stats |
| Incremental bundle (lazy load) | < 50KB gzipped per route | Measure meeting import route chunk |

---

## Running

```bash
# Lighthouse CI
make eval-perf

# Manual latency measurement
curl -w "@curl-format.txt" -X POST http://localhost:3000/meetings \
  -H "Content-Type: application/json" \
  -d '{"title":"Perf Test","completedAt":"2026-05-20T14:00:00Z","participants":[{"displayName":"Alice"}],"transcript":"test","idempotencyToken":"550e8400-e29b-41d4-a716-446655440000"}'
```