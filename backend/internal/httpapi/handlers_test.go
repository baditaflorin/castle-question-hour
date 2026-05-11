package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baditaflorin/castle-question-hour/backend/internal/castle"
	"github.com/baditaflorin/castle-question-hour/backend/internal/config"
	"github.com/baditaflorin/castle-question-hour/backend/internal/metrics"
	"github.com/baditaflorin/castle-question-hour/backend/internal/schedule"
	"github.com/baditaflorin/castle-question-hour/backend/internal/signaling"
	"github.com/baditaflorin/castle-question-hour/backend/internal/summarize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nopSpeaker struct{}

func (nopSpeaker) Speak(context.Context, string) ([]byte, error) { return []byte("RIFF...."), nil }

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	bank, err := schedule.NewBank([]string{"What scared you?"})
	require.NoError(t, err)
	reg := castle.NewRegistry()
	m := metrics.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{PagesOrigin: "https://baditaflorin.github.io", MetricsEnabled: true, VAPIDPublicKey: "PUB", AppVersion: "test"}
	return New(&Deps{
		Cfg:       cfg,
		Log:       log,
		Registry:  reg,
		Bank:      bank,
		Hub:       signaling.NewHub(reg, log, m),
		Summarize: summarize.New("http://no-server", "model", 0),
		TTS:       nopSpeaker{},
		Metrics:   m,
	})
}

func TestHealthz(t *testing.T) {
	router := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetNow_BackfillsFirstHour(t *testing.T) {
	router := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/castle/grey/now", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var q castle.Question
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&q))
	assert.NotEmpty(t, q.BucketID)
	assert.NotEmpty(t, q.Text)
}

func TestGetNow_RejectsInvalidCode(t *testing.T) {
	router := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/castle/UPPER/now", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestVAPIDPublicKey(t *testing.T) {
	router := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vapid-public-key", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "PUB", body["public_key"])
}

func TestSubscribeFlow(t *testing.T) {
	router := newTestRouter(t)

	// missing fields → 400
	req := httptest.NewRequest(http.MethodPost, "/api/v1/castle/grey/subscribe", io.NopCloser(jsonBody(`{"endpoint":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// complete → 200
	req = httptest.NewRequest(http.MethodPost, "/api/v1/castle/grey/subscribe", io.NopCloser(jsonBody(`{"endpoint":"https://push.example/x","p256dh":"abc","auth":"def"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// delete → 200
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/castle/grey/subscribe?endpoint=https://push.example/x", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func jsonBody(s string) *jsonReader { return &jsonReader{s: s} }

type jsonReader struct {
	s string
	o int
}

func (j *jsonReader) Read(p []byte) (int, error) {
	if j.o >= len(j.s) {
		return 0, io.EOF
	}
	n := copy(p, j.s[j.o:])
	j.o += n
	return n, nil
}
