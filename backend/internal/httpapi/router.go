// Package httpapi wires the chi router, middlewares, and handlers.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/oklog/ulid/v2"

	"github.com/baditaflorin/castle-question-hour/backend/internal/castle"
	"github.com/baditaflorin/castle-question-hour/backend/internal/config"
	"github.com/baditaflorin/castle-question-hour/backend/internal/metrics"
	"github.com/baditaflorin/castle-question-hour/backend/internal/schedule"
	"github.com/baditaflorin/castle-question-hour/backend/internal/signaling"
	"github.com/baditaflorin/castle-question-hour/backend/internal/summarize"
	"github.com/baditaflorin/castle-question-hour/backend/internal/tts"
)

// Deps bundles every collaborator the router needs. Constructed in cmd/server/main.
type Deps struct {
	Cfg       *config.Config
	Log       *slog.Logger
	Registry  *castle.Registry
	Bank      *schedule.Bank
	Hub       *signaling.Hub
	Summarize *summarize.Client
	TTS       tts.Speaker
	Metrics   *metrics.Registry
}

// New builds the router.
func New(d *Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(traceIDMiddleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{d.Cfg.PagesOrigin, "http://localhost:5173", "http://localhost:4173"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "X-Trace-Id"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// liveness / readiness / metrics — no timeout middleware so probes are
	// always responsive even when handlers are saturated.
	r.Get("/healthz", healthz)
	r.Get("/readyz", readyz(d))
	if d.Cfg.MetricsEnabled {
		r.Method(http.MethodGet, "/metrics", d.Metrics.Handler())
	}

	// REST surface — uses chi's per-request timeout AND the custom status-logger.
	// Both wrap the ResponseWriter; we keep them off the /signal route so the
	// http.Hijacker interface the WebSocket upgrade needs stays reachable.
	r.Group(func(r chi.Router) {
		r.Use(loggerMiddleware(d.Log))
		r.Use(middleware.Timeout(60 * time.Second))
		r.Route("/api/v1", func(r chi.Router) {
			r.Get("/vapid-public-key", vapidPublicKeyHandler(d))
			r.Get("/server-info", serverInfoHandler(d))
			r.Route("/castle/{code}", func(r chi.Router) {
				r.Get("/now", getNow(d))
				r.Get("/history", getHistory(d))
				r.Post("/subscribe", postSubscribe(d))
				r.Delete("/subscribe", deleteSubscribe(d))
				r.With(stewardGate(d.Cfg.StewardToken)).Post("/summarize", postSummarize(d))
			})
		})
	})

	// WebSocket signaling — bypasses Timeout AND the status-recorder logger.
	r.Get("/api/v1/signal/{code}", signalWS(d))

	return r
}

func traceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Trace-Id")
		if id == "" {
			id = ulid.Make().String()
		}
		w.Header().Set("X-Trace-Id", id)
		ctx := context.WithValue(r.Context(), ctxKeyTraceID{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type ctxKeyTraceID struct{}

func loggerMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sr := &statusRecorder{ResponseWriter: w, status: 200}
			next.ServeHTTP(sr, r)
			trace, _ := r.Context().Value(ctxKeyTraceID{}).(string)
			log.Info("http",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sr.status),
				slog.Duration("took", time.Since(start)),
				slog.String("trace_id", trace),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}
