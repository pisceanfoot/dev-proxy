package cors

import (
	"net/http"
	"strings"

	"dev-proxy/internal/router"
)

// Middleware wraps an http.Handler and adds CORS headers based on route config.
func Middleware(next http.Handler, cfg *router.CORSConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg == nil || !cfg.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			handlePreflight(w, cfg, origin)
			return
		}

		// Set CORS headers on the response before forwarding
		addHeaders(w, cfg, origin)

		next.ServeHTTP(w, r)
	})
}

func handlePreflight(w http.ResponseWriter, cfg *router.CORSConfig, origin string) {
	if !isOriginAllowed(cfg.AllowOrigin, origin) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	addHeaders(w, cfg, origin)
	w.WriteHeader(http.StatusNoContent)
}

func addHeaders(w http.ResponseWriter, cfg *router.CORSConfig, origin string) {
	if cfg.AllowOrigin == "*" || isOriginAllowed(cfg.AllowOrigin, origin) {
		if cfg.AllowOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
	}

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

func isOriginAllowed(allowOrigin string, requestOrigin string) bool {
	if allowOrigin == "*" {
		return true
	}
	return strings.EqualFold(allowOrigin, requestOrigin)
}
