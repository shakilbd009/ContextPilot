# Security Baseline

> Version 2.0.0 · Mandatory reading for all engineers

> **Current state (2026-06-01):** this document is the policy baseline. Implementation status is tracked separately — see [STATUS.md](../STATUS.md) → "Recovery state". No feature is "Production-Ready" yet. The rules below still apply to every PR; they are not aspirational.

---

## Overview

ContextPilot processes meeting content that may include sensitive business information, attendee names, decisions, and action items. Security is not an afterthought — it is a first-class requirement baked into every BRD.

This document defines the security baseline for all implementation work. Deviations require a documented risk acceptance and an ADR.

---

## Data Classification

| Class | Examples | Handling |
|-------|----------|----------|
| **PII** | Attendee names, email addresses, phone numbers | Masked in logs; encrypted at rest |
| **Confidential** | Meeting transcripts, decisions, action items | Encrypted in transit and at rest; access controlled |
| **Public** | Meeting titles (if visible to all users) | Standard access |

---

## Secrets Management

1. **No secrets in code.** Database URLs, API keys, and tokens live in environment variables — never hardcoded.
2. **`.env.example`** is the canonical list of environment variables. It contains no real values (secrets are `***` or empty).
3. **`.env.local`** is git-ignored and never committed. It overrides `.env.example` for local dev.
4. **CI secrets** live in GitHub Actions secrets, not in the repo.
5. **Docker** does not bake secrets into images. Use `--env-file` or runtime environment injection.

---

## Input Validation

| Rule | Enforcement |
|------|-------------|
| All user input is untrusted | Validate on server; do not trust client-side validation alone |
| SQL injection | Use parameterized queries only; no string concatenation |
| XSS | Escape HTML in user-provided strings; set `Content-Security-Policy` |
| Path traversal | Validate file paths; do not allow `..` in user-provided paths |
| Email addresses | RFC 5322 validation |
| UUIDs | Validate format before DB query |
| Meeting titles | Max 256 chars; no HTML |

---

## Authentication & Sessions

- **Session management**: HTTP-only, Secure, SameSite=Strict cookies
- **Token format**: JWT RS256 (not HS256 — no shared secrets in frontend)
- **Session expiry**: 24-hour sliding window; 7-day absolute
- **CSRF protection**: Double-submit cookie pattern for state-changing operations
- **Rate limiting**: 100 req/min per user; burst 150

---

## Authorization

- Users see only meetings they own or are invited to
- Row-level security in PostgreSQL (RLS policies)
- Admin operations require separate admin role — never a superuser flag in application code
- Feature flags do not bypass authorization checks

---

## Privacy & Retention

- PII is retained for 90 days by default; configurable per organization
- Retention policy enforced in `ff_enable_privacy_retention_controls` (BRD-06)
- PII masking in logs: attendee names replaced with `[REDACTED]` in non-production environments
- Data export: users can export all their meeting data in JSON format (GDPR compliance)
- Data deletion: hard delete after retention period; no soft delete for PII

---

## Dependency Management

| Action | Frequency |
|--------|-----------|
| `go mod tidy && go vet` | Every PR |
| `pnpm audit` | Every PR |
| Trivy scan on Docker image | Every release |
| Dependency review (manual) | Monthly |

---

## Security Headers

All responses include:
```
Content-Security-Policy: default-src 'self'
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Referrer-Policy: strict-origin-when-cross-origin
```

---

## Vulnerability Reporting

- **Production security issues**: Follow the incident response process in `docs/security-incident-response.md` (future)
- **CVEs in dependencies**: File a security advisory on GitHub; patch within 48h for critical, 7 days for high
- **Penetration testing**: Annual, with findings tracked as security BRDs

---

## Threat Model

| Threat | Mitigation |
|--------|------------|
| Unauthorized meeting access | Row-level security + auth middleware |
| Injection (SQL, XSS) | Input validation + CSP |
| Session hijacking | HTTP-only Secure cookies + short expiry |
| Secrets leakage | Env vars only; no hardcoding; CI secrets |
| Data breach (PII) | Encryption at rest + access logging |
| Denial of service | Rate limiting + request timeout |

---

## Compliance Notes

- **GDPR**: User data export and deletion are required features (BRD-06)
- **SOC 2**: Audit logs for all data access; retention policy enforced
- **HIPAA**: Not in scope for v1; add BAA if healthcare customers targeted

---

## Secure Development Practices

1. **No `eval()`** in any language — code execution from user input is never acceptable
2. **No `exec()`** or `os.system()` with user-provided strings in Go
3. **No `innerHTML`** with user-provided strings in frontend (use textContent or sanitized HTML)
4. **No `document.write()`** in any frontend code
5. **No secret URLs** in error messages — only generic error IDs returned to clients

---

## Security Review Checklist (Pre-Merge)

- [ ] No secrets in code (use `trufflehog` or `git-secrets`)
- [ ] All input validated on server
- [ ] Auth checks on every protected route
- [ ] CSP header set
- [ ] Dependencies audited (`go vet`, `pnpm audit`)
- [ ] Feature flags do not bypass auth
- [ ] Error responses are generic (no stack traces in production)