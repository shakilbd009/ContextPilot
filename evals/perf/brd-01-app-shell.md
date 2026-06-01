# Performance Eval: brd-01-app-shell

> 🔴 Failing — implementation pending

## Scope

Performance benchmarks for App Shell: FCP, TTIR, bundle size, Lighthouse score.

## Benchmarks

| Metric | Target |
|--------|--------|
| First Contentful Paint | < 1.5s on 3G |
| Time to Interactive | < 3s on 3G |
| Lighthouse Performance | >= 90 |
| JS Bundle (initial) | < 150KB gzipped |
| CSS Bundle | < 30KB gzipped |

## Running

```bash
# Lighthouse CI
make eval-perf
```