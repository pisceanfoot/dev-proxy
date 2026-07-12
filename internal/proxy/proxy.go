package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// NewReverseProxy creates an httputil.ReverseProxy for the given upstream URL.
// If rewriteHost is true, the outgoing Host header will be set to the upstream hostname.
// If insecure is true, TLS verification against the upstream is skipped.
// If upstreamPath is non-empty, the forwarded path is rewritten: routePrefix is stripped
// from the request path and upstreamPath is prepended to the remainder.
func NewReverseProxy(upstream string, rewriteHost bool, insecure bool, routePrefix string, upstreamPath string) (*httputil.ReverseProxy, error) {
	u, err := url.Parse(upstream)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if insecure && strings.HasPrefix(u.Scheme, "https") {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.Transport = transport

	// Custom Director to handle Host header rewriting and optional path rewriting.
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		if rewriteHost && u.Host != "" {
			req.Host = u.Host
		}
		if upstreamPath != "" {
			suffix := strings.TrimPrefix(req.URL.Path, routePrefix)
			if !strings.HasPrefix(suffix, "/") {
				suffix = "/" + suffix
			}
			req.URL.Path = upstreamPath + suffix
			req.URL.RawPath = "" // clear to force re-encoding
		}
	}

	return proxy, nil
}

// ServeHTTP proxies the request to the upstream. The path in req.URL is expected
// to already have the prefix stripped by the caller (router).
func ServeHTTP(proxy *httputil.ReverseProxy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})
}
