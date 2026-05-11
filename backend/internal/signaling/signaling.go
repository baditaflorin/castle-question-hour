// Package signaling is a thin WebSocket relay used to bootstrap a Yjs
// WebRTC mesh. The backend never sees Y.Doc contents — only offer/answer/ICE
// envelopes flowing between peers.
package signaling

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"github.com/coder/websocket"

	"github.com/baditaflorin/castle-question-hour/backend/internal/castle"
	"github.com/baditaflorin/castle-question-hour/backend/internal/metrics"
)

// Hub keeps per-castle peer sets and broadcasts signaling envelopes among them.
type Hub struct {
	mu       sync.Mutex
	rooms    map[string]map[*peer]struct{}
	registry *castle.Registry
	log      *slog.Logger
	metrics  *metrics.Registry
}

type peer struct {
	id   string
	conn *websocket.Conn
}

// Envelope is the JSON shape we accept from clients. Type values mirror
// y-webrtc's signaling protocol: "subscribe", "unsubscribe", "publish", "ping".
type Envelope struct {
	Type    string          `json:"type"`
	Topics  []string        `json:"topics,omitempty"`
	Topic   string          `json:"topic,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func NewHub(reg *castle.Registry, log *slog.Logger, m *metrics.Registry) *Hub {
	return &Hub{
		rooms:    make(map[string]map[*peer]struct{}),
		registry: reg,
		log:      log,
		metrics:  m,
	}
}

// ServeHTTP upgrades to WebSocket and starts the per-peer read loop.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request, castleCode string) {
	if err := castle.ValidateCode(castleCode); err != nil {
		http.Error(w, "bad castle code", http.StatusBadRequest)
		return
	}
	if _, err := h.registry.GetOrCreate(castleCode); err != nil {
		http.Error(w, "bad castle code", http.StatusBadRequest)
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Origin check is done at nginx via CORS; in dev we let the proxy do it.
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.log.Warn("ws accept failed", "err", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	p := &peer{id: r.Header.Get("X-Trace-Id"), conn: conn}
	h.join(castleCode, p)
	h.metrics.SignalPeers.Inc()
	defer func() {
		h.leave(castleCode, p)
		h.metrics.SignalPeers.Dec()
	}()

	ctx := r.Context()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				h.log.Debug("ws read end", "err", err)
			}
			return
		}
		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			h.log.Debug("ws bad envelope", "err", err)
			continue
		}
		h.handle(ctx, castleCode, p, env, data)
	}
}

func (h *Hub) join(code string, p *peer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[code]; !ok {
		h.rooms[code] = make(map[*peer]struct{})
		h.metrics.CastlesActive.Set(float64(len(h.rooms)))
	}
	h.rooms[code][p] = struct{}{}
}

func (h *Hub) leave(code string, p *peer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.rooms[code]; ok {
		delete(room, p)
		if len(room) == 0 {
			delete(h.rooms, code)
		}
	}
	h.metrics.CastlesActive.Set(float64(len(h.rooms)))
}

// handle relays publish-typed envelopes to every other peer in the same castle.
// subscribe/unsubscribe/ping are local control messages and don't fan out.
func (h *Hub) handle(ctx context.Context, code string, from *peer, env Envelope, raw []byte) {
	switch env.Type {
	case "publish":
		h.broadcast(ctx, code, from, raw)
	case "subscribe", "unsubscribe", "ping":
		// y-webrtc compatibility: respond to ping with pong; subscribe/unsubscribe are
		// effectively no-ops at the hub level because every peer in the room sees every
		// publish for that room.
		if env.Type == "ping" {
			_ = from.conn.Write(ctx, websocket.MessageText, []byte(`{"type":"pong"}`))
		}
	}
}

func (h *Hub) broadcast(ctx context.Context, code string, from *peer, raw []byte) {
	h.mu.Lock()
	peers := make([]*peer, 0, len(h.rooms[code]))
	for p := range h.rooms[code] {
		if p != from {
			peers = append(peers, p)
		}
	}
	h.mu.Unlock()

	for _, p := range peers {
		// Don't let one slow peer hold up the loop; write timeout handled by ServeHTTP ctx.
		if err := p.conn.Write(ctx, websocket.MessageText, raw); err != nil {
			h.log.Debug("ws write failed", "err", err)
		}
	}
}
