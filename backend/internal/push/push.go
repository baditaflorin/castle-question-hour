// Package push wraps the Web Push protocol so the scheduler can fan-out
// hourly notifications without knowing VAPID internals.
package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/baditaflorin/castle-question-hour/backend/internal/castle"
	"github.com/baditaflorin/castle-question-hour/backend/internal/metrics"
)

// Sender pushes notifications to a castle's registered phones.
type Sender struct {
	publicKey  string
	privateKey string
	subject    string
	log        *slog.Logger
	metrics    *metrics.Registry
}

func New(publicKey, privateKey, subject string, log *slog.Logger, m *metrics.Registry) *Sender {
	return &Sender{
		publicKey:  publicKey,
		privateKey: privateKey,
		subject:    subject,
		log:        log,
		metrics:    m,
	}
}

// Payload is the JSON delivered to the service worker.
type Payload struct {
	Kind     string `json:"kind"`     // "question" | "summary-ready"
	BucketID string `json:"bucket_id,omitempty"`
	Question string `json:"question,omitempty"`
	Castle   string `json:"castle"`
}

// NotifyCastle pushes a "question" payload to every subscription on the castle.
// Errors per-subscription are logged, not propagated — one phone offline must
// not stop the others.
func (s *Sender) NotifyCastle(ctx context.Context, c *castle.Castle, q castle.Question) error {
	if s.publicKey == "" || s.privateKey == "" {
		return errors.New("vapid keys not configured")
	}
	body := Payload{
		Kind:     "question",
		BucketID: q.BucketID,
		Question: q.Text,
		Castle:   c.Code,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal push payload: %w", err)
	}
	for _, sub := range c.Subs() {
		s.sendOne(ctx, sub, raw)
	}
	return nil
}

func (s *Sender) sendOne(ctx context.Context, sub castle.PushSubscription, payload []byte) {
	wp := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.P256DH, Auth: sub.Auth},
	}
	resp, err := webpush.SendNotificationWithContext(ctx, payload, wp, &webpush.Options{
		VAPIDPublicKey:  s.publicKey,
		VAPIDPrivateKey: s.privateKey,
		Subscriber:      s.subject,
		TTL:             3600,
	})
	if err != nil {
		s.metrics.PushSent.WithLabelValues("error").Inc()
		s.log.Warn("push send failed", "endpoint_len", len(sub.Endpoint), "err", err)
		return
	}
	defer resp.Body.Close() //nolint:errcheck // we drain via Close, body content is not used
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		s.metrics.PushSent.WithLabelValues("ok").Inc()
	case resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound:
		s.metrics.PushSent.WithLabelValues("expired").Inc()
		s.log.Info("push subscription expired", "status", resp.StatusCode)
	default:
		s.metrics.PushSent.WithLabelValues("error").Inc()
		s.log.Warn("push non-2xx", "status", resp.StatusCode)
	}
}
