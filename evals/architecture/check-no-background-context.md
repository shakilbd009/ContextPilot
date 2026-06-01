# check-no-background-context

> Fitness function: no bare `context.Background()` in production code.

## Why This Rule Exists

`context.Background()` creates a root context with no deadline, no cancellation, and no values. When used in production request handlers, it means:
- The request has no timeout — a slow downstream dependency can hang the connection indefinitely
- The context cannot be cancelled when the client disconnects
- Distributed tracing cannot propagate correctly (no trace ID)
- Resource cleanup (DB connections, goroutines) does not happen when the request ends

Production code must use:
- A context passed from the request (Echo's `c.Request().Context()`)
- A derived context with deadline or cancellation (`context.WithTimeout`, `context.WithCancel`)
- A structured logging context with request ID

## What This Checks

```bash
grep -rnE 'context\.Background\(\)' backend/ --include='*.go'
```

Excludes:
- `*_test.go` files (tests may need a standalone context)
- `main.go` or `cmd/*` files where background context is used for server initialization (not per-request)
- Files with `// ARCH_OK:` on the line above the usage

## Grace Period

None. `context.Background()` in request handlers is a correctness issue.

## Escape Hatch

```go
// ARCH_OK: server init — background context used for startup initialization, not per-request
func startServer() {
    srv := &http.Server{
        Addr:    ":8080",
        Handler: mux,
        BaseContext: func() context.Context { return context.Background() },
    }
}
```

## Acceptable Patterns

```go
// ACCEPTABLE — from request context
func handler(c echo.Context) error {
    ctx := c.Request().Context()
    return doSomething(ctx)
}

// ACCEPTABLE — with timeout
func handler(c echo.Context) error {
    ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
    defer cancel()
    return doSomething(ctx)
}

// ACCEPTABLE — test fixture
func TestWithBackground(t *testing.T) {
    ctx := context.Background() // ARCH_OK: test context
    // ...
}
```

## Phase 0 Behavior

Exits 0 when `backend/` is empty (no Go source to scan).