package schedule

import (
	"context"
	"log/slog"
	"time"

	"github.com/baditaflorin/castle-question-hour/backend/internal/castle"
	"github.com/baditaflorin/castle-question-hour/backend/internal/metrics"
	"github.com/robfig/cron/v3"
)

// Pusher abstracts the push fan-out so schedule doesn't depend on push internals.
type Pusher interface {
	NotifyCastle(ctx context.Context, c *castle.Castle, q castle.Question) error
}

// Scheduler emits a new question to every active castle on a cron expression.
type Scheduler struct {
	cron     *cron.Cron
	expr     string
	registry *castle.Registry
	bank     *Bank
	push     Pusher
	log      *slog.Logger
	metrics  *metrics.Registry
}

func New(expr string, reg *castle.Registry, bank *Bank, push Pusher, log *slog.Logger, m *metrics.Registry) *Scheduler {
	return &Scheduler{
		cron:     cron.New(),
		expr:     expr,
		registry: reg,
		bank:     bank,
		push:     push,
		log:      log,
		metrics:  m,
	}
}

// Start registers the cron entry and starts ticking. Stop with ctx cancellation.
func (s *Scheduler) Start(ctx context.Context) error {
	_, err := s.cron.AddFunc(s.expr, func() { s.Tick(ctx) })
	if err != nil {
		return err
	}
	s.cron.Start()
	go func() {
		<-ctx.Done()
		s.cron.Stop()
	}()
	s.log.Info("scheduler started", "expr", s.expr, "bank_size", s.bank.Size())
	return nil
}

// Tick fires a new question for every active castle. Exposed so tests can call it directly.
func (s *Scheduler) Tick(ctx context.Context) {
	bucket := BucketID(time.Now().UTC())
	castles := s.registry.All()
	s.log.Info("hourly tick", "castles", len(castles), "bucket", bucket)
	for _, c := range castles {
		q := castle.Question{
			BucketID:  bucket,
			Text:      s.bank.NextFor(c.Code),
			EmittedAt: time.Now().UTC(),
		}
		c.SetQuestion(q)
		s.metrics.QuestionsEmitted.Inc()
		if err := s.push.NotifyCastle(ctx, c, q); err != nil {
			s.log.Warn("push fanout failed", "castle", c.Code, "err", err)
		}
	}
}

// BucketID returns the canonical ID for an hour bucket: RFC3339 truncated to the hour, UTC.
func BucketID(t time.Time) string {
	return t.UTC().Truncate(time.Hour).Format(time.RFC3339)
}
