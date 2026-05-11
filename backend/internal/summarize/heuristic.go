package summarize

import (
	"sort"
	"strings"
	"unicode"
)

// Heuristic returns a deterministic theme breakdown derived purely from word
// frequencies and co-occurrences. Used when the LLM is unreachable so the
// ritual still happens, just at lower fidelity.
//
// Algorithm:
//  1. Tokenize each answer, drop stopwords and short tokens.
//  2. Score each token by frequency across answers (DF, not TF).
//  3. Pick up to maxThemes seed tokens by descending score.
//  4. Assign each answer to its highest-scoring matching seed.
//  5. The seed becomes the theme Title (Title-Cased), the answers become
//     its Samples (first sentence trimmed), the Summary is a one-line
//     composition of the seed and the sample count.
//
// This is intentionally simple. It is not pretending to be the LLM.
func Heuristic(answers []string, maxThemes int) *Result {
	if maxThemes <= 0 {
		maxThemes = 5
	}
	if len(answers) == 0 {
		return &Result{Mode: "heuristic", AnswerCount: 0}
	}

	type seed struct {
		token string
		df    int
	}

	df := map[string]int{}
	tokensPerAnswer := make([][]string, len(answers))
	for i, a := range answers {
		toks := normalize(a)
		tokensPerAnswer[i] = toks
		seen := map[string]struct{}{}
		for _, t := range toks {
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			df[t]++
		}
	}

	seeds := make([]seed, 0, len(df))
	for tok, n := range df {
		if n < 2 {
			continue // a unique word isn't a theme; needs at least two mentions
		}
		seeds = append(seeds, seed{tok, n})
	}
	sort.Slice(seeds, func(i, j int) bool {
		if seeds[i].df != seeds[j].df {
			return seeds[i].df > seeds[j].df
		}
		return seeds[i].token < seeds[j].token
	})
	if len(seeds) > maxThemes {
		seeds = seeds[:maxThemes]
	}

	// fallback: if no token recurs, produce one "mixed reflections" theme
	if len(seeds) == 0 {
		samples := firstSentences(answers, 3)
		return &Result{
			Mode:        "heuristic",
			AnswerCount: len(answers),
			Themes: []Theme{
				{
					Title:   "Mixed reflections",
					Summary: "Each answer stood on its own — no single thread surfaced more than once.",
					Samples: samples,
				},
			},
		}
	}

	// assign answers to seeds
	themes := make([]Theme, len(seeds))
	for i, s := range seeds {
		themes[i] = Theme{Title: titleCase(s.token)}
	}
	seedIdx := map[string]int{}
	for i, s := range seeds {
		seedIdx[s.token] = i
	}
	for i, toks := range tokensPerAnswer {
		bestIdx := -1
		bestRank := len(seeds) + 1
		for _, t := range toks {
			if r, ok := seedIdx[t]; ok && r < bestRank {
				bestRank = r
				bestIdx = r
			}
		}
		if bestIdx == -1 {
			continue // this answer doesn't match any seed; skipped from samples
		}
		first := firstSentence(answers[i])
		if len(themes[bestIdx].Samples) < 3 {
			themes[bestIdx].Samples = append(themes[bestIdx].Samples, first)
		}
	}

	// fill in summaries
	out := themes[:0]
	for _, t := range themes {
		if len(t.Samples) == 0 {
			continue
		}
		t.Summary = paraphrase(t.Title, len(t.Samples))
		out = append(out, t)
	}

	// if everything was filtered, fall back to the mixed-reflections theme
	if len(out) == 0 {
		samples := firstSentences(answers, 3)
		return &Result{
			Mode:        "heuristic",
			AnswerCount: len(answers),
			Themes: []Theme{
				{
					Title:   "Mixed reflections",
					Summary: "Several voices, no shared thread.",
					Samples: samples,
				},
			},
		}
	}

	return &Result{Mode: "heuristic", AnswerCount: len(answers), Themes: out}
}

// normalize lowercases the text, strips punctuation, splits on whitespace,
// and drops short or stopword tokens.
func normalize(s string) []string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '-', r == ' ':
			b.WriteRune(r)
		default:
			b.WriteByte(' ')
		}
	}
	fields := strings.Fields(b.String())
	out := fields[:0]
	for _, f := range fields {
		if len(f) < 4 {
			continue
		}
		if _, stop := stopwords[f]; stop {
			continue
		}
		out = append(out, f)
	}
	return out
}

func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	for i, r := range s {
		if r == '.' || r == '!' || r == '?' || r == '\n' {
			return strings.TrimSpace(s[:i+1])
		}
	}
	if len(s) > 160 {
		return strings.TrimSpace(s[:160]) + "…"
	}
	return s
}

func firstSentences(answers []string, n int) []string {
	if n > len(answers) {
		n = len(answers)
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, firstSentence(answers[i]))
	}
	return out
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func paraphrase(token string, sampleCount int) string {
	switch sampleCount {
	case 1:
		return "One voice spoke about " + strings.ToLower(token) + "."
	case 2:
		return "Two voices touched on " + strings.ToLower(token) + "."
	default:
		return "Several voices touched on " + strings.ToLower(token) + "."
	}
}

// stopwords is a small English stoplist tuned for short reflective answers.
// Not exhaustive; just enough to keep "I/that/this/with" from dominating themes.
var stopwords = map[string]struct{}{
	"about": {}, "after": {}, "again": {}, "against": {}, "almost": {},
	"already": {}, "also": {}, "always": {}, "another": {}, "anyone": {},
	"anything": {}, "around": {}, "because": {}, "before": {}, "being": {},
	"below": {}, "between": {}, "could": {}, "didn": {}, "does": {},
	"doesn": {}, "doing": {}, "down": {}, "during": {}, "each": {},
	"even": {}, "ever": {}, "every": {}, "everything": {}, "from": {},
	"goes": {}, "going": {}, "gone": {}, "have": {}, "having": {},
	"hers": {}, "herself": {}, "himself": {}, "into": {}, "itself": {},
	"just": {}, "kind": {}, "know": {}, "later": {}, "less": {},
	"like": {}, "made": {}, "make": {}, "many": {}, "might": {},
	"more": {}, "most": {}, "much": {}, "must": {}, "myself": {},
	"never": {}, "nothing": {}, "ones": {}, "only": {}, "other": {},
	"others": {}, "ourselves": {}, "over": {}, "same": {}, "should": {},
	"shouldn": {}, "since": {}, "some": {}, "something": {}, "still": {},
	"such": {}, "take": {}, "than": {}, "that": {}, "their": {},
	"theirs": {}, "them": {}, "themselves": {}, "then": {}, "there": {},
	"these": {}, "they": {}, "thing": {}, "things": {}, "think": {},
	"this": {}, "those": {}, "through": {}, "thus": {}, "very": {},
	"want": {}, "well": {}, "were": {}, "what": {}, "when": {},
	"where": {}, "which": {}, "while": {}, "with": {}, "would": {},
	"year": {}, "your": {}, "yours": {}, "yourself": {},
}
