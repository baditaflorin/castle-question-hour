// Minimal global castle session state: which code we're joined to, the live
// Y.Doc handle, and answer counts derived from it via subscribe-on-mount.

import { useEffect, useState } from "react";
import { type CastleDoc, openCastleDoc } from "../lib/castle-doc";

const STORAGE_KEY = "cqh:castle-code";
const CODE_RE = /^[a-z0-9]([a-z0-9-]{1,30}[a-z0-9])?$/;

// readCastleFromHash returns the value of `#castle=…` if present and valid,
// otherwise empty. The hash is preserved (we don't strip it) so a refresh
// re-applies the override.
function readCastleFromHash(): string {
  if (typeof window === "undefined") return "";
  const m = window.location.hash.match(/(?:^|[&#])castle=([^&]+)/);
  if (!m) return "";
  const decoded = decodeURIComponent(m[1]).trim().toLowerCase();
  return CODE_RE.test(decoded) ? decoded : "";
}

export function useCastleCode(): [string, (code: string) => void] {
  const [code, setCode] = useState<string>(() => {
    if (typeof localStorage === "undefined") return "";
    const fromHash = readCastleFromHash();
    if (fromHash) {
      localStorage.setItem(STORAGE_KEY, fromHash);
      return fromHash;
    }
    return localStorage.getItem(STORAGE_KEY) ?? "";
  });
  return [
    code,
    (next: string) => {
      const cleaned = next.trim().toLowerCase();
      setCode(cleaned);
      if (cleaned) localStorage.setItem(STORAGE_KEY, cleaned);
      else localStorage.removeItem(STORAGE_KEY);
    },
  ];
}

export function useCastleDoc(code: string): CastleDoc | null {
  const [doc, setDoc] = useState<CastleDoc | null>(null);
  useEffect(() => {
    if (!code) {
      setDoc(null);
      return;
    }
    const cd = openCastleDoc(code);
    setDoc(cd);
    return () => cd.destroy();
  }, [code]);
  return doc;
}

export function useAnswerCount(cd: CastleDoc | null, bucketId: string): number {
  const [n, setN] = useState(0);
  useEffect(() => {
    if (!cd || !bucketId) return;
    const arr = cd.answers(bucketId);
    const update = () => setN(arr.length);
    update();
    arr.observe(update);
    return () => arr.unobserve(update);
  }, [cd, bucketId]);
  return n;
}

// usePeerCount returns the number of *other* phones currently joined to the
// mesh, derived from y-webrtc awareness. Each peer publishes a heartbeat
// awareness state; we subtract self.
export function usePeerCount(cd: CastleDoc | null): number {
  const [n, setN] = useState(0);
  useEffect(() => {
    if (!cd) return;
    const aw = cd.rtc.awareness;
    // publish a minimal self-state so other peers count us
    aw.setLocalState({ joinedAt: Date.now() });
    const update = () => {
      const states = aw.getStates();
      setN(Math.max(0, states.size - 1));
    };
    update();
    aw.on("change", update);
    return () => {
      aw.off("change", update);
      aw.setLocalState(null);
    };
  }, [cd]);
  return n;
}

export function readAnswers(cd: CastleDoc | null, bucketId: string): string[] {
  if (!cd) return [];
  return cd.answers(bucketId).toArray();
}
