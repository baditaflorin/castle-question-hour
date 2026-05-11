# 0013 — Testing strategy

- Status: accepted
- Date: 2026-05-11

## Decision

| Layer            | Tooling                                | Where                              | Gate           |
|------------------|----------------------------------------|------------------------------------|----------------|
| Backend unit     | stdlib `testing` + `testify`           | `*_test.go` next to source         | `make test`    |
| Backend integ    | `testcontainers-go`, build tag `integration` | `backend/test/integration/`  | `make test-integration` |
| Frontend unit    | Vitest                                 | `*.test.ts` next to source         | `make test`    |
| E2E happy path   | Playwright                             | `frontend/test/e2e/`               | `make smoke`   |
| Smoke (compose)  | bash script + curl + Playwright headless | `scripts/smoke.sh`               | `make smoke`   |

- Coverage target: ≥70% on `internal/` and on frontend logic modules (`src/lib/`, `src/state/`).
- Tests must run pre-push in under 60s (excluding `make smoke`, which is opt-in for releases).
- No flakies committed. If a test is flaky, quarantine with `t.Skip(reason)` and link a tracking
  issue, or fix it.

## What we deliberately don't test

- The exact phrasing of LLM output. We assert structure (3–7 themes, non-empty, no banned terms),
  not content.
- Piper's audio quality. We assert non-empty WAV with a sensible duration.
- Real Web Push delivery to FCM/APNs. We assert our handler hits a stub HTTP endpoint matching
  the W3C spec.
