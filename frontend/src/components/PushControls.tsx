import { useEffect, useState } from "react";
import { disablePush, enablePush, pushState, type PushState } from "../lib/push";

export function PushControls({ code }: { code: string }) {
  const [state, setState] = useState<PushState>({ status: "idle" });
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    pushState().then(setState);
  }, []);

  if (state.status === "unsupported") {
    return (
      <p className="text-xs text-bone/40">
        This browser can't deliver hourly nudges. Open the app at the top of each hour, or use a
        different phone.
      </p>
    );
  }
  if (state.status === "denied") {
    return (
      <p className="text-xs text-ember">
        Notifications are blocked. Enable them in browser settings if you want hourly pings.
      </p>
    );
  }

  return (
    <div className="flex items-center gap-3 text-xs">
      {state.status === "subscribed" ? (
        <button
          disabled={busy}
          className="text-bone/60 underline underline-offset-2 hover:text-bone"
          onClick={async () => {
            setBusy(true);
            try {
              setState(await disablePush(code));
            } finally {
              setBusy(false);
            }
          }}
        >
          Stop hourly pings
        </button>
      ) : (
        <button
          disabled={busy}
          className="text-candle underline underline-offset-2 hover:brightness-110"
          onClick={async () => {
            setBusy(true);
            try {
              setState(await enablePush(code));
            } finally {
              setBusy(false);
            }
          }}
        >
          Turn on hourly pings
        </button>
      )}
    </div>
  );
}
