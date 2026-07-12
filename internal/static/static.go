package static

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Serve checks if a static file exists for the request path and serves it.
// If no matching file is found, calls next.ServeHTTP to fall through to upstream.
func Serve(staticDir string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if staticDir == "" {
			next.ServeHTTP(w, r)
			return
		}

		filePath := filepath.Join(staticDir, filepath.FromSlash(r.URL.Path))

		info, err := os.Stat(filePath)
		if err != nil || info.IsDir() {
			next.ServeHTTP(w, r)
			return
		}

		contentType := detectContentType(filePath)
		w.Header().Set("Content-Type", contentType)
		http.ServeFile(w, r, filePath)
	})
}

func detectContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	default:
		return "application/octet-stream"
	}
}
