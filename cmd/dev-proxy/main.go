package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
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
	// Resolve config path once — flags are parsed only here, never in Load().
	configPath := "dev-proxy.yaml"
	if p := os.Getenv("DEV_PROXY_CONFIG"); p != "" {
		configPath = p
	}
	flag.StringVar(&configPath, "config", configPath, "Path to dev-proxy YAML config file")
	flag.Parse()

	cfg, err := config.Load(configPath)
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

	// Atomic router pointer — safe for concurrent reads during reload.
	var rt atomic.Pointer[router.Router]
	rt.Store(router.New(groups))

	// Snapshot server config at startup for change detection on reload.
	origServer := cfg.Server

	servers := startServers(cfg, &rt, cm, sm)
	logStartupInfo(cfg, groups)

	w, err := watcher.New(
		configPath,
		func() error {
			newCfg, err := config.Load(configPath)
			if err != nil {
				logger.Error("config reload failed: %v", err)
				return err
			}

			// Re-apply log level immediately — no restart required.
			newLevel, err := logger.ParseLevel(newCfg.LogLevel)
			if err != nil {
				logger.Error("config reload failed: invalid log_level: %v", err)
				return err
			}
			logger.SetLevel(newLevel)

			// Warn if any server-level config changed (requires restart).
			if changed := serverConfigChanged(origServer, newCfg.Server); len(changed) > 0 {
				logger.Info("server config changed (%s); restart required to apply",
					strings.Join(changed, ", "))
			}

			// Atomically swap in the new router — routes, hosts, upstreams applied live.
			newGroups := buildHostGroups(newCfg)
			rt.Store(router.New(newGroups))
			logger.Info("config reloaded")
			return nil
		},
		func(e error) { logger.Error("config reload failed: %v", e) },
		func(e error) { logger.Error("watcher: %v", e) },
	)
	if err == nil {
		sm.Register(w.Close)
		if err := w.Start(); err != nil {
			logger.Error("watcher: %v", err)
		}
	}

	sm.Wait()
	shutdownServers(servers)
}

// serverConfigChanged returns the names of any server config fields that differ
// between a (original) and b (new). listen_ports is compared after sorting.
func serverConfigChanged(a, b config.ServerConfig) []string {
	var changed []string

	aPorts := make([]int, len(a.ListenPorts))
	copy(aPorts, a.ListenPorts)
	bPorts := make([]int, len(b.ListenPorts))
	copy(bPorts, b.ListenPorts)
	sort.Ints(aPorts)
	sort.Ints(bPorts)
	if !intsEqual(aPorts, bPorts) {
		changed = append(changed, "listen_ports")
	}

	if a.RedirectHTTP != b.RedirectHTTP {
		changed = append(changed, "redirect_http")
	}

	if tlsConfigChanged(a.TLS, b.TLS) {
		changed = append(changed, "tls")
	}

	return changed
}

