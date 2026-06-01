# ContextPilot Security Testing — Environment Briefing

**Prepared for:** Ethical Hacker Live Test
**Date:** 2026-05-30
**Environment:** Development local

---

## 1. Service Health

| Service | URL | Status | Notes |
|---------|-----|--------|-------|
| Backend | http://localhost:3000 | HEALTHY | `/healthz` returns `OK` |
| Frontend | http://localhost:5173 | HEALTHY | Returns HTTP 200 |
| PostgreSQL | localhost:5432 | RUNNING (Docker) | `contextpilot` DB |
| Redis | localhost:6379 | RUNNING (Docker) | `redis_data` volume |

> **Note:** Docker daemon is running (used for backend/frontend containers). `docker-compose ps` works for lifecycle commands.

---

## 2. Active Feature Flags

| Flag | Backend Value | Browser Value | Source |
|------|--------------|---------------|--------|
| `FF_ENABLE_APP_SHELL` | `false` | `VITE_FF_ENABLE_APP_SHELL=false` | docker-compose.yml |
| `FF_ENABLE_UPCOMING_MEETINGS` | `true` | `VITE_FF_ENABLE_UPCOMING_MEETINGS=true` | docker-compose.yml |
| `FF_ENABLE_MANUAL_MEETING_IMPORT` | `true` | `VITE_FF_ENABLE_MANUAL_MEETING_IMPORT=true` | docker-compose.yml |
| `FF_ENABLE_MEETING_MEMORY_PROCESSING` | not set (default false) | not set | docker-compose.yml |

Flags are passed via Docker Compose `environment:` blocks. No `.env` file overrides in dev.

---

## 3. Auth Mechanism (STUB — Phase 1)

**Auth is stub-only per BRD-01.** No real authentication.

- Header: `X-User-ID: <UUID v4>`
- Any valid UUID v4 format is accepted
- No token validation, no session, no login endpoint
- All authenticated endpoints return `401 Unauthorized` if header is missing or not a valid UUID

**Test UUIDs (any valid UUIDv4 works):**
```
550e8400-e29b-41d4-a716-446655440000
f47ac10b-58cc-4372-a567-0e02b2c3d479
```

**Note:** `FF_ENABLE_APP_SHELL=false` means the app-shell frontend route is disabled. Testing should target the default (unauthenticated) shell.

---

## 4. API Inventory (Base: `/api/v1`)

### Endpoints (all require `X-User-ID: <UUID>` header)

| Method | Path | FF Required | Notes |
|--------|------|-------------|-------|
| GET | `/api/v1/meetings` | — | List user's meetings; returns 500 if no meetings exist |
| POST | `/api/v1/meetings` | `FF_ENABLE_MANUAL_MEETING_IMPORT=true` | Create completed meeting; validates participants, idempotencyToken (UUID), completedAt |
| GET | `/api/v1/meetings/:id` | — | Get single meeting; returns 500 on lookup (likely UUID format issue) |
| GET | `/api/v1/meetings/:id/memory` | `FF_ENABLE_MEETING_MEMORY_PROCESSING=true` | Returns 403 if flag off |
| GET | `/api/v1/meetings/:id/memory/state` | `FF_ENABLE_MEETING_MEMORY_PROCESSING=true` | Returns 403 if flag off |
| GET | `/api/v1/meetings/:id/memory/versions` | `FF_ENABLE_MEETING_MEMORY_PROCESSING=true` | Returns 403 if flag off |
| POST | `/api/v1/meetings/:id/memory/reprocess` | `FF_ENABLE_MEETING_MEMORY_PROCESSING=true` | Returns 403 if flag off |
| GET | `/api/v1/meetings/:id/memory/conflicts` | `FF_ENABLE_MEETING_MEMORY_PROCESSING=true` | Returns 403 if flag off |
| POST | `/api/v1/meetings/:id/memory/conflicts/:conflictId/resolve` | `FF_ENABLE_MEETING_MEMORY_PROCESSING=true` | Returns 403 if flag off |
| GET | `/api/v1/upcoming` | `FF_ENABLE_UPCOMING_MEETINGS=true` | List upcoming meetings |
| POST | `/api/v1/upcoming` | `FF_ENABLE_UPCOMING_MEETINGS=true` | Create upcoming meeting |
| GET | `/api/v1/upcoming/:id` | `FF_ENABLE_UPCOMING_MEETINGS=true` | Get single upcoming meeting |
| PATCH | `/api/v1/upcoming/:id` | `FF_ENABLE_UPCOMING_MEETINGS=true` | Update upcoming meeting |
| DELETE | `/api/v1/upcoming/:id` | `FF_ENABLE_UPCOMING_MEETINGS=true` | Cancel upcoming meeting |

