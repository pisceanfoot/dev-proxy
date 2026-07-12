package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
)

// RouteConfig represents a single proxy route configuration.
type RouteConfig struct {
	LocalPort   int
	PathPrefix  string
	HostMatch   string
	Upstream    string
	RewriteHost bool
	TLSEnabled  bool
	Insecure    bool
	CORSAllowOrigin string
	StaticDir     string
}

// Config holds all application configuration.
type Config struct {
	Port      int
	Upstream  string
	StaticDir string
	Insecure  bool
	Routes    []RouteConfig
	EnvFile   string
}

// Load reads .env file, applies CLI flags (which override), and validates.
func Load() (*Config, error) {
	envPath := ".env"
	if p := os.Getenv("DEV_PROXY_ENV_FILE"); p != "" {
		envPath = p
	}

	envVars, err := ParseEnv(envPath)
	if err != nil {
		return nil, fmt.Errorf("parse .env: %w", err)
	}

	// CLI flags (these override env vars)
	var (
		portFlag      int
		upstreamFlag  string
		staticDirFlag string
		insecureFlag  bool
		envFileFlag   string
		routesJSON    string
	)

	flag.IntVar(&portFlag, "port", 0, "Local port to listen on")
	flag.StringVar(&upstreamFlag, "upstream", "", "Upstream host (e.g. http://localhost:3000)")
	flag.StringVar(&staticDirFlag, "static-dir", "", "Static file directory to override upstream responses")
	flag.BoolVar(&insecureFlag, "insecure", false, "Skip TLS verification for upstream HTTPS")
	flag.StringVar(&envFileFlag, "env-file", "", "Path to .env file (default: .env in cwd)")
	flag.StringVar(&routesJSON, "routes", "", "JSON array of route configs (optional)")

	flag.Parse()

	// Apply precedence: CLI flags > env vars > defaults
	config := &Config{
		Port:      portFlag,
		Upstream:  upstreamFlag,
		StaticDir: staticDirFlag,
		Insecure:  insecureFlag,
		EnvFile:   envPath,
	}

	if config.Port == 0 {
		if p := envVars["DEV_PROXY_PORT"]; p != "" {
			port, err := strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("invalid DEV_PROXY_PORT %q: %w", p, err)
			}
			config.Port = port
		} else {
			config.Port = 8080 // default
		}
	}

	if config.Upstream == "" {
		if u := envVars["DEV_PROXY_UPSTREAM"]; u != "" {
			config.Upstream = u
		}
	}

	if config.StaticDir == "" {
		if sd := envVars["DEV_PROXY_STATIC_DIR"]; sd != "" {
			config.StaticDir = sd
		}
	}

	// Parse additional routes from JSON if provided
	if routesJSON != "" {
		// TODO: parse JSON routes
	}

	// Build single-route config from port/upstream/static
	if config.Upstream != "" || config.StaticDir != "" {
		config.Routes = append(config.Routes, RouteConfig{
			LocalPort:   config.Port,
			PathPrefix:  "/",
			Upstream:    config.Upstream,
			StaticDir:   config.StaticDir,
			Insecure:    config.Insecure,
			RewriteHost: true,
		})
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Port)
	}

	for i, r := range c.Routes {
		if r.LocalPort < 1 || r.LocalPort > 65535 {
			return fmt.Errorf("route %d: invalid port: %d", i, r.LocalPort)
		}
		if r.Upstream != "" {
			if _, err := url.Parse(r.Upstream); err != nil {
				return fmt.Errorf("route %d: invalid upstream URL %q: %w", i, r.Upstream, err)
			}
		}
	}

	return nil
}
