package router

import (
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
)

// CORSConfig holds per-route CORS settings.
type CORSConfig struct {
	Enabled         bool
	AllowOrigin     string // "*" or specific origin
}

// MatchedRoute is the result of matching a request against the route table.
type MatchedRoute struct {
	LocalPort         int
	PathPrefix        string
	PathExact         string
	PathRegex         *regexp.Regexp
	HostMatch         string
	Upstream          string
	UpstreamPath      string
	URLRewriteRegex   *regexp.Regexp
	URLRewriteReplace string
	CORS              *CORSConfig
	StaticDir         string
	ReviewMode        bool
	Insecure          bool
	TLSEnabled        bool
}

// HostGroup is an ordered set of routes behind a host glob pattern.
type HostGroup struct {
	Match  string
	Routes []MatchedRoute
}

// Router evaluates incoming requests using two-phase host-then-path matching.
// When host groups are present, the first group whose Match pattern fits the
// request Host is selected; path matching is then confined to that group.
// When no host groups are provided (flat routes), all routes are wrapped in a
// single implicit group that matches any host ("*").
type Router struct {
	hostGroups []HostGroup
}

// New creates a Router from an ordered list of host groups.
func New(groups []HostGroup) *Router {
	return &Router{hostGroups: groups}
}

// NewFromRoutes wraps a flat route slice in an implicit catch-all host group.
func NewFromRoutes(routes []MatchedRoute) *Router {
	return New([]HostGroup{{Match: "*", Routes: routes}})
}

// MatchResult carries the outcome of a Router.Match call, including the
// host group that was selected and the specific route within it.
type MatchResult struct {
	HostGroupPattern string // the matched HostGroup's Match field, e.g. "api.local"
	Route            *MatchedRoute
}

// Match performs two-phase matching:
//  1. Find the first HostGroup whose Match pattern fits req.Host.
//  2. Within that group, find the first route where all path criteria pass.
//
// Returns nil when no host group or no path matches.
func (r *Router) Match(req *http.Request) *MatchResult {
	for i := range r.hostGroups {
		group := &r.hostGroups[i]

		matched, err := path.Match(group.Match, req.Host)
		if err != nil || !matched {
			continue
		}

		// Host matched — search paths within this group only (no fall-through).
		for j := range group.Routes {
			rt := &group.Routes[j]

			if rt.PathExact != "" && req.URL.Path != rt.PathExact {
				continue
			}
			if rt.PathPrefix != "" && !strings.HasPrefix(req.URL.Path, rt.PathPrefix) {
				continue
			}
			if rt.PathRegex != nil && !rt.PathRegex.MatchString(req.URL.Path) {
				continue
			}

			return &MatchResult{HostGroupPattern: group.Match, Route: rt}
		}

		return nil // host matched but no path matched
	}
	return nil
}

// StripPrefix returns the path with the route's prefix removed.
func (r *Router) StripPrefix(rt *MatchedRoute, reqPath string) string {
	if !strings.HasPrefix(reqPath, rt.PathPrefix) {
		return reqPath
	}
	stripped := reqPath[len(rt.PathPrefix):]
	if stripped == "" {
		return "/"
	}
	return stripped
}

// HandleNoMatch writes a 504 response with the unmatched host and path.
func HandleNoMatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusGatewayTimeout)
	fmt.Fprintf(w, "no route matched: host=%s path=%s\n", r.Host, r.URL.Path)
}
