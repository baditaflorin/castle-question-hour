import { describe, it, expect } from "vitest";
import { Question, SummaryResponse, Theme } from "../src/lib/api";

describe("schemas", () => {
  it("parses a Question", () => {
    const q = Question.parse({
      bucket_id: "2026-05-11T10:00:00Z",
      text: "What scared you?",
      emitted_at: "2026-05-11T10:00:00Z",
    });
    expect(q.text).toBe("What scared you?");
  });

  it("rejects a malformed Question", () => {
    expect(() => Question.parse({ bucket_id: 1 })).toThrow();
  });

  it("parses a Theme with samples", () => {
    const t = Theme.parse({ title: "Grief", summary: "Many spoke of loss.", samples: ["a", "b"] });
    expect(t.samples).toHaveLength(2);
  });

  it("parses a SummaryResponse without audio", () => {
    const r = SummaryResponse.parse({
      bucket_id: "x",
      question: "y",
      answer_count: 5,
      themes: [],
    });
    expect(r.audio_wav).toBeUndefined();
  });
});
