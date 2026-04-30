package middleware

import (
	"net/http"
	"strings"
)

// AuthSwap removes the GitHub Authorization header and injects GitLab PRIVATE-TOKEN.
func AuthSwap(pat string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "token ") || strings.HasPrefix(auth, "Bearer ") {
				r.Header.Del("Authorization")
			}
			r.Header.Set("PRIVATE-TOKEN", pat)
			next.ServeHTTP(w, r)
		})
	}
}
