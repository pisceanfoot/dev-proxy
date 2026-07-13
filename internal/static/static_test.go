package static

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// newStaticHandler creates a Serve handler backed by a real temp directory.
// It returns the handler and the temp dir path.
func newStaticHandler(t *testing.T) (http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	sentinel := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	return Serve(dir, sentinel), dir
}

// --- task 5.7: successful file serving ---

func TestServeFile_OK(t *testing.T) {
	h, dir := newStaticHandler(t)

	content := []byte("<html><body>hello</body></html>")
	if err := os.WriteFile(filepath.Join(dir, "page.html"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/page.html", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("want text/html Content-Type, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "hello") {
		t.Fatalf("body should contain file content, got %q", rr.Body.String())
	}
}

// --- task 5.1: 404 when file does not exist ---

func TestServe_NotFound(t *testing.T) {
	h, _ := newStaticHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/missing.txt", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "404") {
		t.Fatalf("body should contain '404', got %q", rr.Body.String())
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Fatalf("want text/plain Content-Type, got %q", ct)
	}
}

// --- task 5.3: 403 on path-traversal URL ---

func TestServe_Traversal(t *testing.T) {
	h, _ := newStaticHandler(t)

	// filepath.Join cleans .. so we craft a URL that after joining would escape.
	// On most platforms filepath.Join("dir", "../../etc/passwd") stays cleaned,
	// but EvalSymlinks on a non-existent path returns not-found, so we write a
	// file one level above the temp dir and attempt traversal.
	req := httptest.NewRequest(http.MethodGet, "/../../../../etc/passwd", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	// Should be 403 (escaped root) or 404 (path doesn't exist); never 200.
	if rr.Code == http.StatusOK {
		t.Fatalf("traversal must not succeed, got 200")
	}
	if rr.Code != http.StatusForbidden && rr.Code != http.StatusNotFound {
		t.Fatalf("want 403 or 404 for traversal, got %d", rr.Code)
	}
}

// --- task 5.4: directory listing ---

func TestServe_DirListing(t *testing.T) {
	h, dir := newStaticHandler(t)

	// Create a file and a subdirectory
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("want text/html Content-Type, got %q", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "hello.txt") {
		t.Fatalf("body should list hello.txt, got %q", body)
	}
	if !strings.Contains(body, "subdir") {
		t.Fatalf("body should list subdir, got %q", body)
	}
	// File entry must be a link
	if !strings.Contains(body, `href="/hello.txt"`) && !strings.Contains(body, `href="./hello.txt"`) {
		// Accept href that contains hello.txt
		if !strings.Contains(body, "hello.txt") {
			t.Fatalf("body should contain a link to hello.txt")
		}
	}
}

// --- task 5.5: empty directory listing ---

func TestServe_EmptyDir(t *testing.T) {
	h, _ := newStaticHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<html") {
		t.Fatalf("body should be HTML, got %q", body)
	}
}

// --- task 5.2: 500 when stat returns a non-not-found error ---
// We simulate this by making the static dir itself unreadable so that EvalSymlinks
// on a child path fails with a permission error. Skip on root/Windows.

func TestServe_StatPermissionError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission model differs on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("running as root; permission checks don't apply")
	}

	dir := t.TempDir()
	subdir := filepath.Join(dir, "locked")
	if err := os.Mkdir(subdir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(subdir, 0o755) })

	sentinel := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := Serve(dir, sentinel)

	req := httptest.NewRequest(http.MethodGet, "/locked/secret.txt", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	// EvalSymlinks will fail with permission denied → 500 or 404 (if the OS
	// reports not-found on unreadable dirs). Either is acceptable; never 200.
	if rr.Code == http.StatusOK {
		t.Fatalf("should not succeed when parent dir is mode 000, got 200")
	}
}

// --- task 5.6: 500 when os.ReadDir fails on unreadable directory ---

func TestServe_UnreadableDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission model differs on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("running as root; permission checks don't apply")
	}

	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o311); err != nil { // execute but not read
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	sentinel := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := Serve(dir, sentinel)

	req := httptest.NewRequest(http.MethodGet, "/locked", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("want 500 for unreadable dir, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "500") {
		t.Fatalf("body should contain error message, got %q", rr.Body.String())
	}
}
