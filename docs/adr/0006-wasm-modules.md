# 0006 — WASM modules

- Status: accepted
- Date: 2026-05-11

## Decision

**No WASM in v1.** The brief uses local LLM + Piper on the backend (Mode C), so we don't need
WebLLM or piper-wasm in the browser. Y.js core is plain JS; y-webrtc is plain JS plus simple-peer.

If we later move to a Mode A fork (no backend), we'd revisit:
- WebLLM with a 3B model (~2GB download) → opt-in only.
- piper-wasm for browser TTS.

Until then this ADR stays "no WASM" so we don't pay the COOP/COEP and bundle-size costs for no
benefit.

## Consequences

- Vite config does not need `crossOriginIsolated` headers or any worker-pool plumbing.
- The Pages site is a plain SPA. No `_headers` workaround needed.
