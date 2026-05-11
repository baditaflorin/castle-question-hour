// Minimal global castle session state: which code we're joined to, the live
// Y.Doc handle, and answer counts derived from it via subscribe-on-mount.

import { useEffect, useState } from "react";
import { type CastleDoc, openCastleDoc } from "../lib/castle-doc";

const STORAGE_KEY = "cqh:castle-code";

export function useCastleCode(): [string, (code: string) => void] {
  const [code, setCode] = useState<string>(() => {
    if (typeof localStorage === "undefined") return "";
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

export function readAnswers(cd: CastleDoc | null, bucketId: string): string[] {
  if (!cd) return [];
  return cd.answers(bucketId).toArray();
}
