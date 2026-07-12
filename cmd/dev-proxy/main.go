package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"dev-proxy/internal/config"
	"dev-proxy/internal/cors"
	devtls "dev-proxy/internal/tls"
	"dev-proxy/internal/logger"
	"dev-proxy/internal/proxy"
	"dev-proxy/internal/router"
	"dev-proxy/internal/shutdown"
	"dev-proxy/internal/static"
	"dev-proxy/internal/watcher"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[dev-proxy] Failed to load configuration: %v", err)
	}

	// Initialise logger level from config (default "error" = silent).
	level, err := logger.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatalf("[dev-proxy] %v", err)
	}
	logger.SetLevel(level)

	cm := devtls.NewCertManager()
	sm := shutdown.New(5 * time.Second)

	groups := buildHostGroups(cfg)
	rt := router.New(groups)

	servers := startServers(cfg, rt, cm, sm)

	logStartupInfo(cfg, groups)

	w, err := watcher.New(cfg.ConfigPath, func() error {
		newCfg, err := config.Load()
		if err != nil {
			return err
		}
		newGroups := buildHostGroups(newCfg)
		rt = router.New(newGroups)
		logger.Info("config reloaded")
		return nil
	})
	if err == nil {
		sm.Register(w.Close)
		if err := w.Start(); err != nil {
			logger.Error("watcher: %v", err)
		}
	}

	sm.Wait()
	shutdownServers(servers)
}

// logStartupInfo emits info-level startup summary after servers are bound.
func logStartupInfo(cfg *config.Config, groups []router.HostGroup) {
	for _, port := range cfg.Server.ListenPorts {
		scheme := "http"
		if isPortTLSEnabled(port, cfg) {
			scheme = "https"
		}
		logger.Info("listening on %s://localhost:%d", scheme, port)
	}

	totalRoutes := 0
	for _, g := range groups {
		totalRoutes += len(g.Routes)
	}
	logger.Info("%d host group(s), %d route(s) configured", len(groups), totalRoutes)
}

// resolveUpstream resolves a route's upstream field to a concrete URL plus
// connection settings. Inline URLs (containing "://") are used as-is; otherwise
// the value is treated as a name and looked up in cfg.Upstreams.
func resolveUpstream(rc config.RouteConfig, upstreams map[string]config.UpstreamConfig) (upstreamURL string, rewriteHost bool, insecure bool) {
	if strings.Contains(rc.Upstream, "://") {
		return rc.Upstream, rc.RewriteHost, rc.Insecure
	}
	up, ok := upstreams[rc.Upstream]
	if !ok {
		log.Fatalf("[dev-proxy] unknown upstream %q (should have been caught at startup)", rc.Upstream)
	}
	return up.URL, up.RewriteHost, up.Insecure
}

// buildMatchedRoute converts a config RouteConfig into a router.MatchedRoute,
// resolving named upstreams and compiling regex patterns.
func buildMatchedRoute(i int, rc config.RouteConfig, upstreams map[string]config.UpstreamConfig) router.MatchedRoute {
	corsCfg := &router.CORSConfig{}
	if rc.CORSAllowOrigin != "" {
		corsCfg.Enabled = true
		corsCfg.AllowOrigin = rc.CORSAllowOrigin
	}

	upstreamURL, rewriteHost, insecure := resolveUpstream(rc, upstreams)

	mr := router.MatchedRoute{
		PathPrefix:   rc.PathPrefix,
		PathExact:    rc.PathExact,
		HostMatch:    rc.HostMatch,
		Upstream:     upstreamURL,
		UpstreamPath: rc.UpstreamPath,
		RewriteHost:  rewriteHost,
		CORS:         corsCfg,
		StaticDir:    rc.StaticDir,
		Insecure:     insecure,
	}

	if rc.PathRegex != "" {
		re, err := regexp.Compile(rc.PathRegex)
		if err != nil {
			log.Fatalf("[dev-proxy] route %d: invalid path_regex %q: %v", i, rc.PathRegex, err)
		}
		mr.PathRegex = re
	}

	return mr
}

