package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNewReverseProxy_InvalidURL(t *testing.T) {
	_, err := NewReverseProxy("://invalid-url", false, "", "")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestNewReverseProxy_ValidUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	proxy, err := NewReverseProxy(upstream.URL, false, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proxy == nil {
		t.Fatal("expected proxy, got nil")
	}
}

func TestNewReverseProxy_RewriteHost(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	proxyURL, _ := url.Parse(upstream.URL)

	proxy, err := NewReverseProxy(upstream.URL, false, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "original-host"

	// Use the proxy's Director directly to test host rewriting
	proxy.Director(req)

	if req.Host != proxyURL.Host {
		t.Fatalf("expected Host=%q, got %q", proxyURL.Host, req.Host)
	}
}

func TestNewReverseProxy_InsecureHTTPS(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	proxy, err := NewReverseProxy(upstream.URL, true, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proxy == nil {
		t.Fatal("expected proxy, got nil")
	}
	if proxy.Transport == nil {
		t.Fatal("expected transport to be set")
	}
}

func TestNewReverseProxy_PathRewriting(t *testing.T) {
	tests := []struct {
		name         string
		routePrefix  string
		upstreamPath string
		requestPath  string
		wantPath     string
	}{
		{
			name:         "strip prefix and prepend upstream",
			routePrefix:  "/api",
			upstreamPath: "/v1",
			requestPath:  "/api/users",
			wantPath:     "/v1/users",
		},
		{
			name:         "exact prefix match",
			routePrefix:  "/api",
			upstreamPath: "/v1",
			requestPath:  "/api",
			wantPath:     "/v1/",
		},
		{
			name:         "no upstreamPath",
			routePrefix:  "/api",
			upstreamPath: "",
			requestPath:  "/api/users",
			wantPath:     "/api/users",
		},
		{
			name:         "routePrefix not matching",
			routePrefix:  "/api",
			upstreamPath: "/v1",
			requestPath:  "/other/users",
			wantPath:     "/v1/other/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			defer upstream.Close()

			proxy, err := NewReverseProxy(upstream.URL, false, tt.routePrefix, tt.upstreamPath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			proxy.Director(req)

			if req.URL.Path != tt.wantPath {
				t.Fatalf("expected path %q, got %q", tt.wantPath, req.URL.Path)
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("proxied"))
	}))
	defer upstream.Close()

	proxy, err := NewReverseProxy(upstream.URL, false, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	handler := ServeHTTP(proxy)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	if string(body) != "proxied" {
		t.Fatalf("expected body %q, got %q", "proxied", string(body))
	}
}

func TestServeHTTP_WithPathPrefix(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(r.URL.Path))
	}))
	defer upstream.Close()

	proxy, err := NewReverseProxy(upstream.URL, false, "/api", "/v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	handler := ServeHTTP(proxy)
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	if !strings.HasPrefix(string(body), "/v1/") {
		t.Fatalf("expected body to start with /v1/, got %q", string(body))
	}
}
