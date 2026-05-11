# API

The full schema is in [backend/api/openapi.yaml](../backend/api/openapi.yaml).
This page is a quick-reference for humans.

Base URL on a deployed castle box: `https://castle.example.local:25342`.
Base URL in local dev: `http://localhost:8080` (Vite proxies `/api` from the
frontend dev server).

## Conventions

- Times: RFC3339 UTC.
- Castle code regex: `^[a-z0-9]([a-z0-9-]{1,30}[a-z0-9])?$`.
- Errors: `{"error":{"code":"snake_case","message":"human readable"}}`.

## Endpoints

### Liveness and readiness

```bash
curl -s https://castle.example.local:25342/healthz
# {"status":"ok"}

curl -s https://castle.example.local:25342/readyz
# {"status":"ok","bank_size":24,"version":"v0.1.0"}
```

### Get this hour's question

```bash
curl -s https://castle.example.local:25342/api/v1/castle/the-grey-castle/now
# {
#   "bucket_id":"2026-05-11T14:00:00Z",
#   "text":"What scared you most this year?",
#   "emitted_at":"2026-05-11T14:00:01Z"
# }
```

If the castle has never been seen before, the backend creates it and picks the
first question on the spot.

### Hour history (last N buckets)

```bash
curl -s 'https://castle.example.local:25342/api/v1/castle/the-grey-castle/history?n=12'
```

Returns questions only — answers are never stored on the backend.

### Subscribe / unsubscribe to hourly push

```bash
curl -s https://castle.example.local:25342/api/v1/vapid-public-key
# {"public_key":"BG…"}

curl -s -XPOST -H 'content-type: application/json' \
  https://castle.example.local:25342/api/v1/castle/the-grey-castle/subscribe \
  -d '{"endpoint":"https://fcm.googleapis.com/…","p256dh":"…","auth":"…"}'

curl -s -XDELETE \
  'https://castle.example.local:25342/api/v1/castle/the-grey-castle/subscribe?endpoint=https://fcm.googleapis.com/…'
```

### Trigger the meal-time summary

```bash
curl -s -XPOST -H 'content-type: application/json' \
  https://castle.example.local:25342/api/v1/castle/the-grey-castle/summarize \
  -d '{
    "bucket_id":"2026-05-11T14:00:00Z",
    "question":"What scared you most this year?",
    "answers":[
      "Telling my brother I forgive him.",
      "Quitting and not having a plan.",
      "Asking for help when I usually don't."
    ],
    "with_audio": true
  }'
# {
#   "bucket_id":"2026-05-11T14:00:00Z",
#   "question":"What scared you most this year?",
#   "answer_count": 3,
#   "themes":[
#     {"title":"Honesty with kin","summary":"Several spoke of long-deferred truth-telling.","samples":["…"]},
#     {"title":"Leaping without a net","summary":"…"}
#   ],
#   "audio_wav":"UklGRiQ…"   // base64-encoded WAV (only when with_audio=true)
# }
```

### Signaling WebSocket

`ws(s)://<host>/api/v1/signal/<castle-code>` — the y-webrtc protocol.
Envelope shape: `{"type":"publish|subscribe|unsubscribe|ping","topics":[…],"data":…}`.
The backend is a pure relay — it never decodes `data`.

## Status code conventions

| Code | When                                                   |
|------|--------------------------------------------------------|
| 200  | Success.                                               |
| 400  | Bad input (invalid castle code, malformed JSON).       |
| 404  | Resource not found (rare — most endpoints autocreate). |
| 502  | LLM unreachable on `/summarize`.                       |
| 500  | Unexpected internal error (logged with `trace_id`).    |