// buildHostGroups constructs router.HostGroup slices from config.
// When cfg.Hosts is defined it takes precedence; cfg.Routes is wrapped in an
// implicit "*" group otherwise.
func buildHostGroups(cfg *config.Config) []router.HostGroup {
	if len(cfg.Hosts) > 0 {
		var groups []router.HostGroup
		for _, hg := range cfg.Hosts {
			var routes []router.MatchedRoute
			for i, rc := range hg.Routes {
				routes = append(routes, buildMatchedRoute(i, rc, cfg.Upstreams))
			}
			groups = append(groups, router.HostGroup{
				Match:  hg.Match,
				Routes: routes,
			})
		}
		return groups
	}

	var routes []router.MatchedRoute
	for i, rc := range cfg.Routes {
		routes = append(routes, buildMatchedRoute(i, rc, cfg.Upstreams))
	}
	if len(routes) == 0 {
		routes = append(routes, router.MatchedRoute{
			PathPrefix:  "/",
			RewriteHost: true,
		})
	}
	return []router.HostGroup{{Match: "*", Routes: routes}}
}

func startServers(cfg *config.Config, rt *router.Router, cm *devtls.CertManager, sm *shutdown.Manager) map[int]*http.Server {
	servers := make(map[int]*http.Server)

	for _, port := range cfg.Server.ListenPorts {
		isTLS := isPortTLSEnabled(port, cfg)
		handler := buildHandler(rt, &cfg.Server, isTLS)

		addr := fmt.Sprintf(":%d", port)
		srv := &http.Server{
			Addr:    addr,
			Handler: handler,
		}

		if isTLS {
			var cp *devtls.CertPair
			var err error

			if cfg.Server.TLS != nil && cfg.Server.TLS.CertFile != "" {
				cp, err = cm.LoadFromDisk(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
				if err != nil {
					log.Fatalf("[dev-proxy] Failed to load certificate for port %d: %v", port, err)
				}
			} else {
				cp, err = cm.GetOrGenerate("server")
				if err != nil {
					log.Fatalf("[dev-proxy] Failed to generate self-signed certificate for port %d: %v", port, err)
				}
			}
			tlsCfg := &tls.Config{Certificates: []tls.Certificate{}}
			cert, err := tls.X509KeyPair(cp.CertPEM, cp.KeyPEM)
			if err != nil {
				log.Fatalf("[dev-proxy] Failed to parse certificate for port %d: %v", port, err)
			}
			tlsCfg.Certificates = append(tlsCfg.Certificates, cert)
			srv.TLSConfig = tlsCfg

			go func(p int) {
				if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
					handlePortError(p, err)
				}
			}(port)
		} else {
			go func(p int) {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					handlePortError(p, err)
				}
			}(port)
		}

		servers[port] = srv
		sm.Register(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return srv.Shutdown(ctx)
		})
	}

	return servers
}

func isPortTLSEnabled(port int, cfg *config.Config) bool {
	if cfg.Server.TLS == nil || !cfg.Server.TLS.Enabled {
		return false
	}
	maxPort := 0
	for _, p := range cfg.Server.ListenPorts {
		if p > maxPort {
			maxPort = p
		}
	}
	return port == maxPort
}

func handlePortError(port int, err error) {
	if strings.Contains(err.Error(), "address already in use") {
		log.Fatalf("[dev-proxy] FATAL: port %d is already in use — another process is bound to it. Run 'lsof -i :%d' or 'netstat -an | grep %d' to find the conflicting process.",
			port, port, port)
	}
	log.Fatalf("[dev-proxy] FATAL: server on port %d error: %v", port, err)
}

