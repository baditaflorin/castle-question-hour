# 0015 — Deployment topology

- Status: accepted
- Date: 2026-05-11

## Decision

```
phones (LAN)               GitHub Pages (Internet)
  │                                │
  │       static UI fetch          │
  │ ◄──────────────────────────────┘
  │
  │  WS + REST (TLS) → castle.example.local:25342
  ▼
┌─────────────────────────── castle box ────────────────────────────┐
│  nginx :443 (public :25342)  ──►  app :8080  (internal bridge)    │
│                                    │                              │
│                                    ├──► ollama :11434             │
│                                    ├──► piper subprocess          │
│                                    └──► sqlite (optional volume)  │
└───────────────────────────────────────────────────────────────────┘
```

- Single box, single `docker compose up -d`.
- Image: `ghcr.io/baditaflorin/castle-question-hour:vX.Y.Z` (amd64, distroless).
- nginx terminates TLS via Let's Encrypt (operator's responsibility; certbot path documented).
- Backend listens on internal `:8080`, never published directly.
- Public port **`25342`** as mandated by §17.
- Ollama and Piper run as sibling containers / sibling processes; the app talks to them by name
  inside the bridge network.

## Consequences

- The Pages frontend is configured at build time with `VITE_API_BASE` — different castles can run
  on different hostnames without rebuilding Pages by passing the API host as a URL hash, e.g.,
  `https://baditaflorin.github.io/castle-question-hour/#api=https://castle.example.local:25342`.
- Operator owns DNS + TLS for the castle box. If they can't terminate TLS, the frontend can run
  locally via `make pages-preview` for an on-prem-only demo.

## Alternatives considered

- Kubernetes: enormously oversized for one box. Rejected.
- systemd units + bare Go binary (no Docker): would force operators to install Go, Piper, Ollama
  by hand. Rejected for portability.
