# Events Contract

> Phase 0 placeholder. Async event schemas will be defined per BRD as features are implemented.

## Overview

ContextPilot uses events for:
- Cross-service communication (future)
- Audit trail (future)
- Real-time notifications (future)

## Event Categories

| Category | Description | Status |
|----------|-------------|--------|
| Meeting events | Created, updated, deleted, completed | Planned |
| Briefing events | Generated, delivered | Planned |
| User events | Joined, left, preferences changed | Planned |

## Event Envelope

All events follow this envelope:

```json
{
  "id": "evt_uuid",
  "type": "meeting.created",
  "version": "1",
  "timestamp": "ISO 8601",
  "source": "contextpilot-backend",
  "correlation_id": "req_uuid",
  "payload": {}
}
```

## Delivery Guarantees

- At-least-once delivery for all events
- Events are idempotent via `event_id`
- Replay is supported for up to 30 days

## Status

This is a placeholder. Event schemas will be added when BRD-05 (Provider Connectors) or later BRDs require async processing.