package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStewardGate_NoTokenLetsThrough(t *testing.T) {
	called := false
	h := stewardGate("")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStewardGate_RejectsMissingToken(t *testing.T) {
	called := false
	h := stewardGate("s3cr3t")(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	assert.False(t, called)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "steward_required")
}

func TestStewardGate_AcceptsMatchingToken(t *testing.T) {
	called := false
	h := stewardGate("s3cr3t")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Steward-Token", "s3cr3t")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestStewardGate_RejectsMismatch(t *testing.T) {
	called := false
	h := stewardGate("s3cr3t")(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Steward-Token", "wrong")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.False(t, called)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
