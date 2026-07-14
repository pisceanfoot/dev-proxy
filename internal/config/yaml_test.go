package config

import (
	"strings"
	"testing"
)

// minimalUpstreams returns a valid upstreams map for use in route tests.
func minimalUpstreams() map[string]UpstreamConfig {
	return map[string]UpstreamConfig{
		"backend": {URL: "http://localhost:8080"},
	}
}

// --- url_rewrite validation ---

func TestValidateRoute_URLRewrite_Valid(t *testing.T) {
	r := RouteConfig{
		PathPrefix: "/api",
		Upstream:   "http://upstream:9000",
		URLRewrite: &URLRewriteConfig{
			Match:   `^/api/(.*)`,
			Replace: "/v2/$1",
		},
	}
	if err := validateRoute(0, r, "", minimalUpstreams()); err != nil {
		t.Fatalf("want no error for valid url_rewrite, got: %v", err)
	}
}

func TestValidateRoute_URLRewrite_AndUpstreamPath_Rejected(t *testing.T) {
	r := RouteConfig{
		PathPrefix:   "/api",
		Upstream:     "http://upstream:9000",
		UpstreamPath: "/v2",
		URLRewrite:   &URLRewriteConfig{Match: `^/api/(.*)`, Replace: "/v2/$1"},
	}
	err := validateRoute(0, r, "", minimalUpstreams())
	if err == nil {
		t.Fatal("want error when url_rewrite and upstream_path are both set, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("want 'mutually exclusive' in error message, got: %v", err)
	}
}

func TestValidateRoute_URLRewrite_MissingMatch_Rejected(t *testing.T) {
	r := RouteConfig{
		Upstream:   "http://upstream:9000",
		URLRewrite: &URLRewriteConfig{Match: "", Replace: "/v2/$1"},
	}
	err := validateRoute(0, r, "", minimalUpstreams())
	if err == nil {
		t.Fatal("want error when url_rewrite.match is empty, got nil")
	}
	if !strings.Contains(err.Error(), "url_rewrite.match") {
		t.Fatalf("want 'url_rewrite.match' in error, got: %v", err)
	}
}

func TestValidateRoute_URLRewrite_MissingReplace_Rejected(t *testing.T) {
	r := RouteConfig{
		Upstream:   "http://upstream:9000",
		URLRewrite: &URLRewriteConfig{Match: `^/api/(.*)`, Replace: ""},
	}
	err := validateRoute(0, r, "", minimalUpstreams())
	if err == nil {
		t.Fatal("want error when url_rewrite.replace is empty, got nil")
	}
	if !strings.Contains(err.Error(), "url_rewrite.replace") {
		t.Fatalf("want 'url_rewrite.replace' in error, got: %v", err)
	}
}

func TestValidateRoute_URLRewrite_InvalidRegex_Rejected(t *testing.T) {
	r := RouteConfig{
		Upstream:   "http://upstream:9000",
		URLRewrite: &URLRewriteConfig{Match: "[invalid", Replace: "$1"},
	}
	err := validateRoute(0, r, "", minimalUpstreams())
	if err == nil {
		t.Fatal("want error for invalid url_rewrite.match regex, got nil")
	}
	if !strings.Contains(err.Error(), "[invalid") {
		t.Fatalf("want bad pattern in error message, got: %v", err)
	}
}

func TestValidateRoute_NoURLRewrite_Valid(t *testing.T) {
	r := RouteConfig{
		PathPrefix: "/",
		Upstream:   "http://upstream:9000",
	}
	if err := validateRoute(0, r, "", minimalUpstreams()); err != nil {
		t.Fatalf("want no error for route without url_rewrite, got: %v", err)
	}
}
