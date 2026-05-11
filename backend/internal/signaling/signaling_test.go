package signaling

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baditaflorin/castle-question-hour/backend/internal/castle"
	"github.com/baditaflorin/castle-question-hour/backend/internal/metrics"
)

func newTestServer(t *testing.T, hub *Hub, castleCode string) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub.ServeHTTP(w, r, castleCode)
	}))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	return srv, wsURL
}

func TestSignaling_PublishFansOutToAllSubscribers(t *testing.T) {
	reg := castle.NewRegistry()
	hub := NewHub(reg, slog.New(slog.NewTextHandler(io.Discard, nil)), metrics.New())
	srv, wsURL := newTestServer(t, hub, "grey")
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	a, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer a.CloseNow() //nolint:errcheck
	b, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer b.CloseNow() //nolint:errcheck

	// Both peers subscribe to the same topic.
	mustWrite(t, ctx, a, `{"type":"subscribe","topics":["t1"]}`)
	mustWrite(t, ctx, b, `{"type":"subscribe","topics":["t1"]}`)

	// Give the server a tick to register subscriptions.
	time.Sleep(50 * time.Millisecond)

	// Peer A publishes; both A and B should receive (echo-to-sender per y-webrtc).
	mustWrite(t, ctx, a, `{"type":"publish","topic":"t1","data":"hello"}`)

	envA := readEnv(t, ctx, a)
	envB := readEnv(t, ctx, b)
	assert.Equal(t, "publish", envA.Type)
	assert.Equal(t, "publish", envB.Type)
	assert.Equal(t, "t1", envB.Topic)
	assert.Equal(t, 2, envB.Clients, "publish must include subscriber count")
}

func TestSignaling_UnsubscribeStopsDelivery(t *testing.T) {
	reg := castle.NewRegistry()
	hub := NewHub(reg, slog.New(slog.NewTextHandler(io.Discard, nil)), metrics.New())
	srv, wsURL := newTestServer(t, hub, "grey")
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	a, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer a.CloseNow() //nolint:errcheck
	b, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer b.CloseNow() //nolint:errcheck

	mustWrite(t, ctx, a, `{"type":"subscribe","topics":["t1"]}`)
	mustWrite(t, ctx, b, `{"type":"subscribe","topics":["t1"]}`)
	time.Sleep(30 * time.Millisecond)
	mustWrite(t, ctx, b, `{"type":"unsubscribe","topics":["t1"]}`)
	time.Sleep(30 * time.Millisecond)
	mustWrite(t, ctx, a, `{"type":"publish","topic":"t1","data":"x"}`)

	envA := readEnv(t, ctx, a)
	assert.Equal(t, 1, envA.Clients, "B unsubscribed; only A remains")
}

func TestSignaling_PingPong(t *testing.T) {
	reg := castle.NewRegistry()
	hub := NewHub(reg, slog.New(slog.NewTextHandler(io.Discard, nil)), metrics.New())
	srv, wsURL := newTestServer(t, hub, "grey")
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer c.CloseNow() //nolint:errcheck

	mustWrite(t, ctx, c, `{"type":"ping"}`)
	env := readEnv(t, ctx, c)
	assert.Equal(t, "pong", env.Type)
}

func mustWrite(t *testing.T, ctx context.Context, c *websocket.Conn, msg string) {
	t.Helper()
	require.NoError(t, c.Write(ctx, websocket.MessageText, []byte(msg)))
}

func readEnv(t *testing.T, ctx context.Context, c *websocket.Conn) Envelope {
	t.Helper()
	_, data, err := c.Read(ctx)
	require.NoError(t, err)
	var env Envelope
	require.NoError(t, json.Unmarshal(data, &env))
	return env
}
