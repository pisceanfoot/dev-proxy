package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	groups := []HostGroup{
		{Match: "*.example.com", Routes: []MatchedRoute{{PathExact: "/"}}},
	}
	r := New(groups)
	if r == nil {
		t.Fatal("expected Router, got nil")
	}
	if len(r.hostGroups) != 1 {
		t.Fatalf("expected 1 host group, got %d", len(r.hostGroups))
	}
}

func TestNewFromRoutes(t *testing.T) {
	routes := []MatchedRoute{{PathExact: "/"}}
	r := NewFromRoutes(routes)
	if r == nil {
		t.Fatal("expected Router, got nil")
	}
	if len(r.hostGroups) != 1 {
		t.Fatalf("expected 1 host group, got %d", len(r.hostGroups))
	}
	if r.hostGroups[0].Match != "*" {
		t.Fatalf("expected catch-all host group, got %q", r.hostGroups[0].Match)
	}
}

func TestMatch_ExactPath(t *testing.T) {
	routes := []MatchedRoute{
		{PathExact: "/health", Upstream: "http://localhost:8080"},
		{PathExact: "/api", Upstream: "http://localhost:8081"},
	}
	r := NewFromRoutes(routes)

	tests := []struct {
		name    string
		path    string
		wantNil bool
		wantUpstream string
	}{
		{"exact match health", "/health", false, "http://localhost:8080"},
		{"exact match api", "/api", false, "http://localhost:8081"},
		{"no match", "/other", true, ""},
		{"trailing slash", "/health/", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			result := r.Match(req)
			if tt.wantNil {
				if result != nil {
					t.Fatalf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected match, got nil")
			}
			if result.Route.Upstream != tt.wantUpstream {
				t.Fatalf("expected upstream %q, got %q", tt.wantUpstream, result.Route.Upstream)
			}
		})
	}
}

func TestMatch_PrefixPath(t *testing.T) {
	routes := []MatchedRoute{
		{PathPrefix: "/api", Upstream: "http://localhost:8080"},
		{PathPrefix: "/static", Upstream: "http://localhost:8081"},
	}
	r := NewFromRoutes(routes)

	tests := []struct {
		name    string
		path    string
		wantNil bool
		wantUpstream string
	}{
		{"prefix match api", "/api/users", false, "http://localhost:8080"},
		{"prefix match static", "/static/css/app.css", false, "http://localhost:8081"},
		{"exact prefix", "/api", false, "http://localhost:8080"},
		{"no match", "/other", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			result := r.Match(req)
			if tt.wantNil {
				if result != nil {
					t.Fatalf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected match, got nil")
			}
			if result.Route.Upstream != tt.wantUpstream {
				t.Fatalf("expected upstream %q, got %q", tt.wantUpstream, result.Route.Upstream)
			}
		})
	}
}

func TestMatch_RegexPath(t *testing.T) {
	routes := []MatchedRoute{
		{PathRegex: regexp.MustCompile(`^/v[0-9]+/.*$`), Upstream: "http://localhost:8080"},
		{PathRegex: regexp.MustCompile(`^/api/.*$`), Upstream: "http://localhost:8081"},
	}
	r := NewFromRoutes(routes)

	tests := []struct {
		name    string
		path    string
		wantNil bool
		wantUpstream string
	}{
		{"regex match v1", "/v1/users", false, "http://localhost:8080"},
		{"regex match api", "/api/items", false, "http://localhost:8081"},
		{"no match", "/other", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			result := r.Match(req)
			if tt.wantNil {
				if result != nil {
					t.Fatalf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected match, got nil")
			}
			if result.Route.Upstream != tt.wantUpstream {
				t.Fatalf("expected upstream %q, got %q", tt.wantUpstream, result.Route.Upstream)
			}
		})
	}
}

func TestMatch_HostGroup(t *testing.T) {
	groups := []HostGroup{
		{
			Match: "api.example.com",
			Routes: []MatchedRoute{
				{PathExact: "/", Upstream: "http://api-backend"},
			},
		},
		{
			Match: "*.example.com",
			Routes: []MatchedRoute{
				{PathExact: "/", Upstream: "http://wildcard-backend"},
			},
		},
	}
	r := New(groups)

	tests := []struct {
		name    string
		host    string
		path    string
		wantNil bool
		wantUpstream string
		wantHostGroup string
	}{
		{"exact host match", "api.example.com", "/", false, "http://api-backend", "api.example.com"},
		{"wildcard host match", "www.example.com", "/", false, "http://wildcard-backend", "*.example.com"},
		{"no host match", "other.com", "/", true, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Host = tt.host
			result := r.Match(req)
			if tt.wantNil {
				if result != nil {
					t.Fatalf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected match, got nil")
			}
			if result.Route.Upstream != tt.wantUpstream {
				t.Fatalf("expected upstream %q, got %q", tt.wantUpstream, result.Route.Upstream)
			}
			if result.HostGroupPattern != tt.wantHostGroup {
				t.Fatalf("expected host group %q, got %q", tt.wantHostGroup, result.HostGroupPattern)
			}
		})
	}
}

func TestMatch_NoMatch(t *testing.T) {
	t.Run("no host groups", func(t *testing.T) {
		r := New([]HostGroup{})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		result := r.Match(req)
		if result != nil {
			t.Fatalf("expected nil, got %+v", result)
		}
	})

	t.Run("host matched but no path", func(t *testing.T) {
		groups := []HostGroup{
			{
				Match: "*",
				Routes: []MatchedRoute{
					{PathExact: "/exact", Upstream: "http://localhost"},
				},
			},
		}
		r := New(groups)
		req := httptest.NewRequest(http.MethodGet, "/other", nil)
		result := r.Match(req)
		if result != nil {
			t.Fatalf("expected nil, got %+v", result)
		}
	})

	t.Run("no routes in flat", func(t *testing.T) {
		r := NewFromRoutes([]MatchedRoute{})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		result := r.Match(req)
		if result != nil {
			t.Fatalf("expected nil, got %+v", result)
		}
	})
}

func TestStripPrefix(t *testing.T) {
	r := NewFromRoutes([]MatchedRoute{})

	tests := []struct {
		name       string
		route      *MatchedRoute
		reqPath    string
		wantResult string
	}{
		{"matching prefix", &MatchedRoute{PathPrefix: "/api"}, "/api/users", "/users"},
		{"exact prefix", &MatchedRoute{PathPrefix: "/api"}, "/api", "/"},
		{"non-matching prefix", &MatchedRoute{PathPrefix: "/api"}, "/other", "/other"},
		{"empty suffix", &MatchedRoute{PathPrefix: "/api/"}, "/api/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.StripPrefix(tt.route, tt.reqPath)
			if got != tt.wantResult {
				t.Fatalf("expected %q, got %q", tt.wantResult, got)
			}
		})
	}
}

func TestHandleNoMatch(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.Host = "example.com"
	rec := httptest.NewRecorder()

	HandleNoMatch(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected status %d, got %d", http.StatusGatewayTimeout, rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Fatalf("expected Content-Type %q, got %q", "text/plain; charset=utf-8", contentType)
	}

	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "no route matched") {
		t.Fatalf("expected body to contain 'no route matched', got %q", string(body))
	}
	if !strings.Contains(string(body), "example.com") {
		t.Fatalf("expected body to contain host, got %q", string(body))
	}
	if !strings.Contains(string(body), "/missing") {
		t.Fatalf("expected body to contain path, got %q", string(body))
	}
}
