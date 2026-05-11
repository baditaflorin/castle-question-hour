import { useState } from "react";
import { clsx } from "clsx";

type Props = {
  onJoin: (code: string) => void;
};

const CODE_RE = /^[a-z0-9]([a-z0-9-]{1,30}[a-z0-9])?$/;

export function JoinScreen({ onJoin }: Props) {
  const [code, setCode] = useState("");
  const valid = CODE_RE.test(code);

  return (
    <div className="mx-auto max-w-md p-6 mt-16 text-center">
      <h1 className="font-serif text-3xl text-candle mb-4">A castle of quiet questions</h1>
      <p className="text-bone/80 mb-8">
        Once an hour, a deep question lights up your phone. Answers stay anonymous and live on your
        phones — never our servers. At the next meal we'll read what came up.
      </p>
      <label className="block text-left text-sm text-bone/70 mb-1" htmlFor="code">
        Castle code
      </label>
      <input
        id="code"
        autoFocus
        spellCheck={false}
        autoCapitalize="none"
        autoComplete="off"
        value={code}
        onChange={(e) => setCode(e.target.value.trim().toLowerCase())}
        placeholder="the-grey-castle"
        className="w-full rounded-md bg-slate/80 border border-slate/60 px-3 py-2 text-bone placeholder:text-bone/40 focus:outline-none focus:border-candle"
      />
      <p className="text-xs text-bone/50 mt-2 text-left">
        Lowercase letters, digits, and hyphens. Whoever shares the code shares the room.
      </p>
      <button
        disabled={!valid}
        onClick={() => onJoin(code)}
        className={clsx(
          "mt-6 w-full rounded-md px-4 py-2 font-medium transition-colors",
          valid ? "bg-candle text-night hover:brightness-95" : "bg-slate/40 text-bone/40 cursor-not-allowed",
        )}
      >
        Enter the castle
      </button>
    </div>
  );
}
