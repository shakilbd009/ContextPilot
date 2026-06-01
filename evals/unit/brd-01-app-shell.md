# Unit Eval: brd-01-app-shell

> 🔴 Failing — implementation pending

## Scope

Unit tests for App Shell components: Button, Input, Card, Badge, Avatar, Alert, Spinner.

## Component Tests

| Component | Scenario |
|-----------|----------|
| Button | renders in default, hover, active, disabled, loading states |
| Button | primary, secondary, ghost variants render correctly |
| Input | renders with label and error message |
| Input | focus ring is visible on tab |
| Card | renders with hover effect |
| Badge | renders success/warning/danger/neutral variants |
| Avatar | renders image; initials fallback when no image |
| Alert | renders info/success/warning/error variants |
| Spinner | renders and animates |

## Running

```bash
# From frontend/ directory
pnpm test
```