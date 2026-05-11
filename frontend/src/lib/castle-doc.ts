// Y.Doc per castle, synced over WebRTC mesh, mirrored to IndexedDB so a phone
// that drops off can replay locally.
//
// Shape:
//   answersByBucket: Y.Map<string, Y.Array<string>>  // bucketID → list of answer texts
//
// We deliberately don't bind any author identifier to an answer. The Y.js clientID
// is random per session and not surfaced in the UI.

import * as Y from "yjs";
import { WebrtcProvider } from "y-webrtc";
import { IndexeddbPersistence } from "y-indexeddb";
import { api } from "./api";
import { fetchIceServers } from "./turnConfig";

export type CastleDoc = {
  doc: Y.Doc;
  rtc: WebrtcProvider;
  idb: IndexeddbPersistence;
  answers: (bucketId: string) => Y.Array<string>;
  appendAnswer: (bucketId: string, text: string) => void;
  countAnswers: (bucketId: string) => number;
  destroy: () => void;
};

export function openCastleDoc(code: string): CastleDoc {
  const doc = new Y.Doc();

  // The signaling URL is our backend; y-webrtc accepts an array of signaling servers.
  const signal = api.signalURL(code);
  // Start STUN-only; refresh with HMAC TURN credentials once they arrive.
  // y-webrtc does not let us mutate ICE servers on an existing provider, so
  // a freshly-fetched set will only apply on the next reconnect — that's
  // acceptable because the first browser session typically does get a few
  // reconnects from background-tab close codes anyway.
  const rtc = new WebrtcProvider(`cqh:${code}`, doc, {
    signaling: [signal],
    maxConns: 24,
    filterBcConns: true,
    peerOpts: { config: { iceServers: [{ urls: "stun:stun.l.google.com:19302" }] } },
  });

  void fetchIceServers().then((iceServers) => {
    // Stash the fresh credentials on the provider's peerOpts so future peer
    // connections (the ones y-webrtc creates lazily as it discovers peers)
    // pick them up. Existing already-negotiated peers keep their ICE.
    const opts = (rtc as unknown as { peerOpts?: { config?: { iceServers?: unknown } } }).peerOpts;
    if (opts?.config) opts.config.iceServers = iceServers;
  });

  const idb = new IndexeddbPersistence(`cqh:${code}`, doc);

  const answersByBucket = doc.getMap<Y.Array<string>>("answersByBucket");

  function answers(bucketId: string): Y.Array<string> {
    let arr = answersByBucket.get(bucketId);
    if (!arr) {
      arr = new Y.Array<string>();
      answersByBucket.set(bucketId, arr);
    }
    return arr;
  }

  function appendAnswer(bucketId: string, text: string) {
    const trimmed = text.trim();
    if (!trimmed) return;
    answers(bucketId).push([trimmed]);
  }

  function countAnswers(bucketId: string): number {
    return answersByBucket.get(bucketId)?.length ?? 0;
  }

  return {
    doc,
    rtc,
    idb,
    answers,
    appendAnswer,
    countAnswers,
    destroy() {
      rtc.destroy();
      idb.destroy();
      doc.destroy();
    },
  };
}
