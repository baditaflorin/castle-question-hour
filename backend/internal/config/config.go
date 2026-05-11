// Package config loads runtime configuration from environment variables.
//
// Env binding is intentionally explicit (no struct tags interpreted by viper)
// so the call sites read like a manifest of every knob we expose.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppVersion string

	ServerAddr         string
	ServerReadTimeout  time.Duration
	ServerWriteTimeout time.Duration

	PagesOrigin string

	SQLitePath string

	OllamaURL     string
	OllamaModel   string
	OllamaTimeout time.Duration

	PiperBin     string
	PiperVoice   string
	PiperHTTPURL string

	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string

	HourlyEnabled bool
	HourlyCron    string
	QuestionsFile string

	StewardToken string // when non-empty, /summarize requires X-Steward-Token to match

	LogLevel       string
	MetricsEnabled bool
}

// Load reads env, applies defaults, and returns a validated Config.
func Load() (*Config, error) {
	c := &Config{
		AppVersion:         env("APP_VERSION", "dev"),
		ServerAddr:         env("SERVER_ADDR", ":8080"),
		ServerReadTimeout:  envDuration("SERVER_READ_TIMEOUT", 10*time.Second),
		ServerWriteTimeout: envDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		PagesOrigin:        env("PAGES_ORIGIN", "https://baditaflorin.github.io"),
		SQLitePath:         env("SQLITE_PATH", ""),
		OllamaURL:          env("OLLAMA_URL", "http://ollama:11434"),
		OllamaModel:        env("OLLAMA_MODEL", "llama3.2:3b"),
		OllamaTimeout:      envDuration("OLLAMA_TIMEOUT", 60*time.Second),
		PiperBin:           env("PIPER_BIN", "/usr/local/bin/piper"),
		PiperVoice:         env("PIPER_VOICE", "/voices/en_US-lessac-medium.onnx"),
		PiperHTTPURL:       env("PIPER_HTTP_URL", ""),
		VAPIDPublicKey:     env("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey:    env("VAPID_PRIVATE_KEY", ""),
		VAPIDSubject:       env("VAPID_SUBJECT", "mailto:steward@example.org"),
		HourlyEnabled:      envBool("HOURLY_ENABLED", true),
		HourlyCron:         env("HOURLY_CRON", "0 * * * *"),
		QuestionsFile:      env("QUESTIONS_FILE", "/etc/cqh/questions.json"),
		StewardToken:       env("STEWARD_TOKEN", ""),
		LogLevel:           env("LOG_LEVEL", "info"),
		MetricsEnabled:     envBool("METRICS_ENABLED", true),
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) validate() error {
	if !strings.HasPrefix(c.ServerAddr, ":") && !strings.Contains(c.ServerAddr, ":") {
		return fmt.Errorf("SERVER_ADDR must include a port, got %q", c.ServerAddr)
	}
	if c.HourlyEnabled && c.QuestionsFile == "" {
		return fmt.Errorf("HOURLY_ENABLED=true but QUESTIONS_FILE is empty")
	}
	return nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envDuration(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
