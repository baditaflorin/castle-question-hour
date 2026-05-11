// Service worker for castle-question-hour.
// Two responsibilities: (1) show the hourly Web Push, (2) cache the app shell
// for offline question replay.

const APP_SHELL = "cqh-shell-v1";
const PRECACHE = ["./", "./index.html", "./manifest.webmanifest"];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(APP_SHELL).then((c) => c.addAll(PRECACHE)).then(() => self.skipWaiting()),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== APP_SHELL).map((k) => caches.delete(k))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  const req = event.request;
  if (req.method !== "GET") return;
  // network-first for API
  if (req.url.includes("/api/")) return;
  event.respondWith(
    caches.match(req).then((cached) => cached || fetch(req)),
  );
});

self.addEventListener("push", (event) => {
  let payload;
  try {
    payload = event.data ? event.data.json() : {};
  } catch (e) {
    payload = { kind: "question", question: "A new question is here." };
  }
  const title = payload.castle ? `castle: ${payload.castle}` : "Castle question";
  const body =
    payload.kind === "summary-ready"
      ? "The hour's themes are ready to read."
      : payload.question || "A new question is here.";
  event.waitUntil(
    self.registration.showNotification(title, {
      body,
      icon: "./icon-192.png",
      badge: "./icon-192.png",
      tag: payload.bucket_id || "cqh-tick",
      renotify: false,
      data: payload,
    }),
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((clientsList) => {
      for (const client of clientsList) {
        if (client.url.includes(self.registration.scope) && "focus" in client) {
          return client.focus();
        }
      }
      return self.clients.openWindow(self.registration.scope);
    }),
  );
});
