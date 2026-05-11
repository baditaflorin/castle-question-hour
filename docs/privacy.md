# Privacy

This page describes what castle-question-hour collects, where it lives, and
who can see it. It is not legal advice; it is the operator's honest map.

## TL;DR

- **Your answer text is not stored on any server we run.** It rides a WebRTC
  mesh between the phones of people in your castle and is mirrored to each
  phone's IndexedDB.
- The castle backend handles signaling, hourly notifications, and on-demand
  theme extraction. It can see your answer text **only** during the few
  seconds the steward holds the "make a summary" button.
- No analytics, no tracking, no third-party scripts on the frontend by default.
- No account, no profile, no name field. The Yjs `clientID` per phone is a
  random number; it isn't shown anywhere.

## What's collected and where it lives

| Datum                          | Where             | Retention                            |
|--------------------------------|-------------------|--------------------------------------|
| Castle code                    | Phone localStorage; backend RAM | Phone: until you "leave the castle". Backend: until container restart. |
| Push subscription endpoint     | Backend memory (and SQLite if `SQLITE_PATH` set) | Until you turn off pings, your browser drops it, or the operator wipes the box. |
| Answer text                    | Phone IndexedDB + WebRTC mesh   | Phone: until you uninstall or "leave the castle". Mesh: as long as one peer holds it. |
| Theme summaries                | Phone memory of the requesting session only | Discarded when the steward closes the summary screen. |
| Web Push notification payload  | Browser push service (FCM/APNs) | A few seconds in transit. Contains the question text and your castle code. |
| Server logs                    | Docker JSON file driver         | 7 days, 30 MB total cap by default. No answer text. |

## What the operator's box can see

If you're worried about a malicious operator: they can see, on the backend:

- Which castle codes are active.
- Which push endpoints are subscribed to which castle.
- Trace IDs and timings of HTTP requests.
- The full payload of any `/summarize` call the steward initiates (because the
  steward's phone POSTs the assembled answers to the backend for LLM and TTS).

They **cannot** see, on the backend:

- The bare contents of the Yjs CRDT (answers + question history) at any time
  other than during a `/summarize` call.
- A persistent record of who wrote which answer.

The Web Push payload of an hourly notification contains the question text and
the castle code. It does NOT contain answers — answers flow only over the
WebRTC mesh.

## Third-party scripts

There are none on the Pages frontend. The bundle contains React, Yjs,
TanStack Query, Zod, and our own code. No analytics. No tracking pixels. The
service worker only talks to the operator's backend.

## Web Push gateways

Notifications are delivered by your browser's push service:

- Chrome / Edge / Brave / Opera → Firebase Cloud Messaging (Google)
- Safari → Apple Push Notification service
- Firefox → Mozilla autopush

These gateways see your endpoint (an opaque URL they assigned you) and the
encrypted payload. The payload is encrypted with your subscription's keys, so
neither Google nor Apple nor Mozilla can read the question text — only your
device can.

## Your rights

You can:

- Turn off hourly pings any time (button on the app footer).
- "Leave the castle" — clears the castle code from localStorage; we do not
  follow you to the next session.
- Delete your local data: clear site data for the Pages URL in your browser.
  This drops the Yjs IndexedDB store and the Push subscription.

If the castle operator runs a custom build that adds analytics, they're
responsible for documenting that here — the upstream project has none.
