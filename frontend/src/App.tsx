import { useState } from "react";
import { JoinScreen } from "./components/JoinScreen";
import { HourScreen } from "./components/HourScreen";
import { SummaryScreen } from "./components/SummaryScreen";
import { PushControls } from "./components/PushControls";
import { useCastleCode, useCastleDoc } from "./state/castle-store";
import type { Question } from "./lib/api";

type View = { kind: "hour" } | { kind: "summary"; question: Question };

export function App() {
  const [code, setCode] = useCastleCode();
  const cd = useCastleDoc(code);
  const [view, setView] = useState<View>({ kind: "hour" });

  if (!code) {
    return (
      <Shell>
        <JoinScreen onJoin={setCode} />
      </Shell>
    );
  }
  if (!cd) {
    return (
      <Shell>
        <div className="mx-auto max-w-md p-6 mt-16 text-center text-bone/60">
          Opening the castle…
        </div>
      </Shell>
    );
  }

  return (
    <Shell>
      {view.kind === "hour" ? (
        <HourScreen
          code={code}
          cd={cd}
          onShowSummary={(q) => setView({ kind: "summary", question: q })}
          onLeave={() => setCode("")}
        />
      ) : (
        <SummaryScreen
          code={code}
          cd={cd}
          question={view.question}
          onBack={() => setView({ kind: "hour" })}
        />
      )}
      <footer className="mx-auto max-w-xl p-6 mt-8 flex items-center justify-between text-xs text-bone/40 border-t border-slate/40">
        <PushControls code={code} />
        <span>v{import.meta.env.VITE_APP_VERSION ?? "dev"}</span>
      </footer>
    </Shell>
  );
}

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen candle-glow">
      <main>{children}</main>
    </div>
  );
}
