// Package tts wraps Piper as either a subprocess or an HTTP service and
// returns WAV bytes to the caller.
package tts

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"

	"github.com/baditaflorin/castle-question-hour/backend/internal/metrics"
)

var ErrPiperUnavailable = errors.New("piper unavailable")

// Speaker is the production interface; one of the two implementations
// (subprocess or HTTP) is wired by main based on env.
type Speaker interface {
	Speak(ctx context.Context, text string) ([]byte, error)
}

// SubprocessSpeaker invokes the piper binary.
type SubprocessSpeaker struct {
	Bin     string
	Voice   string
	Log     *slog.Logger
	Metrics *metrics.Registry
}

// HTTPSpeaker calls a piper-http wrapper.
type HTTPSpeaker struct {
	URL     string
	Voice   string
	HTTP    *http.Client
	Log     *slog.Logger
	Metrics *metrics.Registry
}

// Speak runs piper and returns the WAV bytes on stdout.
func (s *SubprocessSpeaker) Speak(ctx context.Context, text string) ([]byte, error) {
	if s.Bin == "" || s.Voice == "" {
		return nil, ErrPiperUnavailable
	}
	cmd := exec.CommandContext(ctx, s.Bin, "--model", s.Voice, "--output_raw")
	cmd.Stdin = strings.NewReader(text)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		s.Metrics.TTSSubprocessFails.Inc()
		s.Log.Warn("piper subprocess failed", "err", err, "stderr_len", stderr.Len())
		return nil, fmt.Errorf("piper: %w", err)
	}
	if stdout.Len() == 0 {
		s.Metrics.TTSSubprocessFails.Inc()
		return nil, fmt.Errorf("piper produced empty output (stderr=%s)", strings.TrimSpace(stderr.String()))
	}
	return wrapWAV(stdout.Bytes()), nil
}

// Speak POSTs JSON to a piper-http endpoint and returns the WAV bytes.
func (s *HTTPSpeaker) Speak(ctx context.Context, text string) ([]byte, error) {
	if s.URL == "" {
		return nil, ErrPiperUnavailable
	}
	body := strings.NewReader(fmt.Sprintf(`{"text":%q,"voice":%q}`, text, s.Voice))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL+"/api/tts", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("piper http: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("piper http %d: %s", resp.StatusCode, raw)
	}
	wav, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return wav, nil
}

// wrapWAV synthesizes a minimal WAV header around raw PCM s16le 22050Hz output.
// Piper --output_raw emits headerless PCM; we wrap so browsers can play it.
func wrapWAV(pcm []byte) []byte {
	const (
		sampleRate = 22050
		channels   = 1
		bitsPer    = 16
	)
	byteRate := sampleRate * channels * bitsPer / 8
	blockAlign := channels * bitsPer / 8
	dataSize := len(pcm)
	fileSize := 36 + dataSize

	buf := make([]byte, 44+dataSize)
	copy(buf[0:4], "RIFF")
	putUint32LE(buf[4:8], uint32(fileSize))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	putUint32LE(buf[16:20], 16)
	putUint16LE(buf[20:22], 1)
	putUint16LE(buf[22:24], uint16(channels))
	putUint32LE(buf[24:28], sampleRate)
	putUint32LE(buf[28:32], uint32(byteRate))
	putUint16LE(buf[32:34], uint16(blockAlign))
	putUint16LE(buf[34:36], uint16(bitsPer))
	copy(buf[36:40], "data")
	putUint32LE(buf[40:44], uint32(dataSize))
	copy(buf[44:], pcm)
	return buf
}

func putUint16LE(b []byte, v uint16) { b[0] = byte(v); b[1] = byte(v >> 8) }
func putUint32LE(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}
