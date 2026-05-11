# 0002 — Architecture overview & module boundaries

- Status: accepted
- Date: 2026-05-11

## Context

We need clear seams so the moving parts — WebRTC mesh, scheduler, push, LLM, TTS — don't bleed
into each other. The codebase will grow.

## Decision

Two top-level apps with hard boundaries:

```
frontend/                     Vite + TS + React + Yjs (Pages)
backend/                      Go API (Docker)
  cmd/server/                 main entry
  cmd/vapid-keygen/           one-shot helper to mint VAPID keys
  internal/
    castle/                   castle session + Yjs doc state
    signaling/                WebRTC signaling over WebSocket
    schedule/                 hourly cron + question bank
    push/                     Web Push sender (VAPID)
    summarize/                Ollama client + theme extraction
    tts/                      Piper invoker
    httpapi/                  chi router, handlers, middleware
    config/                   viper + envconfig
    logging/                  slog setup
    metrics/                  prometheus collectors
    storage/                  in-memory + optional SQLite
    utils/                    HandleErrorOrLogWithMessages
  api/openapi.yaml            REST contract
  configs/questions.json      seed question bank
```

The browser only talks to the backend over three surfaces:

1. `GET /api/v1/castle/{code}/now` — current question for this hour.
2. `WS /api/v1/signal/{code}` — Yjs awareness + WebRTC signaling.
3. `POST /api/v1/castle/{code}/summarize` — steward triggers theme extraction + TTS.

The phone-to-phone data plane (Yjs Y.Doc with the answers) is **WebRTC mesh** — the backend never
sees raw answer text unless the steward asks for a summary, at which point the steward's device
posts the assembled text via TLS.

## Consequences

- Answers stay in the mesh by default. The backend's role is signaling, scheduling, and a
  privileged-on-demand summarizer.
- Each internal package is owned by one file or a tight group of files (§4: no file >300 lines).
- Tests can mock each internal package via interfaces — no global state, explicit DI in `cmd/server`.

## Alternatives considered

- Backend as the authoritative Yjs server (`y-websocket` style): would centralize answer text on
  the backend, which weakens the anonymity story. Rejected.
- A monolith with no `internal/` segmentation: would let the schedule package import the TTS
  package and vice versa; rejected for testability.
