package httpapi

import (
	"fmt"
	"strings"

	"github.com/baditaflorin/castle-question-hour/backend/internal/summarize"
)

// composeSpoken renders themes into a paragraph Piper can read aloud at a meal.
// Deliberately gentle and brief — this is a ritual, not a report.
func composeSpoken(question string, themes []summarize.Theme) string {
	if len(themes) == 0 {
		return "We sat with: " + question + ". No one shared yet."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Earlier we sat with: %s.\n", strings.TrimRight(question, "."))
	b.WriteString("Across the answers, these threads came up.\n")
	for i, t := range themes {
		fmt.Fprintf(&b, "%d. %s. %s\n", i+1, t.Title, strings.TrimSpace(t.Summary))
	}
	b.WriteString("May the table hold these.")
	return b.String()
}
