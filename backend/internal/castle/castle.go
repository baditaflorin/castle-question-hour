// Package castle owns the in-memory state of joined castle sessions:
// connected peers, the current hour bucket, and the lifecycle of subscriptions.
//
// We deliberately don't store answer text here — that lives in the Yjs doc
// on phones. This struct keeps only what the backend needs for signaling,
// scheduling, and on-demand summarization.
package castle

import (
	"errors"
	"regexp"
	"sync"
	"time"
)

var (
	ErrInvalidCode    = errors.New("castle code is invalid")
	ErrCastleNotFound = errors.New("castle not found")

	codeRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,30}[a-z0-9])?$`)
)

// Question is the prompt active for a given hour bucket.
type Question struct {
	BucketID string    `json:"bucket_id"` // RFC3339 truncated to the hour
	Text     string    `json:"text"`
	EmittedAt time.Time `json:"emitted_at"`
}

// PushSubscription is a Web Push endpoint registered by a phone.
type PushSubscription struct {
	Endpoint string `json:"endpoint"`
	P256DH   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

// Castle holds per-room runtime state. Thread-safe.
type Castle struct {
	Code            string
	CurrentQuestion Question
	History         []Question
	Subscriptions   map[string]PushSubscription // keyed by endpoint
	PeerCount       int

	mu sync.Mutex
}

// Registry is the set of all live castles in the process.
type Registry struct {
	mu      sync.RWMutex
	castles map[string]*Castle
}

func NewRegistry() *Registry {
	return &Registry{castles: make(map[string]*Castle)}
}

// ValidateCode returns ErrInvalidCode if code doesn't match the contract in ADR 0004.
func ValidateCode(code string) error {
	if !codeRE.MatchString(code) {
		return ErrInvalidCode
	}
	return nil
}

// GetOrCreate returns the castle for code, creating one if absent.
func (r *Registry) GetOrCreate(code string) (*Castle, error) {
	if err := ValidateCode(code); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.castles[code]
	if !ok {
		c = &Castle{
			Code:          code,
			Subscriptions: make(map[string]PushSubscription),
		}
		r.castles[code] = c
	}
	return c, nil
}

// Get returns the castle or ErrCastleNotFound.
func (r *Registry) Get(code string) (*Castle, error) {
	if err := ValidateCode(code); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.castles[code]
	if !ok {
		return nil, ErrCastleNotFound
	}
	return c, nil
}

// All returns a snapshot of every live castle (for fan-out, e.g. hourly push).
func (r *Registry) All() []*Castle {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Castle, 0, len(r.castles))
	for _, c := range r.castles {
		out = append(out, c)
	}
	return out
}

// SetQuestion records q as the active question and pushes the prior one onto History.
func (c *Castle) SetQuestion(q Question) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.CurrentQuestion.BucketID != "" {
		c.History = append(c.History, c.CurrentQuestion)
		if len(c.History) > 168 { // 7 days
			c.History = c.History[len(c.History)-168:]
		}
	}
	c.CurrentQuestion = q
}

// Subscribe registers a Web Push subscription; idempotent on endpoint.
func (c *Castle) Subscribe(s PushSubscription) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Subscriptions[s.Endpoint] = s
}

// Unsubscribe removes a subscription by endpoint.
func (c *Castle) Unsubscribe(endpoint string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Subscriptions, endpoint)
}

// Subs returns a snapshot of subscriptions (safe to range over without a lock).
func (c *Castle) Subs() []PushSubscription {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]PushSubscription, 0, len(c.Subscriptions))
	for _, s := range c.Subscriptions {
		out = append(out, s)
	}
	return out
}

// Snapshot returns a copy of the current question and recent history.
func (c *Castle) Snapshot() (Question, []Question) {
	c.mu.Lock()
	defer c.mu.Unlock()
	hist := make([]Question, len(c.History))
	copy(hist, c.History)
	return c.CurrentQuestion, hist
}
