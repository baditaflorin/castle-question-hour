/**
 * TURN credential fetcher.
 *
 * castle-question-hour ships its own backend signaling at /api/v1/signal/{code}
 * — see api.signalURL() — so we don't replace the signaling URL. We DO add
 * TURN credentials from a separate token server so peers behind symmetric NAT
 * (mobile carriers, corporate firewalls) have a working relay path.
 *
 * Default token endpoint: https://turn.0docker.com/credentials
 * Default relay:          turn:turn.0docker.com:3479
 *
 *  • https://github.com/baditaflorin/turn-token-server
 *  • https://github.com/baditaflorin/coturn-hetzner
 *
 * Override with VITE_TURN_TOKEN_URL at build time or with localStorage at
 * runtime. Set the value to an empty string to disable TURN entirely
 * (STUN-only fallback).
 */

const DEFAULT_TURN_TOKEN_URL = "https://turn.0docker.com/credentials";

const STUN_SERVERS = [
  { urls: "stun:stun.l.google.com:19302" },
  { urls: "stun:stun1.l.google.com:19302" },
];

type TurnCredentialResponse = {
  username: string;
  password: string;
  ttl: number;
  uris: string[];
};

export type IceServer = { urls: string; username?: string; credential?: string };

function loadTurnTokenUrl(): string {
  if (typeof localStorage === "undefined") return DEFAULT_TURN_TOKEN_URL;
  return (
    localStorage.getItem("cqh:turnTokenUrl") ??
    ((import.meta as ImportMeta).env?.VITE_TURN_TOKEN_URL as string | undefined) ??
    DEFAULT_TURN_TOKEN_URL
  );
}

export async function fetchIceServers(): Promise<IceServer[]> {
  const tokenUrl = loadTurnTokenUrl();
  if (!tokenUrl) return STUN_SERVERS;
  try {
    const res = await fetch(tokenUrl, { cache: "no-store" });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const cred = (await res.json()) as TurnCredentialResponse;
    if (!Array.isArray(cred.uris) || cred.uris.length === 0) {
      throw new Error("token server returned no TURN URIs");
    }
    return [
      ...STUN_SERVERS,
      ...cred.uris.map((u) => ({
        urls: u,
        username: cred.username,
        credential: cred.password,
      })),
    ];
  } catch (err) {
    console.warn("[turn] credential fetch failed, falling back to STUN-only:", err);
    return STUN_SERVERS;
  }
}
