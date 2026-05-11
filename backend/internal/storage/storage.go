// Package storage offers an opt-in SQLite persistence layer for Web Push
// subscriptions so that subscriptions survive a backend restart.
//
// In-memory is the default; SQLite is wired only when SQLITE_PATH is set.
package storage

import (
	"context"
	"errors"

	"github.com/baditaflorin/castle-question-hour/backend/internal/castle"
)

// SubscriptionStore is the persistence contract. The in-memory implementation
// returns ErrNotPersistent for callers that explicitly want disk; the SQLite
// implementation persists.
type SubscriptionStore interface {
	Save(ctx context.Context, castleCode string, sub castle.PushSubscription) error
	Delete(ctx context.Context, castleCode, endpoint string) error
	LoadAll(ctx context.Context, sink func(castleCode string, sub castle.PushSubscription)) error
}

var ErrNotPersistent = errors.New("subscription store is in-memory only")

// MemoryStore satisfies SubscriptionStore without touching disk. It's a no-op
// wrapper because Castle already holds subscriptions in memory.
type MemoryStore struct{}

func (MemoryStore) Save(context.Context, string, castle.PushSubscription) error { return nil }
func (MemoryStore) Delete(context.Context, string, string) error                { return nil }
func (MemoryStore) LoadAll(context.Context, func(string, castle.PushSubscription)) error {
	return nil
}
