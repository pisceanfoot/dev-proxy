package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"dev-proxy/internal/config"
	"dev-proxy/internal/cors"
	devtls "dev-proxy/internal/tls"
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

	cm := devtls.NewCertManager()
	sm := shutdown.New(5 * time.Second)

	routes := buildRoutes(cfg)
	rt := router.New(routes)

	servers := startServers(cfg, routes, rt, cm, sm)

	w, err := watcher.New(cfg.ConfigPath, func() error {
		newCfg, err := config.Load()
		if err != nil {
			return err
		}
		newRoutes := buildRoutes(newCfg)
		rt = router.New(newRoutes)
		fmt.Println("[dev-proxy] Routes updated")
		return nil
	})
	if err == nil {
		sm.Register(w.Close)
		if err := w.Start(); err != nil {
			fmt.Printf("[dev-proxy] Watcher error: %v\n", err)
		}
	}

	printBanner(cfg, routes)

	sm.Wait()
	shutdownServers(servers)
}

func buildRoutes(cfg *config.Config) []router.MatchedRoute {
	var matched []router.MatchedRoute
	for _, rc := range cfg.Routes {
		corsCfg := &router.CORSConfig{}
		if rc.CORSAllowOrigin != "" {
			corsCfg.Enabled = true
			corsCfg.AllowOrigin = rc.CORSAllowOrigin
		}

		matched = append(matched, router.MatchedRoute{
			PathPrefix:  rc.PathPrefix,
			HostMatch:   rc.HostMatch,
			Upstream:    rc.Upstream,
			RewriteHost: rc.RewriteHost,
			CORS:        corsCfg,
			StaticDir:   rc.StaticDir,
			Insecure:    rc.Insecure,
		})
	}

	if len(matched) == 0 {
		matched = append(matched, router.MatchedRoute{
			PathPrefix: "/",
			RewriteHost: true,
		})
	}

	return matched
}

func matchRoutesToPorts(routes []router.MatchedRoute, ports []int) map[int][]router.MatchedRoute {
	result := make(map[int][]router.MatchedRoute)
	for _, port := range ports {
		for _, r := range routes {
			mr := r
			mr.LocalPort = port
			result[port] = append(result[port], mr)
		}
	}
	return result
}

func startServers(cfg *config.Config, matchedRoutes []router.MatchedRoute, rt *router.Router, cm *devtls.CertManager, sm *shutdown.Manager) map[int]*http.Server {
	portRoutes := matchRoutesToPorts(matchedRoutes, cfg.Server.ListenPorts)
	servers := make(map[int]*http.Server)

	for _, port := range cfg.Server.ListenPorts {
		isTLS := isPortTLSEnabled(port, cfg)
		handler := buildHandler(portRoutes[port], rt, &cfg.Server, isTLS)

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
			tlsCfg := &tls.Config{
				Certificates: []tls.Certificate{},
			}
			cert, err := tls.X509KeyPair(cp.CertPEM, cp.KeyPEM)
			if err != nil {
				log.Fatalf("[dev-proxy] Failed to parse certificate for port %d: %v", port, err)
			}
			tlsCfg.Certificates = append(tlsCfg.Certificates, cert)
			srv.TLSConfig = tlsCfg

			go func(p int) {
				fmt.Printf("[dev-proxy] Listening on https://localhost%s\n", addr)
				if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
					handlePortError(p, err)
				}
			}(port)
		} else {
			go func(p int) {
				fmt.Printf("[dev-proxy] Listening on http://localhost%s\n", addr)
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
	// TLS is applied to the highest port in listen_ports.
	// With [80, 443]: 443 gets TLS. With [8443] alone: 8443 gets TLS.
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

func buildHandler(portRoutes []router.MatchedRoute, rt *router.Router, serverCfg *config.ServerConfig, isTLS bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := rt.Match(r)
		if route == nil {
			router.HandleNotFound(w)
			return
		}

		var handler http.Handler

		if route.Upstream != "" {
			rp, err := proxy.NewReverseProxy(route.Upstream, route.RewriteHost, route.Insecure)
			if err != nil {
				log.Fatalf("Failed to create reverse proxy for upstream %s: %v", route.Upstream, err)
			}
			handler = rp
		} else if route.StaticDir != "" {
			handler = http.NotFoundHandler()
		}

		if handler == nil {
			router.HandleNotFound(w)
			return
		}

		if route.StaticDir != "" {
			staticHandler := static.Serve(route.StaticDir, handler)
			handler = staticHandler
		}

		if route.CORS != nil && route.CORS.Enabled {
			corsHandler := cors.Middleware(handler, route.CORS)
			handler = corsHandler
		}

		if serverCfg.RedirectHTTP && !isTLS {
			targetPort := findTLSPort(serverCfg.ListenPorts)
			if targetPort > 0 {
				handler = redirectMiddleware(handler, targetPort)
			}
		}

		loggingMiddleware(handler, route).ServeHTTP(w, r)
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

func loggingMiddleware(next http.Handler, route *router.MatchedRoute) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lw, r)
		duration := time.Since(start)

		upstream := route.Upstream
		if upstream == "" {
			upstream = "static:" + route.StaticDir
		}

		fmt.Printf("[%s] %s %s → %s (%d) %v\n",
			r.Method, r.URL.Path, r.Host, upstream, lw.statusCode, duration)
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

func printBanner(cfg *config.Config, routes []router.MatchedRoute) {
	fmt.Println()
	fmt.Println("  ┌─────────────────────────────────────┐")
	fmt.Println("  │         dev-proxy v0.1.0            │")
	fmt.Println("  ├─────────────────────────────────────┤")

	for _, port := range cfg.Server.ListenPorts {
		scheme := "http"
		if isPortTLSEnabled(port, cfg) {
			scheme = "https"
		}
		for _, r := range routes {
			target := r.Upstream
			if target == "" {
				target = "static:" + r.StaticDir
			}
			fmt.Printf("  │  %s://localhost:%d%s → %s\n", scheme, port, r.PathPrefix, target)
		}
	}

	fmt.Println("  └─────────────────────────────────────┘")
	fmt.Println()

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Done()
}

func shutdownServers(servers map[int]*http.Server) {
	for port, srv := range servers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			fmt.Printf("[dev-proxy] Error shutting down server on port %d: %v\n", port, err)
		}
		fmt.Printf("[dev-proxy] Server on port %d stopped\n", port)
	}
}
