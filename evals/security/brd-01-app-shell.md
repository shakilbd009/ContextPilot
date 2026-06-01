# Security Eval: brd-01-app-shell

> 🔴 Failing — implementation pending

## Scope

Security checks for the App Shell: XSS prevention, CSRF, content security policy, secure headers.

## Checks

| Check | Method |
|-------|--------|
| XSS | Reflected XSS in search params / URL |
| CSP | Content-Security-Policy header present |
| HTTPS | HSTS header on production |
| CSRF | CSRF token on form submissions |
| Auth | Unauthenticated access to `/meetings` redirects to login |

## Running

```bash
# Requires app running
make eval-security
```