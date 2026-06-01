# Integration Eval: brd-01-app-shell

> 🔴 Failing — implementation pending

## Scope

Integration tests for App Shell navigation and routing in the SvelteKit app.

## Scenarios

| Scenario | Description |
|----------|-------------|
| Route guard | App shell routes redirect to `/` when flag is false |
| Layout | Layout component mounts with correct children |
| Nav state | Active route is highlighted in nav |

## Running

```bash
# Requires backend and frontend both running
make eval-integration
```