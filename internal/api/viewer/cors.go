package viewer

import (
	"net/http"
	"net/url"
)

// WithLocalCORS is a middleware that adds CORS headers only for requests
// originating from localhost or 127.0.0.1 (any port). It is applied only to
// viewer routes — not globally (ADR-E27-007).
//
// Behavior:
//   - No Origin header → call next normally (no CORS headers added).
//   - Origin from localhost or 127.0.0.1 → echo Origin as
//     Access-Control-Allow-Origin + set Vary: Origin.
//   - OPTIONS request from local origin → respond 204 with preflight headers,
//     do NOT call next.
//   - OPTIONS request from any other origin → respond 204, do NOT call next
//     (no CORS headers set; browser blocks the request).
//   - Non-OPTIONS, non-local origin → call next without CORS headers (browser
//     blocks the request due to missing Access-Control-Allow-Origin).
func WithLocalCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// BUG-002 fix: OPTIONS must short-circuit with 204 regardless of
		// Origin, so check method before the empty-origin early return.
		origin := r.Header.Get("Origin")

		if r.Method == http.MethodOptions {
			if isLocalOrigin(origin) {
				// Handle preflight request from a local origin.
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}
			// For all OPTIONS requests (local or not, with or without Origin)
			// short-circuit with 204. Do NOT call next.
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if origin == "" {
			// No Origin header — not a CORS request; pass through.
			next.ServeHTTP(w, r)
			return
		}

		if isLocalOrigin(origin) {
			// Echo the origin so that credentials work correctly with
			// a wildcard-equivalent response.
			// BUG-001 fix: set Allow-Methods and Allow-Headers for ALL
			// local-origin requests, not just OPTIONS preflights.
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		next.ServeHTTP(w, r)
	})
}

// isLocalOrigin reports whether the given origin URL has a host of
// "localhost" or "127.0.0.1". The port is ignored per ADR-E27-007.
func isLocalOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1"
}