// computeUpstreamURL reconstructs the URL that the proxy Director will forward
// to, applying the same path-rewriting logic as proxy.NewReverseProxy.
func computeUpstreamURL(route *router.MatchedRoute, reqPath string) string {
	if route.UpstreamPath != "" {
		suffix := strings.TrimPrefix(reqPath, route.PathPrefix)
		if !strings.HasPrefix(suffix, "/") {
			suffix = "/" + suffix
		}
		return route.Upstream + route.UpstreamPath + suffix
	}
	return route.Upstream + reqPath
}

// routeCriterion returns a compact tag for the most-specific active path
// criterion on a route: regex > exact > prefix > "*".
func routeCriterion(route *router.MatchedRoute) string {
	if route.PathRegex != nil {
		return "regex:" + route.PathRegex.String()
	}
	if route.PathExact != "" {
		return "exact:" + route.PathExact
	}
	if route.PathPrefix != "" {
		return "prefix:" + route.PathPrefix
	}
	return "*"
}

func buildHandler(rt *router.Router, serverCfg *config.ServerConfig, isTLS bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		result := rt.Match(r)

		if result == nil {
			lw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusGatewayTimeout}
			router.HandleNoMatch(lw, r)
			logger.Debug("%s %s host=%s group=- route=- upstream=- status=%d duration=%v",
				r.Method, r.URL.Path, r.Host, lw.statusCode, time.Since(start))
			return
		}

		route := result.Route

		var handler http.Handler

		if route.Upstream != "" {
			rp, err := proxy.NewReverseProxy(route.Upstream, route.RewriteHost, route.Insecure, route.PathPrefix, route.UpstreamPath)
			if err != nil {
				log.Fatalf("Failed to create reverse proxy for upstream %s: %v", route.Upstream, err)
			}
			handler = rp
		} else if route.StaticDir != "" {
			handler = http.NotFoundHandler()
		}

		if handler == nil {
			lw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusGatewayTimeout}
			router.HandleNoMatch(lw, r)
			logger.Debug("%s %s host=%s group=- route=- upstream=- status=%d duration=%v",
				r.Method, r.URL.Path, r.Host, lw.statusCode, time.Since(start))
			return
		}

		if route.StaticDir != "" {
			handler = static.Serve(route.StaticDir, handler)
		}

		if route.CORS != nil && route.CORS.Enabled {
			handler = cors.Middleware(handler, route.CORS)
		}

		if serverCfg.RedirectHTTP && !isTLS {
			targetPort := findTLSPort(serverCfg.ListenPorts)
			if targetPort > 0 {
				handler = redirectMiddleware(handler, targetPort)
			}
		}

		loggingMiddleware(handler, route, result.HostGroupPattern, start).ServeHTTP(w, r)
	})
}

func findTLSPort(ports []int) int {
	max := 0
	for _, p := range ports {
		if p > max {
			max = p
		}
	}
	return max
}

func redirectMiddleware(next http.Handler, targetPort int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if host == "" {
			host = "localhost"
		}
		target := fmt.Sprintf("https://%s:%d%s", host, targetPort, r.URL.Path)
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}

func loggingMiddleware(next http.Handler, route *router.MatchedRoute, hostGroupPattern string, start time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lw, r)

		upstreamURL := computeUpstreamURL(route, r.URL.Path)
		logger.Debug("%s %s host=%s group=%s route=%s upstream=%s upstream_url=%s status=%d duration=%v",
			r.Method, r.URL.Path, r.Host,
			hostGroupPattern,
			routeCriterion(route),
			route.Upstream,
			upstreamURL,
			lw.statusCode,
			time.Since(start))
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lw *loggingResponseWriter) WriteHeader(code int) {
	lw.statusCode = code
	lw.ResponseWriter.WriteHeader(code)
}

func shutdownServers(servers map[int]*http.Server) {
	for port, srv := range servers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("shutting down server on port %d: %v", port, err)
		}
	}
}
