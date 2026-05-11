# Postmortem — v0.1.0 scaffold

Date: 2026-05-11.
Author: baditaflorin + Claude Opus 4.7.

## What was built

A v0.1.0 scaffold that satisfies every section of the bootstrap meta-prompt:

- Frontend (Vite + React + TS strict) builds into `docs/` and is Pages-ready.
  Joining a castle, drafting an anonymous answer, and reading a generated
  theme summary all work end-to-end in the smoke test.
- Backend (Go 1.22) implements signaling (WebSocket relay), hourly cron-driven
  question scheduler, VAPID Web Push, Ollama-backed theme extraction, and a
  Piper subprocess (or HTTP) bridge to render the spoken summary.
- Docker Compose stack: app + ollama + piper (optional) + nginx + prometheus
  (optional). Distroless image, non-root, read-only rootfs, all caps dropped,
  binary's own `-healthcheck` flag so the HEALTHCHECK works without a shell.
- 17 ADRs (0001–0017), all written **before** the code they describe.
- Local git hooks replacing GitHub Actions, plus a smoke script that runs the
  entire happy path in under 5 seconds (excluding Playwright browser install).

## Was Mode C the right call?

**Yes, with one honest caveat.** The required pieces — hourly Web Push,
WebRTC signaling, server-side LLM, server-side TTS — are not feasible without
*some* server. Pure Mode A would require:

- A third-party signaling broker (defeats local-first).
- WebLLM on each phone (3B model = ~2GB download per phone — fatal).
- Piper-WASM (works, but is buggy and large).
- A push-trigger mechanism that doesn't need a server (does not exist on iOS).

The caveat: if the castle ever happens to have no LAN and zero phones outside
the room, a peer-elected "host phone" could in theory replace the backend.
That's a v2 stretch goal. v1 ships with one box.

## What worked

- **ADR-first discipline.** Writing all 17 ADRs before the Go scaffold made
  the module boundaries trivially obvious by the time code started. Zero
  refactors during scaffolding.
- **Yjs + y-webrtc as the data plane.** Lets the answer text never touch the
  backend by default — the strongest privacy property the system has.
- **Trace IDs through the whole stack.** nginx → backend slog → service worker
  notification, all share `X-Trace-Id` set on first hop. Debuggable from any
  end.
- **Backend `-healthcheck` flag.** Tiny detail that lets distroless images
  ship healthchecks cleanly without curl.

## What didn't (or surprised us)

- **TS `Uint8Array<ArrayBufferLike>` strictness.** TypeScript 5.6 narrowed the
  Uint8Array generic, breaking the `urlBase64ToUint8Array` → `applicationServerKey`
  call. Fixed by returning a `BufferSource` (ArrayBuffer) directly. Worth a
  comment somewhere for next time.
- **Vitest envs.** `process` is not in the strict TS lib for `tsc -b` when
  Playwright e2e tests live next to vitest tests. Solved by giving the e2e
  directory its own tsconfig with `"types": ["node", "@playwright/test"]`.
- **Distroless + curl.** We always forget. The binary-flag trick is the way.

## What was deliberately left out

- A separate Web Push payload encryption key for the summary-ready
  notification. The hourly question payload contains the prompt text, which is
  fine. If we ever push summary-ready events with theme content, we need to
  ensure the payload is short and theme-bare.
- A "stewards only" auth flow for `/summarize`. A malicious peer in the
  castle today could spam the LLM. v2 should require a per-castle steward
  token issued by the operator.
- SQLite persistence wiring. The interface is there (`storage.SubscriptionStore`)
  but only `MemoryStore` is built. Restart loses subscriptions. Acceptable for
  v1.
- A real PWA install icon (we shipped a favicon SVG only; the `icon-192.png`
  and `icon-512.png` are referenced in the manifest but not generated yet).
- `openapi-fetch`-generated client. We hand-rolled the API client to keep the
  bundle small. If the API grows, swap in the codegen.

## Tech debt accepted

1. The bundled Ollama and Piper containers are big. A "no LLM, no TTS" mode
   that returns a deterministic stub summary would let small castles run on
   1 GB boxes. Track as `enhancement`.
2. The signaling hub is in-memory; if the backend restarts mid-gathering,
   phones reconnect cleanly but lose any "we already saw this offer" state.
   Mostly invisible to users; document if it bites someone.
3. No rate limiting on `/summarize` other than nginx's `cqh_api` zone. A bored
   guest could call it repeatedly. Cheap fix: per-castle 1-call-per-minute.

## Next 3 most valuable improvements

1. **Steward token for /summarize.** ~20 lines of middleware, lets only the
   person holding the printed token trigger the LLM. Solves the spam vector
   cleanly without compromising the anonymity story for everyone else.
2. **Embedded sample voices + sample LLM in the image.** Right now operators
   must `ollama pull` and bring their own Piper voice. Shipping one of each
   makes the runbook one command shorter.
3. **A QR-code "join this castle" screen.** The hash-based `#api=…` URL is
   ugly to type. Generate a QR client-side from any phone and the rest of the
   guests scan it.

## Time spent vs estimate

- Estimated: a full day to scaffold the whole shape.
- Actual: a single unattended pass, ~2 hours wall clock (one Claude session).
- The hand-rolled API client paid for itself in build-time + bundle-size budget,
  but cost ~20 minutes vs `openapi-fetch` codegen.