func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func tlsConfigChanged(a, b *config.TLSConfig) bool {
	if (a == nil) != (b == nil) {
		return true
	}
	if a == nil {
		return false
	}
	return a.Enabled != b.Enabled || a.CertFile != b.CertFile || a.KeyFile != b.KeyFile
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
// connection settings.
func resolveUpstream(rc config.RouteConfig, upstreams map[string]config.UpstreamConfig) (upstreamURL string, rewriteHost bool, insecure bool) {
	if strings.Contains(rc.Upstream, "://") {
		// Inline URL: RewriteHost is a *bool; dereference or default to false.
		rh := false
		if rc.RewriteHost != nil {
			rh = *rc.RewriteHost
		}
		return rc.Upstream, rh, rc.Insecure
	}
	up, ok := upstreams[rc.Upstream]
	if !ok {
		log.Fatalf("[dev-proxy] unknown upstream %q (should have been caught at startup)", rc.Upstream)
	}
	return up.URL, up.RewriteHost, up.Insecure
}

// buildMatchedRoute converts a config RouteConfig into a router.MatchedRoute.
// hostUpstream is the host-group-level default upstream; it is used when
// rc.Upstream is empty (route inherits from its host group).
// hostRewriteHost is the host-group-level default rewrite_host; it is used
// for inline-upstream routes that omit their own rewrite_host field.
// Named-upstream routes take rewrite_host from the upstream definition instead.
// hostCORSAllowOrigin is the host-group-level default cors_allow_origin; it is
// used when the route omits its own cors_allow_origin field.
func buildMatchedRoute(i int, rc config.RouteConfig, hostUpstream string, hostRewriteHost *bool, hostCORSAllowOrigin string, upstreams map[string]config.UpstreamConfig) router.MatchedRoute {
	// Resolve effective CORS origin: route-level wins, host-level is fallback.
	effectiveCORSOrigin := rc.CORSAllowOrigin
	if effectiveCORSOrigin == "" {
		effectiveCORSOrigin = hostCORSAllowOrigin
	}

	corsCfg := &router.CORSConfig{}
	if effectiveCORSOrigin != "" {
		corsCfg.Enabled = true
		corsCfg.AllowOrigin = effectiveCORSOrigin
	}

	// Resolve effective upstream: route-level wins, host-level is fallback.
	effectiveRC := rc
	if effectiveRC.Upstream == "" {
		effectiveRC.Upstream = hostUpstream
	}

	// Resolve effective rewrite_host for inline-upstream routes:
	//   route-level ptr → host-level ptr → false.
	// Named-upstream routes get rewrite_host from the upstream definition
	// (handled inside resolveUpstream), so we only apply inheritance here.
	if rc.RewriteHost == nil {
		effectiveRewriteHost := false
		if hostRewriteHost != nil {
			effectiveRewriteHost = *hostRewriteHost
		}
		effectiveRC.RewriteHost = &effectiveRewriteHost
	}

	upstreamURL, rewriteHost, insecure := resolveUpstream(effectiveRC, upstreams)

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
func buildHostGroups(cfg *config.Config) []router.HostGroup {
	if len(cfg.Hosts) > 0 {
		var groups []router.HostGroup
		for _, hg := range cfg.Hosts {
			var routes []router.MatchedRoute
			for i, rc := range hg.Routes {
				routes = append(routes, buildMatchedRoute(i, rc, hg.Upstream, hg.RewriteHost, hg.CORSAllowOrigin, cfg.Upstreams))
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
		routes = append(routes, buildMatchedRoute(i, rc, "", nil, "", cfg.Upstreams))
	}
	if len(routes) == 0 {
		routes = append(routes, router.MatchedRoute{
			PathPrefix:  "/",
			RewriteHost: true,
		})
	}
	return []router.HostGroup{{Match: "*", Routes: routes}}
}

func startServers(cfg *config.Config, rt *atomic.Pointer[router.Router], cm *devtls.CertManager, sm *shutdown.Manager) map[int]*http.Server {
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
			var cerr error

			if cfg.Server.TLS != nil && cfg.Server.TLS.CertFile != "" {
				cp, cerr = cm.LoadFromDisk(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
				if cerr != nil {
					log.Fatalf("[dev-proxy] Failed to load certificate for port %d: %v", port, cerr)
				}
			} else {
				cp, cerr = cm.GetOrGenerate("server")
				if cerr != nil {
					log.Fatalf("[dev-proxy] Failed to generate self-signed certificate for port %d: %v", port, cerr)
				}
			}
			tlsCfg := &tls.Config{Certificates: []tls.Certificate{}}
			cert, cerr := tls.X509KeyPair(cp.CertPEM, cp.KeyPEM)
			if cerr != nil {
				log.Fatalf("[dev-proxy] Failed to parse certificate for port %d: %v", port, cerr)
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

// computeUpstreamURL reconstructs the URL that the proxy Director will forward to.
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

// routeCriterion returns a compact tag for the most-specific active path criterion.
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

func buildHandler(rt *atomic.Pointer[router.Router], serverCfg *config.ServerConfig, isTLS bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// Load the current router atomically — safe under concurrent reloads.
		result := rt.Load().Match(r)

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
