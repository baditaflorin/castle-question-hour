// Web Push subscription glue. The VAPID public key is fetched at runtime from
// the backend so the frontend bundle stays secret-free (ADR 0009).

import { api } from "./api";

export type PushState =
  | { status: "unsupported" }
  | { status: "denied" }
  | { status: "idle" }
  | { status: "subscribed"; endpoint: string };

export async function pushState(): Promise<PushState> {
  if (!("serviceWorker" in navigator) || !("PushManager" in window)) {
    return { status: "unsupported" };
  }
  if (Notification.permission === "denied") return { status: "denied" };
  const sw = await navigator.serviceWorker.ready;
  const sub = await sw.pushManager.getSubscription();
  return sub ? { status: "subscribed", endpoint: sub.endpoint } : { status: "idle" };
}

export async function enablePush(castleCode: string): Promise<PushState> {
  if (!("serviceWorker" in navigator) || !("PushManager" in window)) {
    return { status: "unsupported" };
  }
  const perm = await Notification.requestPermission();
  if (perm !== "granted") return { status: "denied" };

  const sw = await navigator.serviceWorker.ready;
  const existing = await sw.pushManager.getSubscription();
  if (existing) {
    await api.subscribe(castleCode, existing);
    return { status: "subscribed", endpoint: existing.endpoint };
  }
  const vapid = await api.vapidKey();
  const sub = await sw.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(vapid),
  });
  await api.subscribe(castleCode, sub);
  return { status: "subscribed", endpoint: sub.endpoint };
}

export async function disablePush(castleCode: string): Promise<PushState> {
  if (!("serviceWorker" in navigator)) return { status: "unsupported" };
  const sw = await navigator.serviceWorker.ready;
  const sub = await sw.pushManager.getSubscription();
  if (sub) {
    await api.unsubscribe(castleCode, sub.endpoint);
    await sub.unsubscribe();
  }
  return { status: "idle" };
}

function urlBase64ToUint8Array(base64: string): BufferSource {
  const padding = "=".repeat((4 - (base64.length % 4)) % 4);
  const b64 = (base64 + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(b64);
  const buf = new ArrayBuffer(raw.length);
  const view = new Uint8Array(buf);
  for (let i = 0; i < raw.length; ++i) view[i] = raw.charCodeAt(i);
  return buf;
}
