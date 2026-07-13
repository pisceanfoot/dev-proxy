package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// TLSConfig holds the server-level TLS certificate file paths.
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// ServerConfig owns port bindings, TLS termination, and HTTP-to-HTTPS redirect.
type ServerConfig struct {
	ListenPorts  []int      `yaml:"listen_ports"`
	TLS          *TLSConfig `yaml:"tls"`
	RedirectHTTP bool       `yaml:"redirect_http"`
}

// UpstreamConfig defines a reusable named upstream target.
type UpstreamConfig struct {
	URL         string `yaml:"url"`
	RewriteHost bool   `yaml:"rewrite_host"`
	Insecure    bool   `yaml:"insecure"`
}

// RouteConfig is a pure proxy rule — no port or TLS fields.
// RewriteHost is a pointer so that omitting the field in YAML ("not set / inherit
// from host group") is distinguishable from explicitly writing rewrite_host: false.
type RouteConfig struct {
	PathPrefix      string `yaml:"path_prefix"`
	PathExact       string `yaml:"path_exact"`
	PathRegex       string `yaml:"path_regex"`
	HostMatch       string `yaml:"host_match"`
	Upstream        string `yaml:"upstream"`
	UpstreamPath    string `yaml:"upstream_path"`
	RewriteHost     *bool  `yaml:"rewrite_host"`
	CORSAllowOrigin string `yaml:"cors_allow_origin"`
	StaticDir       string `yaml:"static_dir"`
	Insecure        bool   `yaml:"insecure"`
}

// HostGroup groups routes under a host match pattern.
// Entries are evaluated in declaration order — first match wins.
// The optional Upstream field provides a default upstream for routes that
// do not specify their own; route-level upstream always takes precedence.
// The optional RewriteHost field provides a default rewrite_host setting for
// inline-upstream routes that omit their own; route-level always takes precedence.
type HostGroup struct {
	Match       string        `yaml:"match"`
	Upstream    string        `yaml:"upstream"`
	RewriteHost *bool         `yaml:"rewrite_host"`
	Routes      []RouteConfig `yaml:"routes"`
}

// Config is the top-level YAML configuration.
type Config struct {
	Server     ServerConfig              `yaml:"server"`
	LogLevel   string                    `yaml:"log_level"`
	Upstreams  map[string]UpstreamConfig `yaml:"upstreams"`
	Hosts      []HostGroup               `yaml:"hosts"`
	Routes     []RouteConfig             `yaml:"routes"`
	ConfigPath string                    `yaml:"-"`
}

// Load reads the YAML config from the given path, parses, and validates it.
// The caller is responsible for resolving the path (e.g. from CLI flags or env).
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", path)
		}
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	cfg := &Config{ConfigPath: path}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	// Validate log level.
	if cfg.LogLevel != "" {
		switch strings.ToLower(strings.TrimSpace(cfg.LogLevel)) {
		case "error", "info", "debug":
			// valid
		default:
			return fmt.Errorf("invalid log_level %q — must be one of: error, info, debug", cfg.LogLevel)
		}
	}

	if len(cfg.Server.ListenPorts) == 0 {
		return fmt.Errorf("server.listen_ports must have at least one port")
	}

	for _, port := range cfg.Server.ListenPorts {
		if port < 1 || port > 65535 {
			return fmt.Errorf("invalid listen port: %d (must be 1-65535)", port)
		}
	}

	if cfg.Server.TLS != nil && cfg.Server.TLS.CertFile != "" {
		if _, err := os.Stat(cfg.Server.TLS.CertFile); os.IsNotExist(err) {
			return fmt.Errorf("TLS cert_file does not exist: %s", cfg.Server.TLS.CertFile)
		}
		if cfg.Server.TLS.KeyFile == "" {
			return fmt.Errorf("tls.key_file is required when tls.cert_file is set")
		}
		if _, err := os.Stat(cfg.Server.TLS.KeyFile); os.IsNotExist(err) {
			return fmt.Errorf("TLS key_file does not exist: %s", cfg.Server.TLS.KeyFile)
		}
	}

	// Validate named upstreams.
	for name, up := range cfg.Upstreams {
		if name == "" {
			return fmt.Errorf("upstream name must not be empty")
		}
		if up.URL == "" {
			return fmt.Errorf("upstream %q: url must not be empty", name)
		}
		if _, err := url.Parse(up.URL); err != nil {
			return fmt.Errorf("upstream %q: invalid url %q: %w", name, up.URL, err)
		}
	}

	// Validate host groups.
	for i, hg := range cfg.Hosts {
		if hg.Match == "" {
			return fmt.Errorf("hosts[%d]: match must not be empty", i)
		}
		if len(hg.Routes) == 0 {
			return fmt.Errorf("hosts[%d] (match=%q): routes must not be empty", i, hg.Match)
		}
		// Validate the host-level upstream if present.
		if hg.Upstream != "" {
			if strings.Contains(hg.Upstream, "://") {
				if _, err := url.Parse(hg.Upstream); err != nil {
					return fmt.Errorf("hosts[%d] (match=%q): invalid upstream URL %q: %w", i, hg.Match, hg.Upstream, err)
				}
			} else {
				if _, ok := cfg.Upstreams[hg.Upstream]; !ok {
					return fmt.Errorf("hosts[%d] (match=%q): upstream %q is not defined in the upstreams map", i, hg.Match, hg.Upstream)
				}
			}
		}
		for j, r := range hg.Routes {
			if err := validateRoute(i*1000+j, r, hg.Upstream, cfg.Upstreams); err != nil {
				return fmt.Errorf("hosts[%d].routes[%d]: %w", i, j, err)
			}
		}
	}

	// Warn when both hosts and routes are defined.
	if len(cfg.Hosts) > 0 && len(cfg.Routes) > 0 {
		fmt.Fprintln(os.Stderr, "[dev-proxy] WARNING: both 'hosts' and 'routes' defined; 'routes' is ignored — use 'hosts' only")
	}

	// Validate flat routes.
	for i, r := range cfg.Routes {
		if err := validateRoute(i, r, "", cfg.Upstreams); err != nil {
			return fmt.Errorf("route %d: %w", i, err)
		}
	}

	return nil
}

func validateRoute(idx int, r RouteConfig, hostUpstream string, upstreams map[string]UpstreamConfig) error {
	// Determine the effective upstream: route-level wins, then host-level fallback.
	effective := r.Upstream
	if effective == "" {
		effective = hostUpstream
	}
	if effective == "" {
		return fmt.Errorf("upstream must not be empty (no route-level upstream and no host-level upstream default)")
	}
	if strings.Contains(effective, "://") {
		if _, err := url.Parse(effective); err != nil {
			return fmt.Errorf("invalid upstream URL %q: %w", effective, err)
		}
	} else {
		if _, ok := upstreams[effective]; !ok {
			return fmt.Errorf("upstream %q is not defined in the upstreams map", effective)
		}
	}
	return nil
}
