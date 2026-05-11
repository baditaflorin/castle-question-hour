# 0017 — Dependency policy

- Status: accepted
- Date: 2026-05-11

## Decision

Production-ready libraries only. No bespoke implementations of solved problems.

### Backend (Go) — picked

- HTTP router: **chi**
- Config: **viper** + **envconfig**
- WebSocket: **coder/websocket** (the maintained fork of nhooyr/websocket)
- Web Push: **SherClockHolmes/webpush-go**
- Validator: **go-playground/validator/v10**
- Logging: **stdlib slog** (JSON handler)
- Metrics: **prometheus/client_golang**
- HTTP client: **stdlib net/http** with timeouts + retry-on-5xx wrapper
- Test: **stdlib testing** + **stretchr/testify**, **testcontainers-go** for integ
- DB (opt): **mattn/go-sqlite3** (only when `SQLITE_PATH` set)
- Cron: **robfig/cron/v3**

### Frontend (TS) — picked

- **vite**, **react@18**, **typescript** (strict)
- **yjs**, **y-webrtc**, **y-indexeddb**
- **@tanstack/react-query**, **zod**
- **tailwindcss**, **clsx**
- **openapi-typescript** + **openapi-fetch** (generated client)
- **vitest** + **@playwright/test**

### Versioning rules

- Pin to a known-good minor; allow patch upgrades via `^` only after a clean `go mod tidy`/`npm
  audit` pass.
- `govulncheck` and `npm audit` run in `pre-push`. Highs and criticals block.
- Document every new dependency in a one-line comment in the manifest if its purpose isn't
  obvious from the name.

## Forbidden

- Authentication libraries we don't need (v1 has no accounts).
- Heavy UI frameworks (MUI, Ant) — overkill for three screens.
- Hand-rolled CRDTs — we have Yjs.
