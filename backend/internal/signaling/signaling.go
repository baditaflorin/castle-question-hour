// Package signaling is a y-webrtc-compatible WebSocket signaling server used
// to bootstrap a Yjs WebRTC mesh. The protocol is topic-based:
//
//	{"type":"subscribe","topics":["cqh:castle-grey", ...]}
//	{"type":"unsubscribe","topics":[...]}
//	{"type":"publish","topic":"cqh:castle-grey","data":...}  // echoed to ALL subscribers
//	{"type":"ping"}                                          // server responds with pong
//
// The backend never decodes `data` — it only routes envelopes between peers.
// Castle code URL bucketing is retained as a sanity boundary, but route
// matching is by topic name to match the y-webrtc reference server.
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

// Hub routes y-webrtc signaling envelopes between peers, indexed by topic.
type Hub struct {
	mu         sync.Mutex
	topicConns map[string]map[*peer]struct{} // topic → connected peers
	connTopics map[*peer]map[string]struct{} // reverse: peer → subscribed topics
	totalPeers int
	registry   *castle.Registry
	log        *slog.Logger
	metrics    *metrics.Registry
}

type peer struct {
	id   string
	conn *websocket.Conn
}

// Envelope is the JSON shape we accept from and emit to clients.
type Envelope struct {
	Type    string          `json:"type"`
	Topics  []string        `json:"topics,omitempty"`
	Topic   string          `json:"topic,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Clients int             `json:"clients,omitempty"`
}

func NewHub(reg *castle.Registry, log *slog.Logger, m *metrics.Registry) *Hub {
	return &Hub{
		topicConns: make(map[string]map[*peer]struct{}),
		connTopics: make(map[*peer]map[string]struct{}),
		registry:   reg,
		log:        log,
		metrics:    m,
	}
}

// ServeHTTP upgrades to WebSocket and starts the per-peer read loop. The
// castleCode is validated and registered for visibility, but routing happens
// by topic, not by castle URL.
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
		// Origin check is enforced at nginx via CORS; the hub trusts that hop.
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.log.Warn("ws accept failed", "err", err)
		return
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	p := &peer{id: r.Header.Get("X-Trace-Id"), conn: conn}
	h.register(p)
	h.metrics.SignalPeers.Inc()
	defer func() {
		h.deregister(p)
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
		h.handle(ctx, p, env)
	}
}

func (h *Hub) register(p *peer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connTopics[p] = make(map[string]struct{})
	h.totalPeers++
	h.metrics.CastlesActive.Set(float64(len(h.topicConns)))
}

func (h *Hub) deregister(p *peer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for t := range h.connTopics[p] {
		delete(h.topicConns[t], p)
		if len(h.topicConns[t]) == 0 {
			delete(h.topicConns, t)
		}
	}
	delete(h.connTopics, p)
	h.totalPeers--
	h.metrics.CastlesActive.Set(float64(len(h.topicConns)))
}

func (h *Hub) handle(ctx context.Context, from *peer, env Envelope) {
	switch env.Type {
	case "subscribe":
		h.subscribe(from, env.Topics)
	case "unsubscribe":
		h.unsubscribe(from, env.Topics)
	case "publish":
		h.publish(ctx, from, env)
	case "ping":
		_ = from.conn.Write(ctx, websocket.MessageText, []byte(`{"type":"pong"}`))
	}
}

func (h *Hub) subscribe(p *peer, topics []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, t := range topics {
		if t == "" {
			continue
		}
		set, ok := h.topicConns[t]
		if !ok {
			set = make(map[*peer]struct{})
			h.topicConns[t] = set
		}
		set[p] = struct{}{}
		h.connTopics[p][t] = struct{}{}
	}
}

func (h *Hub) unsubscribe(p *peer, topics []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, t := range topics {
		if set, ok := h.topicConns[t]; ok {
			delete(set, p)
			if len(set) == 0 {
				delete(h.topicConns, t)
			}
		}
		delete(h.connTopics[p], t)
	}
}

// publish fans out env to every peer subscribed to env.Topic (the y-webrtc
// reference server echoes back to the sender too, including subscriber count
// so peers can use the count to decide when to initiate a peer connection).
func (h *Hub) publish(ctx context.Context, _ *peer, env Envelope) {
	if env.Topic == "" {
		return
	}
	h.mu.Lock()
	subs := h.topicConns[env.Topic]
	receivers := make([]*peer, 0, len(subs))
	for r := range subs {
		receivers = append(receivers, r)
	}
	h.mu.Unlock()

	env.Clients = len(receivers)
	raw, err := json.Marshal(env)
	if err != nil {
		h.log.Warn("publish marshal failed", "err", err)
		return
	}
	for _, r := range receivers {
		if err := r.conn.Write(ctx, websocket.MessageText, raw); err != nil {
			h.log.Debug("ws write failed", "err", err)
		}
	}
}
