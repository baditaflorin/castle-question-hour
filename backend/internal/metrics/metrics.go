// Package metrics defines Prometheus collectors used by the rest of the backend.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Registry struct {
	HTTPRequests        *prometheus.CounterVec
	HTTPDuration        *prometheus.HistogramVec
	CastlesActive       prometheus.Gauge
	SignalPeers         prometheus.Gauge
	PushSent            *prometheus.CounterVec
	SummaryLatency      prometheus.Histogram
	QuestionsEmitted    prometheus.Counter
	TTSSubprocessFails  prometheus.Counter

	reg *prometheus.Registry
}

func New() *Registry {
	reg := prometheus.NewRegistry()
	r := &Registry{
		reg: reg,
		HTTPRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "cqh_http_requests_total", Help: "HTTP requests."},
			[]string{"route", "method", "status"},
		),
		HTTPDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "cqh_http_request_duration_seconds", Help: "HTTP request duration."},
			[]string{"route"},
		),
		CastlesActive: prometheus.NewGauge(
			prometheus.GaugeOpts{Name: "cqh_castles_active", Help: "Currently joined castle sessions."},
		),
		SignalPeers: prometheus.NewGauge(
			prometheus.GaugeOpts{Name: "cqh_signal_peers_connected", Help: "Open signaling peers."},
		),
		PushSent: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "cqh_push_sent_total", Help: "Web Push attempts."},
			[]string{"outcome"},
		),
		SummaryLatency: prometheus.NewHistogram(
			prometheus.HistogramOpts{Name: "cqh_summary_latency_seconds", Help: "Summary pipeline latency."},
		),
		QuestionsEmitted: prometheus.NewCounter(
			prometheus.CounterOpts{Name: "cqh_questions_emitted_total", Help: "Hourly question fires."},
		),
		TTSSubprocessFails: prometheus.NewCounter(
			prometheus.CounterOpts{Name: "cqh_tts_subprocess_failures_total", Help: "Piper failures."},
		),
	}
	reg.MustRegister(
		r.HTTPRequests, r.HTTPDuration, r.CastlesActive, r.SignalPeers,
		r.PushSent, r.SummaryLatency, r.QuestionsEmitted, r.TTSSubprocessFails,
	)
	return r
}

// Handler exposes /metrics.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{Registry: r.reg})
}

// ObserveHTTP wraps a handler and records request count + duration.
func (r *Registry) ObserveHTTP(route string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, status: 200}
		h.ServeHTTP(sr, req)
		r.HTTPRequests.WithLabelValues(route, req.Method, strconv.Itoa(sr.status)).Inc()
		r.HTTPDuration.WithLabelValues(route).Observe(time.Since(start).Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}