### POST /api/v1/meetings — Required Fields
```json
{
  "title": "string (max 500 chars)",
  "completedAt": "ISO 8601 timestamp (required)",
  "participants": [
    {
      "displayName": "string (required)",
      "email": "string (optional)"
    }
  ],
  "transcript": "string OR notes string (at least one required)",
  "idempotencyToken": "UUID v4 string (required)"
}
```

### Unauthenticated Behavior
- Missing `X-User-ID` header → `401 Unauthorized` with body `{"type":"about:blank","title":"Unauthorized","status":401}`
- Invalid (non-UUID) `X-User-ID` → `401 Unauthorized`

---

## 5. Test Data

**No seed data currently in DB.** The `scripts/seed-dev.sh` is a Phase 0 stub (exits 0 with no-op message).

To create test meeting data, use:
```bash
curl -X POST http://localhost:3000/api/v1/meetings \
  -H "X-User-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "title":"All-Hands Q1",
    "completedAt":"2025-01-01T10:00:00Z",
    "participants":[{"displayName":"Alice","email":"alice@example.com"}],
    "transcript":"Alice: Welcome to the all-hands...",
    "idempotencyToken":"f47ac10b-58cc-4372-a567-0e02b2c3d479"
  }'
```

---

## 6. Playwright E2E Tests

**Available specs:**
- `frontend/tests/e2e/brd-02-manual-meeting-import.spec.ts`
- `frontend/tests/e2e/upcoming.spec.ts`

To run (from inside frontend container or host if deps installed):
```bash
cd frontend && pnpm exec playwright test
```

---

## 7. Network Access

- Backend: `localhost:3000` (bound to `*:3000` via Docker)
- Frontend: `localhost:5173` (bound to `*:5173` via Docker)
- PostgreSQL: `localhost:5432` (not exposed externally in default config)
- Redis: `localhost:6379` (not exposed externally in default config)

**No VPN restrictions detected** — standard local network.

---

## 8. Known Issues / Limitations

1. **GET /api/v1/meetings/:id returns 500** — likely UUID format handling issue when looking up by ID; do not rely on this endpoint for individual meeting lookups until fixed.
2. **No seed data** — DB starts empty; create test data via POST before testing memory flows.
3. **Auth is trivially bypassed** — any UUID in `X-User-ID` header grants full access to that user's data. No privilege separation.
4. **`FF_ENABLE_MEETING_MEMORY_PROCESSING` is off** — memory endpoints return 403; enable flag in docker-compose.yml and restart backend to test those flows.
5. **Phase 1 stub codebase** — this is a real Go+SvelteKit implementation, not a mock. Real attack surface.

---

## 9. Scope for Ethical Hacker

**In scope:**
- All HTTP endpoints at `localhost:3000` and `localhost:5173`
- Auth bypass (trivial — any UUID)
- Input validation gaps on any endpoint
- SQL injection, parameter tampering, header injection
- Frontend XSS, CSRF (no CSRF token currently)
- Rate limiting on POST /meetings import endpoint
- Playwright-accessible UI flows

**Out of scope:**
- Docker daemon itself
- PostgreSQL direct access (port 5432 from external)
- Redis direct access (port 6379 from external)
- Any production infrastructure