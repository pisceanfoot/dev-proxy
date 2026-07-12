package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
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
		log.Fatalf("Failed to load configuration: %v", err)
	}

	cm := devtls.NewCertManager()
	sm := shutdown.New(5 * time.Second)

	routes := buildRoutes(cfg, cm)
	rt := router.New(routes)

	servers := startServers(routes, rt, cm, sm)

	w, err := watcher.New(cfg.EnvFile, func() error {
		newCfg, err := config.Load()
		if err != nil {
			return err
		}
		newRoutes := buildRoutes(newCfg, cm)
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

	printBanner(routes)

	sm.Wait()
	shutdownServers(servers)
}

func buildRoutes(cfg *config.Config, cm *devtls.CertManager) []router.MatchedRoute {
	var routes []router.MatchedRoute
	for _, rc := range cfg.Routes {
		corsCfg := &router.CORSConfig{}
		if rc.CORSAllowOrigin != "" {
			corsCfg.Enabled = true
			corsCfg.AllowOrigin = rc.CORSAllowOrigin
		}

		routes = append(routes, router.MatchedRoute{
			LocalPort:   rc.LocalPort,
			PathPrefix:  rc.PathPrefix,
			HostMatch:   rc.HostMatch,
			Upstream:    rc.Upstream,
			RewriteHost: rc.RewriteHost,
			CORS:        corsCfg,
			StaticDir:   rc.StaticDir,
			TLSEnabled:  rc.TLSEnabled,
			Insecure:    rc.Insecure,
		})
	}
	return routes
}

func startServers(routes []router.MatchedRoute, rt *router.Router, cm *devtls.CertManager, sm *shutdown.Manager) map[int]*http.Server {
	servers := make(map[int]*http.Server)

	for _, route := range routes {
		handler := buildHandler(route, rt)

		addr := fmt.Sprintf(":%d", route.LocalPort)
		var srv *http.Server

		if route.TLSEnabled {
			cp, err := cm.GetOrGenerate(fmt.Sprintf("%d", route.LocalPort))
			if err != nil {
				log.Fatalf("Failed to generate certificate for port %d: %v", route.LocalPort, err)
			}
			tlsCfg := &tls.Config{
				Certificates: []tls.Certificate{},
			}
			// Parse PEM into tls.Certificate
			cert, err := tls.X509KeyPair(cp.CertPEM, cp.KeyPEM)
			if err != nil {
				log.Fatalf("Failed to parse certificate for port %d: %v", route.LocalPort, err)
			}
			tlsCfg.Certificates = append(tlsCfg.Certificates, cert)

			srv = &http.Server{
				Addr:      addr,
				Handler:   handler,
				TLSConfig: tlsCfg,
			}
			go func() {
				fmt.Printf("[dev-proxy] Listening on https://localhost%s\n", addr)
				if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
					log.Fatalf("HTTPS server error: %v", err)
				}
			}()
		} else {
			srv = &http.Server{
				Addr:    addr,
				Handler: handler,
			}
			go func() {
				fmt.Printf("[dev-proxy] Listening on http://localhost%s\n", addr)
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatalf("HTTP server error: %v", err)
				}
			}()
		}

		servers[route.LocalPort] = srv
		sm.Register(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return srv.Shutdown(ctx)
		})
	}

	return servers
}

func buildHandler(route router.MatchedRoute, rt *router.Router) http.Handler {
	var handler http.Handler

	// Reverse proxy handler
	if route.Upstream != "" {
		rp, err := proxy.NewReverseProxy(route.Upstream, route.RewriteHost, route.Insecure)
		if err != nil {
			log.Fatalf("Failed to create reverse proxy for upstream %s: %v", route.Upstream, err)
		}
		handler = rp
	} else if route.StaticDir != "" {
		handler = http.NotFoundHandler()
	}

	// Wrap with static file serving (if configured)
	if route.StaticDir != "" {
		staticHandler := static.Serve(route.StaticDir, handler)
		handler = staticHandler
	}

	// Wrap with CORS middleware (if configured)
	if route.CORS != nil && route.CORS.Enabled {
		corsHandler := cors.Middleware(handler, route.CORS)
		handler = corsHandler
	}

	// Wrap with logging middleware
	handler = loggingMiddleware(handler, &route)

	return handler
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

func printBanner(routes []router.MatchedRoute) {
	fmt.Println()
	fmt.Println("  ┌─────────────────────────────────────┐")
	fmt.Println("  │         dev-proxy v0.1.0            │")
	fmt.Println("  ├─────────────────────────────────────┤")
	for _, r := range routes {
		scheme := "http"
		if r.TLSEnabled {
			scheme = "https"
		}
		target := r.Upstream
		if target == "" {
			target = "static:" + r.StaticDir
		}
		fmt.Printf("  │  %s://localhost:%d%s → %s\n", scheme, r.LocalPort, r.PathPrefix, target)
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
