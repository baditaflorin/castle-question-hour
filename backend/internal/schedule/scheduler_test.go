package schedule

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/baditaflorin/castle-question-hour/backend/internal/castle"
	"github.com/baditaflorin/castle-question-hour/backend/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPusher struct {
	mu   sync.Mutex
	hits []string
}

func (s *stubPusher) NotifyCastle(_ context.Context, c *castle.Castle, _ castle.Question) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hits = append(s.hits, c.Code)
	return nil
}

func TestTick_FansOutToAllCastles(t *testing.T) {
	reg := castle.NewRegistry()
	_, err := reg.GetOrCreate("alpha")
	require.NoError(t, err)
	_, err = reg.GetOrCreate("beta")
	require.NoError(t, err)
	bank, err := NewBank([]string{"q1", "q2", "q3"})
	require.NoError(t, err)
	push := &stubPusher{}
	s := New("* * * * *", reg, bank, push, slog.New(slog.NewTextHandler(io.Discard, nil)), metrics.New())

	s.Tick(context.Background())

	push.mu.Lock()
	defer push.mu.Unlock()
	assert.ElementsMatch(t, []string{"alpha", "beta"}, push.hits)

	c, _ := reg.Get("alpha")
	q, _ := c.Snapshot()
	assert.NotEmpty(t, q.Text)
	assert.NotEmpty(t, q.BucketID)
}

func TestBucketID_TruncatesToHour(t *testing.T) {
	tt := time.Date(2026, 5, 11, 14, 37, 42, 0, time.UTC)
	assert.Equal(t, "2026-05-11T14:00:00Z", BucketID(tt))
}
