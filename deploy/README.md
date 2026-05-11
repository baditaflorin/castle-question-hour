# Castle box deployment

This is the operator-facing guide for running the backend on a single Linux box
inside the castle. The frontend lives on GitHub Pages and does not need a server
of its own.

## Prerequisites

- A small Linux box reachable on your castle's local network (a NUC, Pi 5, or
  spare laptop is fine).
- Docker Engine 24+ and the `docker compose` plugin.
- A DNS name pointing at the box (e.g., `castle.example.local`) — or just an IP
  with a self-signed cert if you accept the browser warning.
- About 4 GB free RAM (Ollama is the heavy tenant).

## First-time setup

```bash
# 1. Pull the repo (or just this deploy/ dir + an .env file).
git clone https://github.com/baditaflorin/castle-question-hour
cd castle-question-hour

# 2. Configure secrets.
cp .env.example .env
# Mint Web Push keys (run anywhere — the keypair is just a credential).
docker run --rm ghcr.io/baditaflorin/castle-question-hour:latest /app/server -help >/dev/null 2>&1 || true
go run ./backend/cmd/vapid-keygen >> .env   # or build locally and run the binary

# 3. TLS certs. For Let's Encrypt, run certbot on the host and mount the path:
#    sudo certbot certonly --standalone -d castle.example.com
#    sudo cp /etc/letsencrypt/live/castle.example.com/fullchain.pem deploy/nginx/certs/
#    sudo cp /etc/letsencrypt/live/castle.example.com/privkey.pem  deploy/nginx/certs/
# For a quick local test you can self-sign:
#    openssl req -x509 -nodes -newkey rsa:2048 \
#      -keyout deploy/nginx/certs/privkey.pem \
#      -out    deploy/nginx/certs/fullchain.pem \
#      -days 30 -subj '/CN=castle.local'

# 4. Pull the prebuilt amd64 image from GHCR and bring the stack up.
cd deploy
docker compose pull
docker compose up -d

# 5. Tell Ollama to pull the model that internal/summarize uses by default.
docker exec -it cqh-ollama ollama pull llama3.2:3b

# 6. Verify.
curl -k https://castle.example.com:25342/healthz   # → {"status":"ok"}
curl -k https://castle.example.com:25342/readyz
```

The public surface is **port 25342** on the box. Phones load the static UI from
GitHub Pages and reach this box via:

```
https://baditaflorin.github.io/castle-question-hour/#api=https://castle.example.com:25342
```

The steward gives that URL to everyone at the gathering — usually as a QR code.

## Operations

### Restart / upgrade

```bash
cd deploy
docker compose pull         # fetch the latest image
docker compose up -d        # recreate with the new image
docker compose logs -f app  # watch the JSON logs
```

### Roll back

```bash
docker compose pull ghcr.io/baditaflorin/castle-question-hour:v0.1.2
APP_TAG=v0.1.2 docker compose up -d
```

### Stop the stack

```bash
docker compose down
```

### Backups

The only persisted state is the Ollama model cache (volume `ollama-models`) and
optional Push subscription SQLite at `$SQLITE_PATH`. Both can be re-created from
scratch — the model is re-downloaded and subscriptions get re-registered when
phones come back. No business-critical state needs backup.

### Logs

```bash
docker compose logs -f app | jq .
docker compose logs --since=1h nginx | jq .
```

### Observability (optional)

```bash
docker compose --profile observability up -d
# Prometheus on http://<box>:9090 (inside the internal network only; expose at your own risk).
# Import deploy/observability/grafana-dashboard.json into a Grafana of your choosing.
```

## Resource sizing

| Component   | RAM peak | CPU steady | Disk          |
|-------------|----------|------------|---------------|
| `app`       | ~80 MB   | <5 %       | <100 MB       |
| `ollama`    | 3–4 GB   | up to 1 vCPU during inference | ~2 GB per model |
| `nginx`     | ~30 MB   | <1 %       | minimal       |
| `prometheus`| ~150 MB  | <2 %       | 7-day TSDB    |

A 4 GB box runs the full stack comfortably. Less than that — drop Prometheus
and use a smaller Ollama model (`qwen2.5:1.5b`).

## Security notes

- `/metrics` is blocked at nginx (`return 404`).
- The app container runs as `nonroot`, with a read-only rootfs and all caps
  dropped.
- Backend listens only on the internal Docker bridge — never published.
- `.env` is gitignored and read by `docker compose` via `env_file:`.
- TLS 1.2+ only; HSTS + strict CSP set on every response.

See [SECURITY.md](../SECURITY.md) for the disclosure policy.
