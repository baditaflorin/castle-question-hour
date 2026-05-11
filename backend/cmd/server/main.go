// Command server is the castle-question-hour backend entrypoint.
//
// It wires every collaborator explicitly: config → logger → registry → bank →
// hub → push → scheduler → summarize → tts → router → HTTP server, and
// shuts down gracefully on SIGTERM/SIGINT.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/baditaflorin/castle-question-hour/backend/internal/castle"
	"github.com/baditaflorin/castle-question-hour/backend/internal/config"
	"github.com/baditaflorin/castle-question-hour/backend/internal/httpapi"
	"github.com/baditaflorin/castle-question-hour/backend/internal/logging"
	"github.com/baditaflorin/castle-question-hour/backend/internal/metrics"
	"github.com/baditaflorin/castle-question-hour/backend/internal/push"
	"github.com/baditaflorin/castle-question-hour/backend/internal/schedule"
	"github.com/baditaflorin/castle-question-hour/backend/internal/signaling"
	"github.com/baditaflorin/castle-question-hour/backend/internal/summarize"
	"github.com/baditaflorin/castle-question-hour/backend/internal/tts"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		// pre-logger error — fall back to stderr
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}
	cfg.AppVersion = version

	log := logging.New(cfg.LogLevel).With(slog.String("version", version))
	log.Info("starting", "addr", cfg.ServerAddr)

	if err := run(cfg, log); err != nil {
		log.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, log *slog.Logger) error {
	m := metrics.New()
	reg := castle.NewRegistry()

	bank, err := schedule.LoadBank(cfg.QuestionsFile)
	if err != nil {
		log.Warn("loading question bank failed; falling back to embedded", "err", err)
		bank, err = schedule.NewBank(defaultQuestions())
		if err != nil {
			return err
		}
	}

	hub := signaling.NewHub(reg, log, m)
	sender := push.New(cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject, log, m)
	sched := schedule.New(cfg.HourlyCron, reg, bank, sender, log, m)

	llm := summarize.New(cfg.OllamaURL, cfg.OllamaModel, cfg.OllamaTimeout)

	var speaker tts.Speaker
	if cfg.PiperHTTPURL != "" {
		speaker = &tts.HTTPSpeaker{
			URL:     cfg.PiperHTTPURL,
			Voice:   cfg.PiperVoice,
			HTTP:    &http.Client{Timeout: 30 * time.Second},
			Log:     log.With("component", "piper"),
			Metrics: m,
		}
	} else {
		speaker = &tts.SubprocessSpeaker{
			Bin:     cfg.PiperBin,
			Voice:   cfg.PiperVoice,
			Log:     log.With("component", "piper"),
			Metrics: m,
		}
	}

	router := httpapi.New(&httpapi.Deps{
		Cfg:       cfg,
		Log:       log,
		Registry:  reg,
		Bank:      bank,
		Hub:       hub,
		Summarize: llm,
		TTS:       speaker,
		Metrics:   m,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if cfg.HourlyEnabled {
		if err := sched.Start(ctx); err != nil {
			return err
		}
	}

	srv := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      router,
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel2()
	return srv.Shutdown(shutdownCtx)
}

// defaultQuestions is the in-binary fallback bank if QUESTIONS_FILE is missing
// or unreadable. Keep it small but real.
func defaultQuestions() []string {
	return []string{
		"What scared you most this year?",
		"What did you find that you didn't expect to find?",
		"Who do you owe a quiet thank-you?",
		"What did you make this year that surprised you?",
		"What did you walk away from, and was it worth it?",
		"What part of yourself is rehearsing for next year?",
		"What did you let go of and not miss?",
		"What's the smallest beautiful thing you saw this week?",
	}
}
