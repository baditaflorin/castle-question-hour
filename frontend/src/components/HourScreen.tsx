import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { api, type Question } from "../lib/api";
import { useAnswerCount, usePeerCount } from "../state/castle-store";
import { JoinQR } from "./JoinQR";
import type { CastleDoc } from "../lib/castle-doc";

type Props = {
  code: string;
  cd: CastleDoc;
  onShowSummary: (q: Question) => void;
  onLeave: () => void;
};

export function HourScreen({ code, cd, onShowSummary, onLeave }: Props) {
  const { data, isLoading, refetch, isError } = useQuery<Question>({
    queryKey: ["now", code],
    queryFn: () => api.now(code),
    refetchInterval: 30_000,
  });

  const [draft, setDraft] = useState("");
  const [posted, setPosted] = useState(false);
  const [showQR, setShowQR] = useState(false);
  const bucketId = data?.bucket_id ?? "";
  const count = useAnswerCount(cd, bucketId);
  const peers = usePeerCount(cd);

  // refetch right after the top of the hour
  useEffect(() => {
    const now = new Date();
    const msToNextHour = 60_000 * (60 - now.getMinutes()) - now.getSeconds() * 1000;
    const t = window.setTimeout(() => refetch(), msToNextHour + 1500);
    return () => window.clearTimeout(t);
  }, [data?.bucket_id, refetch]);

  // reset post state on new hour
  useEffect(() => setPosted(false), [bucketId]);

  const placeholder = useMemo(
    () =>
      ["Whatever comes up first…", "Anonymous. One paragraph is fine.", "Whisper, don't perform."][
        bucketId.length % 3
      ],
    [bucketId],
  );

  return (
    <div className="mx-auto max-w-xl p-6 mt-10">
      <header className="flex items-center justify-between text-xs text-bone/50 mb-8 gap-2">
        <span className="flex-shrink-0">
          castle: <span className="text-bone/80">{code}</span>
        </span>
        <span className="flex-1 text-center" aria-live="polite">
          {peers > 0 ? `with ${peers} ${peers === 1 ? "other" : "others"} · ` : ""}
          {count} {count === 1 ? "answer" : "answers"} so far
        </span>
        <span className="flex items-center gap-3 flex-shrink-0">
          <button
            onClick={() => setShowQR(true)}
            className="underline underline-offset-2 hover:text-bone"
            aria-label="Show join QR code"
          >
            invite
          </button>
          <button onClick={onLeave} className="underline underline-offset-2 hover:text-bone">
            leave
          </button>
        </span>
      </header>
      {showQR && <JoinQR code={code} onClose={() => setShowQR(false)} />}

      {isError && (
        <div className="rounded bg-ember/20 text-ember p-3 mb-4">
          Couldn't reach the castle backend. Check the URL after <code>#api=</code>.
        </div>
      )}

      <h1 className="font-serif text-2xl md:text-3xl text-candle leading-snug mb-6 min-h-[3.5rem]">
        {isLoading ? "…" : data?.text}
      </h1>

      {!posted ? (
        <>
          <textarea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder={placeholder}
            rows={6}
            className="w-full rounded-md bg-slate/70 border border-slate/50 p-3 text-bone placeholder:text-bone/40 focus:outline-none focus:border-candle resize-vertical"
          />
          <div className="mt-3 flex items-center justify-between gap-3">
            <p className="text-xs text-bone/40">
              Stays on the mesh of phones in this castle. No name. No account.
            </p>
            <button
              disabled={!draft.trim()}
              onClick={() => {
                if (!bucketId) return;
                cd.appendAnswer(bucketId, draft);
                setDraft("");
                setPosted(true);
              }}
              className="rounded-md bg-candle text-night px-4 py-2 font-medium hover:brightness-95 disabled:bg-slate/40 disabled:text-bone/40 disabled:cursor-not-allowed"
            >
              Share
            </button>
          </div>
        </>
      ) : (
        <div className="rounded-md bg-slate/40 p-4 text-bone/80">
          Thanks. Your words are with the others, anonymously.
          <div className="mt-3">
            <button
              className="text-candle underline underline-offset-2"
              onClick={() => setPosted(false)}
            >
              Share another thought
            </button>
          </div>
        </div>
      )}

      <div className="mt-12 border-t border-slate/40 pt-6 flex items-center justify-between">
        <span className="text-xs text-bone/40">Next prompt at the top of the hour.</span>
        {data && (
          <button
            onClick={() => onShowSummary(data)}
            className="text-sm text-candle underline underline-offset-2 hover:brightness-110"
          >
            Read the meal-time summary →
          </button>
        )}
      </div>
    </div>
  );
}
