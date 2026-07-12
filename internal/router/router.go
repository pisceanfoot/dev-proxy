package router

import (
	"net/http"
	"strings"
)

// CORSConfig holds per-route CORS settings.
type CORSConfig struct {
	Enabled          bool
	AllowOrigin      string // "*" or specific origin
	ForwardUpstream  bool   // if true, pass through upstream CORS headers instead of injecting own
}

// MatchedRoute is the result of matching a request against the route table.
type MatchedRoute struct {
	LocalPort    int
	PathPrefix   string
	HostMatch    string
	Upstream     string
	RewriteHost  bool
	CORS         *CORSConfig
	StaticDir    string
	ReviewMode   bool
	Insecure     bool
	TLSEnabled   bool
}

// Router evaluates incoming requests against an ordered list of routes.
type Router struct {
	routes []MatchedRoute
}

// New creates a Router from a slice of matched routes.
func New(routes []MatchedRoute) *Router {
	return &Router{routes: routes}
}

// Match finds the first route matching the request's host and path.
// Returns nil if no route matches.
func (r *Router) Match(req *http.Request) *MatchedRoute {
	for i := range r.routes {
		rt := &r.routes[i]
		if rt.HostMatch != "" && req.Host != rt.HostMatch {
			continue
		}
		if strings.HasPrefix(req.URL.Path, rt.PathPrefix) {
			return rt
		}
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

// HandleNotFound writes a 404 response.
func HandleNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 page not found"))
}
