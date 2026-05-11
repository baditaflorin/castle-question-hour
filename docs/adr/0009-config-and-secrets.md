# 0009 — Configuration & secrets management

- Status: accepted
- Date: 2026-05-11

## Decision

- All configuration via **environment variables**. Documented in
  [.env.example](../../.env.example).
- **viper** loads env + an optional `/etc/cqh/config.yaml` (for non-secret operator settings).
- **envconfig** binds env to a `config.Config` struct with validator tags.
- Secrets (VAPID private key) live in `.env` on the host. `.env` is gitignored. The
  `docker-compose.yml` references it via `env_file:`. The VAPID *public* key is served at runtime
  via `GET /api/v1/vapid-public-key` so the frontend bundle is secret-free.
- `gitleaks` in `pre-commit` catches accidental commits of `.env`, `*.pem`, or tokens.

## Consequences

- Operator runs `make vapid` once to seed `.env`, then never touches the keys again.
- We never need a secret manager (Vault, SOPS) for v1 — single-box, single-operator deploys.
- If we ever multi-tenant, this ADR is superseded.
