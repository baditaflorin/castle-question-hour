# Security policy

## Supported versions

Only the `main` branch and the most recent tagged release are supported. v0.x is pre-1.0; expect
breaking changes between minor versions until v1.0.0.

## Reporting a vulnerability

Email **baditaflorin@gmail.com** with subject `castle-question-hour: security`. Include:

- The version / commit SHA you're testing against.
- Reproduction steps or a proof-of-concept.
- Impact assessment (what an attacker can do).

We aim to acknowledge reports within 72 hours and to ship a fix or mitigation within 14 days for
high-severity issues. Please do **not** open public GitHub issues for vulnerabilities.

## What's in scope

- The Go backend in `backend/` (signaling, scheduler, push, summarizer, TTS gateway).
- The frontend in `frontend/` shipped to GitHub Pages.
- The `deploy/` artifacts (compose, nginx, env templates).

## What's out of scope

- DoS against a self-hosted Ollama or Piper instance — those are operator concerns, documented in
  the runbook.
- Vulnerabilities in upstream dependencies that don't affect this codebase.
- Findings that require the attacker to already control the castle's local network.

## Hardening notes

- The frontend never holds secrets. VAPID public key is the only "credential" it sees and is
  intentionally public.
- The backend `:8080` is internal only — nginx terminates TLS on public port `25342`.
- `/metrics` is blocked at nginx.
- Submitted answers are stored in-memory by default; persistence to SQLite is opt-in via env var.
