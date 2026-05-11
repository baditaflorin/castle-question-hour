import { useEffect, useRef, useState } from "react";
import QRCode from "qrcode";
import { api } from "../lib/api";

type Props = {
  code: string;
  onClose: () => void;
};

// JoinQR renders a QR code encoding the join URL for the current castle.
// Stewards show this on a laptop screen so guests can scan it from their
// phones instead of typing the hash URL.
//
// The encoded URL preserves the active API base (#api=…) when one was
// supplied, so a guest scanning a QR from inside a castle reaches the same
// backend as the steward.
export function JoinQR({ code, onClose }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [url, setUrl] = useState("");

  useEffect(() => {
    const origin = window.location.origin;
    const path = window.location.pathname; // e.g. /castle-question-hour/
    const apiBase = api.apiBase();
    const params = new URLSearchParams();
    params.set("castle", code);
    if (apiBase) params.set("api", apiBase);
    const joinURL = `${origin}${path}#${params.toString()}`;
    setUrl(joinURL);
    if (canvasRef.current) {
      QRCode.toCanvas(canvasRef.current, joinURL, {
        width: 280,
        margin: 1,
        color: { dark: "#1a1f2e", light: "#f5efe0" },
      }).catch((err) => console.warn("qr render failed", err));
    }
  }, [code]);

  return (
    <div
      className="fixed inset-0 z-10 flex items-center justify-center bg-night/90 p-6"
      role="dialog"
      aria-modal="true"
      aria-label="Castle join QR code"
      onClick={onClose}
    >
      <div
        className="max-w-sm rounded-lg bg-bone p-6 text-night shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="font-serif text-xl mb-3 text-center">Scan to join</h2>
        <canvas
          ref={canvasRef}
          className="mx-auto block"
          aria-label={`QR code for joining castle ${code}`}
        />
        <p className="mt-4 text-center text-sm text-night/70 break-all select-all">{url}</p>
        <p className="mt-3 text-center text-xs text-night/50">castle: {code}</p>
        <button
          className="mt-5 w-full rounded-md bg-night px-4 py-2 text-bone hover:brightness-110"
          onClick={onClose}
        >
          Done
        </button>
      </div>
    </div>
  );
}
