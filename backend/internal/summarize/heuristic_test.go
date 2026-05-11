package summarize

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeuristic_EmptyInput(t *testing.T) {
	r := Heuristic(nil, 5)
	assert.Equal(t, "heuristic", r.Mode)
	assert.Equal(t, 0, r.AnswerCount)
	assert.Empty(t, r.Themes)
}

func TestHeuristic_GroupsByRepeatedToken(t *testing.T) {
	answers := []string{
		"Quitting my job and not having a plan scared me.",
		"Telling my brother the truth scared me.",
		"Quitting my band when no one wanted me to leave.",
		"Asking my father for help when I usually don't.",
	}
	r := Heuristic(answers, 5)

	require.Equal(t, "heuristic", r.Mode)
	require.Equal(t, 4, r.AnswerCount)
	// At least one theme should surface around "quitting" (df=2) or "scared" (df=2).
	require.GreaterOrEqual(t, len(r.Themes), 1, "expected at least one repeated theme")

	titles := make([]string, len(r.Themes))
	for i, t := range r.Themes {
		titles[i] = t.Title
	}
	assert.Contains(t, joinTitles(titles), "Quitting", "themes: %v", titles)
}

func TestHeuristic_NoRepeatsProducesMixed(t *testing.T) {
	answers := []string{
		"Unique aardvark whispered.",
		"Glittery zebra wandered.",
		"Crisp narwhal vanished.",
	}
	r := Heuristic(answers, 5)
	require.Len(t, r.Themes, 1)
	assert.Equal(t, "Mixed reflections", r.Themes[0].Title)
	assert.Len(t, r.Themes[0].Samples, 3)
}

func TestHeuristic_RespectsMaxThemes(t *testing.T) {
	answers := []string{
		"Grief grief grief.", "Grief and grief.",
		"Hope hope hope.", "Hope and hope.",
		"Wonder wonder wonder.", "Wonder and wonder.",
		"Trust trust trust.", "Trust and trust.",
	}
	r := Heuristic(answers, 2)
	assert.LessOrEqual(t, len(r.Themes), 2)
}

func TestNormalize_DropsStopwordsAndShortTokens(t *testing.T) {
	toks := normalize("This is the thing about that")
	assert.Empty(t, toks)
}

func TestFirstSentence_TruncatesGracefully(t *testing.T) {
	long := "This sentence has no terminator " + strings.Repeat("x", 200)
	out := firstSentence(long)
	// 160-char prefix plus a 3-byte ellipsis rune
	assert.LessOrEqual(t, len(out), 163)
	assert.Contains(t, out, "…")
}

func joinTitles(ts []string) string {
	out := ""
	for _, t := range ts {
		out += t + "|"
	}
	return out
}
