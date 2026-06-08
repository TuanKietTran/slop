# Integration Test Results

Date: 2026-06-08

## Build

`go build ./...` — **all packages compile successfully** (ingress, agent-cli, temporal/worker, temporal/client).

## Sample Product App (`samples/product-app/`)

Static HTML e-commerce app served on port 9090 via `serve.go`:
- `index.html` — product listing with 4 cards, search filter, cart badge
- `product.html` — product detail with quantity selector, Add to Cart
- `cart.html` — shopping cart with item list, totals, Checkout button

Started with `go run serve.go` — **serving successfully on :9090**.

## Ingress API (`ingress/`)

Started with `go run ./ingress` on port 8080.

- NATS not available (expected in local dev without NATS server) — logged warning and continued
- Task submission: `POST /tasks` → `{"id":"test-001","status":"queued"}` ✓
- Task listing: `GET /api/tasks` → returns queued task JSON ✓

## Agent CLI Pipeline Stages

### investigate

```
go run ./agent-cli investigate --site http://localhost:9090 --out /tmp/agent-out/
```

**Result:** PASSED  
Playwright driver v1.45.1 unavailable (version mismatch with installed playwright 1.56.1). Fell back to HTTP fetch mode. Saved `snapshot.json` with title "ShopDemo - Products" and 6 links.

### gen-tests

**Skipped** — `ANTHROPIC_API_KEY` not set. Requires live Anthropic API to generate test cases.

### verify

```
go run ./agent-cli verify --out /tmp/agent-out/ --site http://localhost:9090
```

**Result:** PASSED — 2/2 tests passed  
Playwright unavailable; fell back to HTTP reachability checks. Results saved to `results.json`.

## Playwright Test Suite (`samples/test-project/`)

28 tests across 3 spec files, running against http://localhost:9090 using Chromium 141.0.7390.37.

| Spec file | Tests | Passed | Failed |
|---|---|---|---|
| `product-listing.spec.ts` | 9 | 9 | 0 |
| `product-detail.spec.ts` | 12 | 12 | 0 |
| `cart.spec.ts` | 7 | 7 | 0 |
| **Total** | **28** | **28** | **0** |

All 28 tests passed in ~7s.

## Known Limitations / Notes

- **playwright-go v0.4501.1** requires playwright driver 1.45.1 but the environment has 1.56.1. The agent CLI `investigate` and `verify` commands fall back gracefully to HTTP-based checks. For full browser automation, upgrade `playwright-go` to `v0.5700.x` once the module is available in the cache, or install the matching playwright 1.45.1 driver.
- **gen-tests** requires `ANTHROPIC_API_KEY` to call the Claude API. Set the env var and re-run `go run ./agent-cli gen-tests` to generate Playwright test code from the site snapshot.
- **NATS** is not available in the local dev environment. The ingress server logs a warning and operates without pub/sub. The Temporal worker also requires a running Temporal server.
