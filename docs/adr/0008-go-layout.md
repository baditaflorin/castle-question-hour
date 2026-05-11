# 0008 — Go backend layout

- Status: accepted
- Date: 2026-05-11

## Decision

Follow `golang-standards/project-layout` conventions:

```
backend/
  cmd/
    server/         main.go — wire-up only
    vapid-keygen/   one-shot key minter
  internal/
    castle/         session state & Y.Doc room registry
    signaling/      WebSocket relay for WebRTC handshake
    schedule/       hourly job, question selection
    push/           VAPID + WebPush sender
    summarize/      Ollama HTTP client + prompt template
    tts/            Piper subprocess / HTTP wrapper
    httpapi/        chi router, middlewares, handlers
    config/         env + file config loader
    logging/        slog JSON handler factory
    metrics/        prometheus registry & helpers
    storage/        sqlite (opt) + in-memory implementations
    utils/          HandleErrorOrLogWithMessages, common helpers
  pkg/              (empty initially — exported types live here once stable)
  api/openapi.yaml
  configs/questions.json
  test/integration/
```

Conventions:

- One `.go` file per concern. Max 300 lines without justification.
- Explicit DI — `cmd/server/main.go` constructs every collaborator and wires them.
- No `init()` magic, no package-level mutable state.
- Errors wrap with `%w`. Sentinel errors when callers need to branch.
- Use stdlib `slog` (JSON) — never `log.Printf`.
- Use the team's `utils.HandleErrorOrLogWithMessages(err, errMsg, successMsg)` helper at the call
  sites where logging context matters.

## Consequences

- Reviewers can navigate the codebase by domain. No "god" package.
- Every internal package exposes an interface with a `Real…` and `Mock…` implementation when
  there's a non-trivial collaborator (subprocesses, HTTP).
