# 0014 — Error handling

- Status: accepted
- Date: 2026-05-11

## Decision

### Backend

- Errors wrap with `fmt.Errorf("…: %w", err)`. Never swallowed.
- Sentinel errors in each package where callers branch (`ErrCastleNotFound`, `ErrLLMTimeout`,
  `ErrPiperUnavailable`).
- The `internal/utils.HandleErrorOrLogWithMessages(err, errMsg, successMsg)` helper logs and
  returns the err, normalizing call-site shape.
- Handlers convert errors to `{"error": {"code", "message"}}` via a small `httpapi.Error` helper.
  Stack traces are logged, never serialized.
- `panic` is forbidden in normal flow. A top-level `recover` middleware turns panics into 500s
  with a logged trace, exit code unchanged.

### Frontend

- Top-level `<ErrorBoundary>` catches React render errors; shows a "something broke" panel with a
  reload button.
- All async ops use TanStack Query → automatic retry on transient network errors, surfacing the
  final error to a toast.
- `zod.parse` boundary at every backend response → invalid responses produce a clear error
  rather than mysterious downstream nulls.

## Consequences

- Linter rule: `errcheck` enabled in `.golangci.yml`. CI (local hooks) fail on unchecked errors.
- Banned: `_ = err`. If we really mean to ignore, add a comment explaining why.
