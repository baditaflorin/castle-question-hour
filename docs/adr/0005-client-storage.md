# 0005 — Client-side storage strategy

- Status: accepted
- Date: 2026-05-11

## Decision

- **IndexedDB** via `y-indexeddb` persists the Yjs document so a phone that drops off the mesh
  (lock-screen, low signal) can rejoin and replay answers without bothering peers.
- **localStorage** for *non-sensitive* preferences only: castle code, display name (optional),
  push opt-in.
- **No** cookies. No localStorage entry contains personally identifying answer content.
- The service worker keeps the latest hour's question cached so the notification handler can
  display it without a network round-trip.

## Why not OPFS

OPFS would be lovely for audio caching (summary WAVs), but iOS Safari support is incomplete and
the answer corpus is small enough that IndexedDB is fine. Re-evaluate if we add long-running
recordings.

## Privacy property

If the phone is lost or seized, the data on it is the Yjs CRDT — i.e., the answers everyone in
that castle has shared during sessions the phone joined. The user explicitly accepts this when
joining. There is no identity binding between an answer and a phone in the CRDT itself, but the
*set membership* (who joined which castle) is visible. Document this in the privacy doc.
