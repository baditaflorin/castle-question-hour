# 0012 — Metrics & observability

- Status: accepted
- Date: 2026-05-11

## Decision

Backend `/metrics` (Prometheus), blocked at nginx from public traffic. Operators can scrape from
inside the docker network.

### Domain counters/histograms

| Name                                      | Type      | Purpose                                          |
|-------------------------------------------|-----------|--------------------------------------------------|
| `cqh_http_requests_total`                 | counter   | RED — by route, method, status                   |
| `cqh_http_request_duration_seconds`       | histogram | RED — by route                                   |
| `cqh_castles_active`                      | gauge     | currently-joined castle sessions                 |
| `cqh_signal_peers_connected`              | gauge     | open WebSocket peers across all castles          |
| `cqh_push_sent_total`                     | counter   | by outcome (`ok`/`expired`/`error`)              |
| `cqh_summary_latency_seconds`             | histogram | full pipeline (LLM + Piper)                      |
| `cqh_questions_emitted_total`             | counter   | hourly question fires                            |
| `cqh_tts_subprocess_failures_total`       | counter   | Piper crashes / non-zero exits                   |

Frontend: **no analytics** in v1. Beacon-style telemetry would compromise the anonymity property.
If usage insight is needed, the steward can read the `cqh_castles_active` gauge from the box.

## Consequences

- A starter `prometheus.yml` and a Grafana dashboard JSON live under `deploy/observability/`
  (profile-gated in compose).
- All histograms use the default buckets unless we have a reason otherwise.
