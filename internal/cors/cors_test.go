package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"dev-proxy/internal/router"
)

// sentinel is a simple handler that records it was called.
var sentinel = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func enabledCfg(origin string) *router.CORSConfig {
	return &router.CORSConfig{Enabled: true, AllowOrigin: origin}
}

// --- task 4.4: GET with Origin receives Access-Control-Allow-Origin ---

func TestCORS_GET_WithOrigin(t *testing.T) {
	h := Middleware(sentinel, enabledCfg("https://example.com"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("want Access-Control-Allow-Origin: https://example.com, got %q", got)
	}
}

// --- task 4.5: POST with Origin receives Access-Control-Allow-Origin ---

func TestCORS_POST_WithOrigin(t *testing.T) {
	h := Middleware(sentinel, enabledCfg("*"))

	req := httptest.NewRequest(http.MethodPost, "/api", nil)
	req.Header.Set("Origin", "https://other.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("want Access-Control-Allow-Origin: *, got %q", got)
	}
}

// PUT and DELETE follow the same code path; one representative test is sufficient.
func TestCORS_PUT_WithOrigin(t *testing.T) {
	h := Middleware(sentinel, enabledCfg("https://example.com"))

	req := httptest.NewRequest(http.MethodPut, "/resource", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Fatal("want Access-Control-Allow-Origin header, got empty")
	}
}

func TestCORS_DELETE_WithOrigin(t *testing.T) {
	h := Middleware(sentinel, enabledCfg("https://example.com"))

	req := httptest.NewRequest(http.MethodDelete, "/resource/1", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Fatal("want Access-Control-Allow-Origin header, got empty")
	}
}

// --- task 4.6: OPTIONS preflight receives 204 with CORS headers ---

func TestCORS_Preflight_OPTIONS(t *testing.T) {
	h := Middleware(sentinel, enabledCfg("https://example.com"))

	req := httptest.NewRequest(http.MethodOptions, "/api", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("want Access-Control-Allow-Origin, got %q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("want Access-Control-Allow-Methods header, got empty")
	}
	if got := rr.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("want Access-Control-Allow-Headers header, got empty")
	}
}

// --- task 4.7: request without Origin receives no CORS headers ---

func TestCORS_NoOrigin_NoHeaders(t *testing.T) {
	h := Middleware(sentinel, enabledCfg("https://example.com"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// intentionally no Origin header
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("want no Access-Control-Allow-Origin, got %q", got)
	}
}

// --- additional: disabled CORS passes through with no headers ---

func TestCORS_Disabled_NoHeaders(t *testing.T) {
	h := Middleware(sentinel, &router.CORSConfig{Enabled: false})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("want no CORS headers when disabled, got %q", got)
	}
}
