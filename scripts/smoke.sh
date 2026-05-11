#!/usr/bin/env bash
# Build, start the backend on a free port, serve the Pages build statically,
# then assert healthz/readyz, the /api/v1/castle/<code>/now endpoint, and the
# Playwright happy path. Tear everything down on exit.

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

bold() { printf '\033[1m%s\033[0m\n' "$*"; }
note() { printf '  %s\n' "$*"; }

BACKEND_PORT="${BACKEND_PORT:-18080}"
FRONTEND_PORT="${FRONTEND_PORT:-14173}"

cleanup() {
  rc=$?
  # Kill anything still bound to either port — `( ... ) &` subshells lose track
  # of grandchildren, so PID-based cleanup alone is not enough.
  lsof -ti:"$BACKEND_PORT" 2>/dev/null | xargs -I{} kill -9 {} 2>/dev/null || true
  lsof -ti:"$FRONTEND_PORT" 2>/dev/null | xargs -I{} kill -9 {} 2>/dev/null || true
  [[ -n "${serve_dir:-}" ]] && rm -rf "$serve_dir" 2>/dev/null || true
  exit $rc
}
trap cleanup EXIT INT TERM

bold "→ build backend & frontend"
make build

bold "→ start backend on :$BACKEND_PORT"
SERVER_ADDR=":$BACKEND_PORT" \
  QUESTIONS_FILE="$repo_root/backend/configs/questions.json" \
  HOURLY_ENABLED=false \
  VAPID_PUBLIC_KEY="" VAPID_PRIVATE_KEY="" \
  PAGES_ORIGIN="http://127.0.0.1:$FRONTEND_PORT" \
  backend/bin/server &
backend_pid=$!

# wait for /healthz
for i in {1..40}; do
  if curl -fs "http://127.0.0.1:$BACKEND_PORT/healthz" >/dev/null 2>&1; then
    note "backend ready after ${i}x125ms"
    break
  fi
  sleep 0.125
done

bold "→ /healthz, /readyz"
curl -fs "http://127.0.0.1:$BACKEND_PORT/healthz" | grep -q '"ok"' && note "healthz OK"
curl -fs "http://127.0.0.1:$BACKEND_PORT/readyz"  | grep -q '"ok"' && note "readyz OK"

bold "→ /metrics is exposed inside the bridge"
curl -fs "http://127.0.0.1:$BACKEND_PORT/metrics" | grep -q '^cqh_castles_active' && note "metrics OK"

bold "→ GET /api/v1/castle/smoke-castle/now"
out="$(curl -fs "http://127.0.0.1:$BACKEND_PORT/api/v1/castle/smoke-castle/now")"
echo "$out" | grep -q '"bucket_id"' && note "now → bucket present"
echo "$out" | grep -q '"text"' && note "now → text present"

bold "→ mirror docs/ under base path /castle-question-hour/ and serve on :$FRONTEND_PORT"
# Vite builds with base /castle-question-hour/ so the generated HTML expects
# its assets at that path. Serve docs/ at the right subdirectory so the e2e
# loads the bundle exactly the way GitHub Pages will.
serve_dir="$(mktemp -d)"
mkdir -p "$serve_dir/castle-question-hour"
cp -a docs/. "$serve_dir/castle-question-hour/"
(cd "$serve_dir" && python3 -m http.server "$FRONTEND_PORT" >/dev/null 2>&1) &
frontend_pid=$!
sleep 0.5

bold "→ frontend index.html reachable"
curl -fs "http://127.0.0.1:$FRONTEND_PORT/castle-question-hour/" | grep -q 'castle-question-hour' && note "index OK"

if [[ "${SKIP_E2E:-0}" != "1" ]]; then
  if (cd frontend && [ -d node_modules ]); then
    bold "→ Playwright e2e (headless)"
    (
      cd frontend
      APP_URL="http://127.0.0.1:$FRONTEND_PORT/castle-question-hour/" \
        API_BASE="http://127.0.0.1:$BACKEND_PORT" \
        npx playwright test --reporter=line || {
          echo "playwright failed"
          exit 1
        }
    )
  else
    note "frontend/node_modules missing — skipping e2e (run: cd frontend && npm ci)"
  fi
fi

bold "✓ smoke passed"
