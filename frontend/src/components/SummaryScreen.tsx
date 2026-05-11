import { useEffect, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { api, type Question, type SummaryResponse } from "../lib/api";
import { readAnswers } from "../state/castle-store";
import type { CastleDoc } from "../lib/castle-doc";

type Props = {
  code: string;
  cd: CastleDoc;
  question: Question;
  onBack: () => void;
};

export function SummaryScreen({ code, cd, question, onBack }: Props) {
  const [withAudio, setWithAudio] = useState(true);
  const [audioURL, setAudioURL] = useState<string | null>(null);
  const audioRef = useRef<HTMLAudioElement>(null);

  const mut = useMutation<SummaryResponse, Error>({
    mutationFn: async () => {
      const answers = readAnswers(cd, question.bucket_id);
      return api.summarize(code, {
        bucket_id: question.bucket_id,
        question: question.text,
        answers,
        with_audio: withAudio,
      });
    },
    onSuccess: (r) => {
      if (r.audio_wav) {
        const wav = base64ToBlob(r.audio_wav, "audio/wav");
        const url = URL.createObjectURL(wav);
        setAudioURL(url);
      } else {
        setAudioURL(null);
      }
    },
  });

  useEffect(() => {
    return () => {
      if (audioURL) URL.revokeObjectURL(audioURL);
    };
  }, [audioURL]);

  return (
    <div className="mx-auto max-w-xl p-6 mt-10">
      <button onClick={onBack} className="text-bone/60 text-sm mb-6 hover:text-bone">
        ← back
      </button>

      <h2 className="font-serif text-2xl text-candle mb-2">Meal-time summary</h2>
      <p className="text-bone/70 mb-6 italic">{question.text}</p>

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
          disabled={mut.isPending}
          className="rounded-md bg-candle text-night px-4 py-1.5 font-medium hover:brightness-95 disabled:bg-slate/40 disabled:text-bone/40"
        >
          {mut.isPending ? "Listening to the room…" : "Make summary"}
        </button>
      </div>

      {mut.isError && (
        <div className="rounded bg-ember/20 text-ember p-3 mb-4">
          {mut.error.message}
        </div>
      )}

      {mut.data && (
        <>
          <p className="text-xs text-bone/40 mb-2">
            {mut.data.answer_count} {mut.data.answer_count === 1 ? "voice" : "voices"} — {mut.data.themes.length} threads
          </p>
          <ol className="space-y-3">
            {mut.data.themes.map((t, i) => (
              <li key={i} className="rounded-md bg-slate/40 p-4 border border-slate/30">
                <div className="text-candle font-serif text-lg">{t.title}</div>
                <p className="text-bone/80 mt-1">{t.summary}</p>
                {t.samples && t.samples.length > 0 && (
                  <ul className="text-bone/50 italic text-sm mt-2 list-disc list-inside">
                    {t.samples.slice(0, 3).map((s, j) => (
                      <li key={j}>"{s}"</li>
                    ))}
                  </ul>
                )}
              </li>
            ))}
          </ol>
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
