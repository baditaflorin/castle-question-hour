# castle-question-hour

Anonymous hourly reflection prompts with local peer sync, theme summaries, and spoken meal-time recaps.

[![Live frontend](https://img.shields.io/badge/pages-live-blue)](https://baditaflorin.github.io/castle-question-hour/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Mode C](https://img.shields.io/badge/mode-C-orange)](docs/adr/0001-deployment-mode.md)

> Once an hour a deep question lights up everyone's phone. Answers are anonymous, aggregated locally
> via Yjs over a WebRTC mesh, then surfaced as themes — and spoken aloud by Piper — at the next meal.

## What it is

A castle-wide ritual stack for temporary in-person communities (retreats, residencies, weekend
gatherings, off-sites). Frontend on GitHub Pages, runtime backend on a single Docker host inside
the local network (`:25342` behind nginx).

## Quickstart

```bash
git clone https://github.com/baditaflorin/castle-question-hour
cd castle-question-hour
make install-hooks         # wire local git hooks
make dev                   # frontend dev server on :5173, backend on :8080
make smoke                 # build + boot ephemeral compose + happy-path
```

Open `http://localhost:5173`, enter a castle code, and you'll get the current hour's question.

## Architecture in one paragraph

The browser joins a **Yjs** document over a **WebRTC mesh**. The Docker backend runs an
**hourly scheduler** that picks a question, sends a **Web Push** notification to every subscribed
device, and — when a castle steward calls **"meal time"** — pulls the hour's anonymous answers
out of the Yjs document, sends them to a **local LLM** (Ollama by default) for theme extraction,
and pipes the summary through **Piper** to produce a WAV the castle speakers can play.

Full picture: [docs/architecture.md](docs/architecture.md). Decisions: [docs/adr/](docs/adr/).
Deploy: [deploy/README.md](deploy/README.md). Runbook: [docs/runbook.md](docs/runbook.md).

### WebRTC infrastructure

The Go backend serves WebSocket **signaling** at `/api/v1/signal/{code}` — it's part of this app, not a third party.

For TURN relay (used when peers can't connect directly — symmetric NAT, corporate firewalls, mobile carrier networks) the browser fetches time-limited HMAC credentials from the [turn-token-server](https://github.com/baditaflorin/turn-token-server) running at `https://turn.0docker.com/credentials` by default. Relay endpoint is [coturn-hetzner](https://github.com/baditaflorin/coturn-hetzner) at `turn:turn.0docker.com:3479`. Override with `VITE_TURN_TOKEN_URL`. Set empty to disable TURN (STUN-only).

Why split signaling from TURN? Signaling is a small, app-aware service (it knows about castle codes, ratelimits, etc.) so it lives in this app's backend. TURN is a commodity relay shared across many apps, so it lives in a separate stack that can be reused.

## Repo layout

```
backend/        Go API: signaling, scheduler, summarizer, TTS gateway, push
frontend/       Vite + TS app — builds into docs/ for GitHub Pages
docs/           Published Pages site (built artifact) + ADRs + design docs
deploy/         docker-compose, nginx, env templates
scripts/        smoke, dev, release helpers
.githooks/      pre-commit, commit-msg, pre-push
Makefile        single source of truth
```

## Non-goals (v1)

- No user accounts, profiles, or identity tracking.
- No cloud-hosted answer database.
- No moderation marketplace, social feed, or long-term analytics.

## Success metrics

- 10+ participants join the same castle session from phones.
- Hourly prompt reaches ≥90% of opted-in active devices during a local test.
- Anonymous answers sync across participants within 10 seconds on the local network.
- Meal summary produces 3–7 coherent themes from submitted answers.
- Piper can speak the summary locally — no cloud TTS dependency.

## License

MIT — see [LICENSE](LICENSE).
