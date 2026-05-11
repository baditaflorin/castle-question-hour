package httpapi

import (
	"crypto/subtle"
	"net/http"
)

// stewardGate denies the request unless the X-Steward-Token header matches.
// When token is empty, the gate is a no-op — open castle.
//
// The comparison is constant-time so a slow caller can't measure prefix matches.
func stewardGate(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			given := r.Header.Get("X-Steward-Token")
			if subtle.ConstantTimeCompare([]byte(given), []byte(token)) != 1 {
				writeErr(w, http.StatusUnauthorized, "steward_required",
					"this castle requires the steward's token to summarize")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
