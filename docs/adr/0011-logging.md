# 0011 — Logging strategy

- Status: accepted
- Date: 2026-05-11

## Decision

- Backend logs go to **stdout** as **JSON** via stdlib `slog`.
- Default level `info`; configurable via `LOG_LEVEL` (`debug|info|warn|error`).
- Every request handler has a `trace_id` (ULID) attached to its logger context.
- Sensitive fields (answer text, push endpoint URLs, VAPID private key) are **never** logged at
  any level. We log lengths and hashes.
- Subprocess output from Piper goes to a child logger with `component=piper`, level `debug`.

Frontend:

- Production build: no `console.log`. Only `console.warn`/`console.error` for unexpected paths.
- Browser errors caught by a single error boundary at the app root; surfaced as a toast.

## Consequences

- `docker compose logs` is JSON — pipe through `jq` for human reading.
- Operators can grep for `trace_id` end-to-end (browser sets `X-Trace-Id` from `crypto.randomUUID`
  if the env opts in; otherwise the server mints it).
