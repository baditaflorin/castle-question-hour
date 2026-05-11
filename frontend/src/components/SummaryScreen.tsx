import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { api, APIError, type Question, type SummaryResponse } from "../lib/api";
import { readAnswers, useAnswerCount } from "../state/castle-store";
import type { CastleDoc } from "../lib/castle-doc";

type Props = {
  code: string;
  cd: CastleDoc;
  question: Question;
  onBack: () => void;
};

const STEWARD_TOKEN_KEY = "cqh:steward-token";

export function SummaryScreen({ code, cd, question, onBack }: Props) {
  const [withAudio, setWithAudio] = useState(true);
  const [audioURL, setAudioURL] = useState<string | null>(null);
  const audioRef = useRef<HTMLAudioElement>(null);

  const localCount = useAnswerCount(cd, question.bucket_id);

  const info = useQuery({
    queryKey: ["server-info"],
    queryFn: () => api.serverInfo(),
    staleTime: 60_000,
  });

  const [stewardToken, setStewardToken] = useState<string>(
    () => localStorage.getItem(STEWARD_TOKEN_KEY) ?? "",
  );

  const mut = useMutation<SummaryResponse, Error>({
    mutationFn: async () => {
      const answers = readAnswers(cd, question.bucket_id);
      return api.summarize(
        code,
        {
          bucket_id: question.bucket_id,
          question: question.text,
          answers,
          with_audio: withAudio,
        },
        { stewardToken: info.data?.steward_required ? stewardToken : undefined },
      );
    },
    onSuccess: (r) => {
      if (info.data?.steward_required && stewardToken) {
        localStorage.setItem(STEWARD_TOKEN_KEY, stewardToken);
      }
      if (r.audio_wav) {
        const wav = base64ToBlob(r.audio_wav, "audio/wav");
        const url = URL.createObjectURL(wav);
        setAudioURL(url);
      } else {
        setAudioURL(null);
      }
    },
    onError: (err) => {
      if (err instanceof APIError && err.status === 401) {
        // wipe a stale cached token so the steward sees the prompt again
        localStorage.removeItem(STEWARD_TOKEN_KEY);
        setStewardToken("");
      }
    },
  });

  useEffect(() => {
    return () => {
      if (audioURL) URL.revokeObjectURL(audioURL);
    };
  }, [audioURL]);

  const stewardRequired = info.data?.steward_required === true;
  const stewardMissing = stewardRequired && stewardToken.trim() === "";

  return (
    <div className="mx-auto max-w-xl p-6 mt-10">
      <button onClick={onBack} className="text-bone/60 text-sm mb-6 hover:text-bone">
        ← back
      </button>

      <h2 className="font-serif text-2xl text-candle mb-2">Meal-time summary</h2>
      <p className="text-bone/70 mb-1 italic">{question.text}</p>
      <p className="text-bone/40 text-xs mb-6">
        {localCount} {localCount === 1 ? "voice" : "voices"} visible from this phone's mesh state.
      </p>

      {stewardRequired && (
        <div className="mb-4 rounded-md bg-slate/40 p-3 border border-slate/40">
          <label className="block text-xs text-bone/60 mb-1" htmlFor="steward">
            Steward token (printed on the steward's card)
          </label>
          <input
            id="steward"
            type="password"
            autoComplete="off"
            spellCheck={false}
            value={stewardToken}
            onChange={(e) => setStewardToken(e.target.value)}
            placeholder="paste it here"
            className="w-full rounded bg-night/60 border border-slate/40 px-3 py-1.5 text-bone placeholder:text-bone/30 focus:outline-none focus:border-candle"
          />
        </div>
      )}

      <div className="flex items-center gap-3 mb-4">
        <label className="text-sm text-bone/80 flex items-center gap-2">
          <input
            type="checkbox"
            checked={withAudio}
            onChange={(e) => setWithAudio(e.target.checked)}
            className="accent-candle"
          />
          Also speak it (Piper)
        </label>
        <button
          onClick={() => mut.mutate()}
          disabled={mut.isPending || stewardMissing || localCount === 0}
          className="rounded-md bg-candle text-night px-4 py-1.5 font-medium hover:brightness-95 disabled:bg-slate/40 disabled:text-bone/40 disabled:cursor-not-allowed"
        >
          {mut.isPending ? "Listening to the room…" : "Make summary"}
        </button>
      </div>

      {localCount === 0 && !mut.data && (
        <p className="text-bone/50 text-sm italic">No one shared this hour yet.</p>
      )}

      {mut.isError && (
        <div className="rounded bg-ember/20 text-ember p-3 mb-4">{mut.error.message}</div>
      )}

      {mut.data && (
        <>
          <p className="text-xs text-bone/40 mb-2">
            {mut.data.answer_count} {mut.data.answer_count === 1 ? "voice" : "voices"} ·{" "}
            {mut.data.themes.length} {mut.data.themes.length === 1 ? "thread" : "threads"}
            {mut.data.mode === "heuristic" && (
              <span className="ml-2 px-1.5 py-0.5 rounded bg-slate/40 text-bone/60">
                without LLM
              </span>
            )}
          </p>
          {mut.data.themes.length === 0 ? (
            <p className="text-bone/50 italic">The room was quiet this hour.</p>
          ) : (
            <ol className="space-y-3">
              {mut.data.themes.map((t, i) => (
                <li key={i} className="rounded-md bg-slate/40 p-4 border border-slate/30">
                  <div className="text-candle font-serif text-xl">{t.title}</div>
                  <p className="text-bone/80 mt-1 leading-relaxed">{t.summary}</p>
                  {t.samples && t.samples.length > 0 && (
                    <ul className="text-bone/50 italic text-sm mt-2 list-disc list-inside space-y-0.5">
                      {t.samples.slice(0, 3).map((s, j) => (
                        <li key={j}>"{s}"</li>
                      ))}
                    </ul>
                  )}
                </li>
              ))}
            </ol>
          )}
          {audioURL && (
            <div className="mt-6">
              <audio ref={audioRef} controls src={audioURL} className="w-full" />
            </div>
          )}
        </>
      )}
    </div>
  );
}

function base64ToBlob(b64: string, contentType: string): Blob {
  const bin = atob(b64);
  const buf = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) buf[i] = bin.charCodeAt(i);
  return new Blob([buf], { type: contentType });
}
