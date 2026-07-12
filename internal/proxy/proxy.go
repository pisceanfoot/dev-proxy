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
func NewReverseProxy(upstream string, rewriteHost bool, insecure bool) (*httputil.ReverseProxy, error) {
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

	// Custom Director to handle Host header rewriting and path stripping.
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		if rewriteHost && u.Host != "" {
			req.Host = u.Host
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
