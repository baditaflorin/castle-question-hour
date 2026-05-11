# Runbook

For operators with `ssh` into the castle box. The friendly first-time setup is
in [../deploy/README.md](../deploy/README.md); this page is for "something is
on fire" moments.

## Where stuff is

| Thing                          | Path                                                     |
|--------------------------------|----------------------------------------------------------|
| `.env` (secrets)               | `/srv/castle-question-hour/.env`                         |
| compose                        | `/srv/castle-question-hour/deploy/`                      |
| TLS certs                      | `deploy/nginx/certs/{fullchain,privkey}.pem`             |
| Ollama models                  | Docker volume `castle-question-hour_ollama-models`       |
| Optional SQLite (subs)         | path in `SQLITE_PATH` env var (default unset → in-memory) |

## Quick checks

```bash
docker compose ps
docker compose logs -f --tail=200 app  | jq '.'
curl -fsk https://localhost:25342/healthz
curl -fsk https://localhost:25342/readyz | jq .
```

## "The hourly push isn't firing"

1. `curl -fsk https://localhost:25342/readyz` — is the backend up?
2. `docker compose logs app | jq 'select(.msg == "hourly tick")'` — is the
   scheduler ticking? If not, check that `HOURLY_ENABLED=true` in `.env`.
3. `docker compose logs app | jq 'select(.msg == "push fanout failed")'` —
   what's the per-castle error?
4. Common causes:
   - VAPID keys not configured (`VAPID_PUBLIC_KEY`/`PRIVATE_KEY` empty in `.env`)
   - Subscriptions expired (`outcome="expired"` in `cqh_push_sent_total`) —
     phones need to re-opt-in
   - Apple Push Notification gateway intermittently rate-limiting; retry next hour

## "Themes look bad / generic"

1. Ollama model running? `docker exec cqh-ollama ollama list`
2. Pull a better model: `docker exec -it cqh-ollama ollama pull llama3.2:3b`
   (or `qwen2.5:7b` if you have the RAM).
3. Adjust the system prompt in `internal/summarize/summarize.go` and rebuild.
   (Yes, this requires a release. Don't hot-edit the running container.)

## "Piper is silent / crashes"

1. Check the binary path: `docker exec cqh-app /app/server -help 2>&1` —
   does it report the configured `PIPER_BIN`?
2. Look for `cqh_tts_subprocess_failures_total > 0` in metrics.
3. Fall back to `piper-http` profile: `docker compose --profile with-piper-http up -d`
   and set `PIPER_HTTP_URL=http://piper:10200` in `.env`.

## "I deployed a new tag and now /healthz is 502"

Recent images are pinned by tag (`APP_TAG=vX.Y.Z` in `.env`). To roll back:

```bash
APP_TAG=v0.1.2 docker compose up -d app
docker compose logs --tail=50 app
```

## "Disk is filling up"

```bash
docker system df
docker image prune -af --filter "until=168h"
docker exec cqh-ollama du -sh /root/.ollama
```

The single biggest disk hog is the Ollama model cache. To remove unused models:
`docker exec cqh-ollama ollama rm <model>`.

## Backups & recovery

- The only state worth backing up is the optional SQLite subscriptions file at
  `SQLITE_PATH`. Copy it offline if you want to.
- Pull subscriptions are also auto-recovered on next visit by each phone, so
  even a full wipe of the volume only means a one-time "tap to enable
  notifications again" for users.

## Emergency reset

```bash
cd /srv/castle-question-hour/deploy
docker compose down
docker volume rm castle-question-hour_ollama-models castle-question-hour_prom-data
docker compose up -d
```

After this you must `docker exec -it cqh-ollama ollama pull llama3.2:3b` to
get the model back.
