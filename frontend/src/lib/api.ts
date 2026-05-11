// Tiny typed API client. Hand-rolled instead of openapi-fetch to keep the
// initial JS payload small; the OpenAPI spec is the source of truth for the
// shapes and we mirror them here.

import { z } from "zod";

export const Question = z.object({
  bucket_id: z.string(),
  text: z.string(),
  emitted_at: z.string(),
});
export type Question = z.infer<typeof Question>;

export const Theme = z.object({
  title: z.string(),
  summary: z.string(),
  samples: z.array(z.string()).optional(),
});
export type Theme = z.infer<typeof Theme>;

export const SummaryResponse = z.object({
  bucket_id: z.string(),
  question: z.string(),
  answer_count: z.number(),
  themes: z.array(Theme),
  audio_wav: z.string().optional(),
});
export type SummaryResponse = z.infer<typeof SummaryResponse>;

const apiBase = readApiBase();

function readApiBase(): string {
  // Allow override via URL hash so the same Pages build serves multiple castles:
  //   https://baditaflorin.github.io/castle-question-hour/#api=https://castle.example.local:25342
  if (typeof window !== "undefined" && window.location.hash) {
    const m = window.location.hash.match(/(?:^|[&#])api=([^&]+)/);
    if (m) return decodeURIComponent(m[1]).replace(/\/$/, "");
  }
  const v = (import.meta as ImportMeta).env?.VITE_API_BASE;
  if (typeof v === "string" && v.length > 0) return v.replace(/\/$/, "");
  // dev fallback — same origin (vite proxies /api → backend)
  return "";
}

async function getJSON<T>(path: string, schema: z.ZodSchema<T>): Promise<T> {
  const r = await fetch(`${apiBase}${path}`, { headers: { Accept: "application/json" } });
  if (!r.ok) throw new Error(`GET ${path} → ${r.status}`);
  const raw = await r.json();
  return schema.parse(raw);
}

async function postJSON<T>(path: string, body: unknown, schema: z.ZodSchema<T>): Promise<T> {
  const r = await fetch(`${apiBase}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!r.ok) throw new Error(`POST ${path} → ${r.status}`);
  const raw = await r.json();
  return schema.parse(raw);
}

async function postNoParse(path: string, body: unknown): Promise<void> {
  const r = await fetch(`${apiBase}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!r.ok) throw new Error(`POST ${path} → ${r.status}`);
}

async function delNoParse(path: string): Promise<void> {
  const r = await fetch(`${apiBase}${path}`, { method: "DELETE" });
  if (!r.ok) throw new Error(`DELETE ${path} → ${r.status}`);
}

export const api = {
  apiBase: () => apiBase,
  signalURL: (code: string) => {
    const httpBase = apiBase || window.location.origin;
    return httpBase.replace(/^http/, "ws") + `/api/v1/signal/${encodeURIComponent(code)}`;
  },
  vapidKey: () =>
    getJSON("/api/v1/vapid-public-key", z.object({ public_key: z.string() })).then(
      (r) => r.public_key,
    ),
  now: (code: string) => getJSON(`/api/v1/castle/${encodeURIComponent(code)}/now`, Question),
  history: (code: string, n = 24) =>
    getJSON(`/api/v1/castle/${encodeURIComponent(code)}/history?n=${n}`, z.array(Question)),
  subscribe: (code: string, sub: PushSubscription) =>
    postNoParse(`/api/v1/castle/${encodeURIComponent(code)}/subscribe`, subscriptionPayload(sub)),
  unsubscribe: (code: string, endpoint: string) =>
    delNoParse(
      `/api/v1/castle/${encodeURIComponent(code)}/subscribe?endpoint=${encodeURIComponent(endpoint)}`,
    ),
  summarize: (
    code: string,
    body: { bucket_id?: string; question: string; answers: string[]; with_audio?: boolean },
  ) => postJSON(`/api/v1/castle/${encodeURIComponent(code)}/summarize`, body, SummaryResponse),
};

function subscriptionPayload(sub: PushSubscription) {
  const json = sub.toJSON();
  return {
    endpoint: json.endpoint,
    p256dh: json.keys?.p256dh,
    auth: json.keys?.auth,
  };
}
