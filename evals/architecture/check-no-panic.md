# check-no-panic

> Fitness function: no `panic()` in production code.

## Why This Rule Exists

`panic()` causes goroutines to crash and unwinds the stack. In a production HTTP server, an unhandled panic on a goroutine serving a request will crash the entire process, taking down all in-flight requests. This is unacceptable for a meeting intelligence service that users depend on before important calls.

Expected error handling uses:
- Return errors as values (`error`)
- Use middleware to catch and log errors
- Emit metrics for error rates
- Return RFC 7807 problem responses to clients

`panic()` is acceptable only in:
- Test files (`_test.go`)
- Benchmark files (`*_bench_test.go`)
- `init()` functions that validate configuration at startup
- Unsubscribe/cleanup handlers where returning is impossible

## What This Checks

```bash
grep -rnE 'panic\(' backend/ --include='*.go'
```

Excludes:
- `*_test.go` files
- Files with `// ARCH_OK:` on the line above the `panic(`

## Grace Period

None. Panics in production code are never acceptable.

## Escape Hatch

```go
// ARCH_OK: init validation — process cannot start without valid config
func init() {
    if cfg.DatabaseURL == "" {
        panic("DATABASE_URL is required")
    }
}
```

The `// ARCH_OK:` comment must appear on the line immediately above the `panic(`.

## Phase 0 Behavior

Exits 0 when `backend/` is empty (no Go source to scan).