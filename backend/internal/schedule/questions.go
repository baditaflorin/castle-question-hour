// Package schedule owns the question bank and the hourly cron that emits a
// new question to every active castle.
package schedule

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"sync"
)

var ErrEmptyBank = errors.New("question bank is empty")

// Bank is a thread-safe collection of curated reflection prompts.
type Bank struct {
	mu     sync.Mutex
	items  []string
	cursor map[string]int // castle-code → index into items, for round-robin uniqueness
	rng    *rand.Rand
}

// LoadBank reads a JSON array of strings from path.
func LoadBank(path string) (*Bank, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read questions file %q: %w", path, err)
	}
	var items []string
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, fmt.Errorf("parse questions file %q: %w", path, err)
	}
	if len(items) == 0 {
		return nil, ErrEmptyBank
	}
	src := rand.NewPCG(0xCA571E, 0x91FF1E)
	return &Bank{
		items:  items,
		cursor: make(map[string]int),
		rng:    rand.New(src),
	}, nil
}

// NewBank is the in-memory variant for tests.
func NewBank(items []string) (*Bank, error) {
	if len(items) == 0 {
		return nil, ErrEmptyBank
	}
	src := rand.NewPCG(0xCA571E, 0x91FF1E)
	return &Bank{items: items, cursor: make(map[string]int), rng: rand.New(src)}, nil
}

// NextFor picks the next question for a given castle code. Within a castle,
// we walk the shuffled deck so the same prompt isn't repeated within one cycle.
func (b *Bank) NextFor(castleCode string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	idx, ok := b.cursor[castleCode]
	if !ok || idx >= len(b.items) {
		// shuffle a per-castle view by rotating with a random offset
		offset := b.rng.IntN(len(b.items))
		b.cursor[castleCode] = offset
		return b.items[offset]
	}
	q := b.items[idx]
	b.cursor[castleCode] = idx + 1
	return q
}

// Size returns the bank size.
func (b *Bank) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.items)
}
