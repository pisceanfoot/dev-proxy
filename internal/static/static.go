package static

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// dirListTmpl renders a simple nginx-style directory listing.
var dirListTmpl = template.Must(template.New("dirlist").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Index of {{.Path}}</title>
<style>
body { font-family: monospace; padding: 1rem; }
table { border-collapse: collapse; width: 100%; }
th, td { text-align: left; padding: 4px 12px; }
th { border-bottom: 1px solid #ccc; }
a { text-decoration: none; color: #06c; }
a:hover { text-decoration: underline; }
.size { text-align: right; }
</style>
</head>
<body>
<h1>Index of {{.Path}}</h1>
<table>
<tr><th>Name</th><th>Last Modified</th><th class="size">Size</th></tr>
{{range .Entries}}<tr>
  <td><a href="{{.Href}}">{{.Name}}</a></td>
  <td>{{.ModTime}}</td>
  <td class="size">{{.Size}}</td>
</tr>{{end}}
</table>
</body>
</html>
`))

type dirEntry struct {
	Name    string
	Href    string
	ModTime string
	Size    string
}

type dirListData struct {
	Path    string
	Entries []dirEntry
}

// Serve checks if a static file exists for the request path and serves it.
// When static_dir is configured this handler is authoritative:
//   - 403 if the resolved path escapes staticDir
//   - 404 if the path does not exist
//   - 500 on unexpected OS errors
//   - HTML directory listing when the path is a directory
//   - File content otherwise
//
// The next handler is only called when staticDir is empty.
func Serve(staticDir string, next http.Handler) http.Handler {
	// Resolve symlinks in the root once at construction time (task 1.1).
	resolvedRoot, err := filepath.EvalSymlinks(staticDir)
	if err != nil {
		// If the dir doesn't exist yet or can't be resolved, keep the raw path;
		// every request will receive a 500 at stat time anyway.
		resolvedRoot = staticDir
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if staticDir == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Build and sanitise the candidate path.
		joined := filepath.Join(resolvedRoot, filepath.FromSlash(r.URL.Path))

		// Resolve symlinks on the full path to catch symlink-based traversal (task 1.2).
		filePath, err := filepath.EvalSymlinks(joined)
		if err != nil {
			if os.IsNotExist(err) {
				// Path doesn't exist — return 404 (task 2.1 / 2.3).
				http.Error(w, "404 Not Found", http.StatusNotFound)
				return
			}
			// Other error (permission denied, I/O error) — return 500 (task 2.2 / 2.3).
			http.Error(w, fmt.Sprintf("500 Internal Server Error: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		// Containment check: resolved path must be inside resolvedRoot (task 1.2).
		if !strings.HasPrefix(filePath, resolvedRoot+string(os.PathSeparator)) && filePath != resolvedRoot {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}

		info, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "404 Not Found", http.StatusNotFound)
				return
			}
			http.Error(w, fmt.Sprintf("500 Internal Server Error: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		if info.IsDir() {
			serveDirectoryListing(w, r, filePath)
			return
		}

		contentType := detectContentType(filePath)
		w.Header().Set("Content-Type", contentType)
		http.ServeFile(w, r, filePath)
	})
}

// serveDirectoryListing renders an HTML directory listing for the given dir path (tasks 3.1–3.4).
func serveDirectoryListing(w http.ResponseWriter, r *http.Request, dirPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("500 Internal Server Error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	urlPath := r.URL.Path
	if !strings.HasSuffix(urlPath, "/") {
		urlPath += "/"
	}

	var items []dirEntry
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}

		name := e.Name()
		href := urlPath + name
		sizeStr := "-"
		if !e.IsDir() {
			sizeStr = fmt.Sprintf("%d", info.Size())
		} else {
			href += "/"
			name += "/"
		}

		items = append(items, dirEntry{
			Name:    name,
			Href:    href,
			ModTime: info.ModTime().UTC().Format(time.RFC1123),
			Size:    sizeStr,
		})
	}

	data := dirListData{
		Path:    urlPath,
		Entries: items,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = dirListTmpl.Execute(w, data)
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
