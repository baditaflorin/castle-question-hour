package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/baditaflorin/castle-question-hour/backend/internal/castle"
	"github.com/baditaflorin/castle-question-hour/backend/internal/schedule"
)

type apiError struct {
	Error errBody `json:"error"`
}
type errBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, apiError{Error: errBody{Code: code, Message: msg}})
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readyz(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Cheap version: report ready if we have a bank loaded.
		// (Probing Ollama/Piper here would couple readiness to peers being up.)
		out := map[string]any{
			"status":    "ok",
			"bank_size": d.Bank.Size(),
			"version":   d.Cfg.AppVersion,
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func vapidPublicKeyHandler(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"public_key": d.Cfg.VAPIDPublicKey})
	}
}

func getNow(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		c, err := d.Registry.GetOrCreate(code)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_code", err.Error())
			return
		}
		cur, _ := c.Snapshot()
		if cur.BucketID == "" {
			// First request for this castle — backfill with the current hour's question.
			cur = castle.Question{
				BucketID:  schedule.BucketID(time.Now().UTC()),
				Text:      d.Bank.NextFor(code),
				EmittedAt: time.Now().UTC(),
			}
			c.SetQuestion(cur)
			d.Metrics.QuestionsEmitted.Inc()
		}
		writeJSON(w, http.StatusOK, cur)
	}
}

func getHistory(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		c, err := d.Registry.Get(code)
		if err != nil {
			if errors.Is(err, castle.ErrCastleNotFound) {
				writeJSON(w, http.StatusOK, []castle.Question{})
				return
			}
			writeErr(w, http.StatusBadRequest, "invalid_code", err.Error())
			return
		}
		n := 24
		if q := r.URL.Query().Get("n"); q != "" {
			if v, err := strconv.Atoi(q); err == nil && v > 0 && v <= 168 {
				n = v
			}
		}
		_, hist := c.Snapshot()
		if len(hist) > n {
			hist = hist[len(hist)-n:]
		}
		writeJSON(w, http.StatusOK, hist)
	}
}

type subscribeReq struct {
	Endpoint string `json:"endpoint"`
	P256DH   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

func postSubscribe(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		c, err := d.Registry.GetOrCreate(code)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_code", err.Error())
			return
		}
		var body subscribeReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "bad_json", "invalid payload")
			return
		}
		if body.Endpoint == "" || body.P256DH == "" || body.Auth == "" {
			writeErr(w, http.StatusBadRequest, "incomplete_subscription", "endpoint, p256dh and auth are required")
			return
		}
		c.Subscribe(castle.PushSubscription{Endpoint: body.Endpoint, P256DH: body.P256DH, Auth: body.Auth})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "subs": len(c.Subs())})
	}
}

func deleteSubscribe(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		c, err := d.Registry.Get(code)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
			return
		}
		endpoint := r.URL.Query().Get("endpoint")
		if endpoint == "" {
			writeErr(w, http.StatusBadRequest, "missing_endpoint", "endpoint query param required")
			return
		}
		c.Unsubscribe(endpoint)
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

type summarizeReq struct {
	BucketID  string   `json:"bucket_id"`
	Question  string   `json:"question"`
	Answers   []string `json:"answers"`
	WithAudio bool     `json:"with_audio"`
}

type summarizeResp struct {
	BucketID    string `json:"bucket_id"`
	Question    string `json:"question"`
	AnswerCount int    `json:"answer_count"`
	Themes      any    `json:"themes"`
	AudioWAV    []byte `json:"audio_wav,omitempty"`
}

func postSummarize(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		code := chi.URLParam(r, "code")
		if _, err := d.Registry.GetOrCreate(code); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_code", err.Error())
			return
		}
		var body summarizeReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "bad_json", "invalid payload")
			return
		}
		if body.Question == "" {
			writeErr(w, http.StatusBadRequest, "missing_question", "question is required")
			return
		}

		result, err := d.Summarize.Summarize(r.Context(), body.Question, body.Answers)
		if err != nil {
			d.Log.Warn("summarize failed", "err", err, "castle", code)
			writeErr(w, http.StatusBadGateway, "llm_unavailable", err.Error())
			return
		}
		resp := summarizeResp{
			BucketID:    body.BucketID,
			Question:    body.Question,
			AnswerCount: result.AnswerCount,
			Themes:      result.Themes,
		}
		if body.WithAudio && d.TTS != nil && len(result.Themes) > 0 {
			text := composeSpoken(body.Question, result.Themes)
			wav, err := d.TTS.Speak(r.Context(), text)
			if err != nil {
				d.Log.Warn("tts failed", "err", err)
			} else {
				resp.AudioWAV = wav
			}
		}
		d.Metrics.SummaryLatency.Observe(time.Since(start).Seconds())
		writeJSON(w, http.StatusOK, resp)
	}
}

func signalWS(d *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		d.Hub.ServeHTTP(w, r, code)
	}
}
