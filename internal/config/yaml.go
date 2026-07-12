package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"

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

// RouteConfig is a pure proxy rule — no port or TLS fields.
type RouteConfig struct {
	PathPrefix    string `yaml:"path_prefix"`
	HostMatch     string `yaml:"host_match"`
	Upstream      string `yaml:"upstream"`
	RewriteHost   bool   `yaml:"rewrite_host"`
	CORSAllowOrigin string `yaml:"cors_allow_origin"`
	StaticDir     string `yaml:"static_dir"`
	Insecure      bool   `yaml:"insecure"`
}

// Config is the top-level YAML configuration.
type Config struct {
	Server     ServerConfig    `yaml:"server"`
	Routes     []RouteConfig   `yaml:"routes"`
	ConfigPath string          `yaml:"-"`
}

// Load reads dev-proxy.yaml and applies CLI flags (which override).
func Load() (*Config, error) {
	configPath := "dev-proxy.yaml"
	if p := os.Getenv("DEV_PROXY_CONFIG"); p != "" {
		configPath = p
	}

	var configFlag string
	flag.StringVar(&configFlag, "config", "", "Path to dev-proxy YAML config file (default: dev-proxy.yaml in cwd)")
	flag.Parse()

	if configFlag != "" {
		configPath = configFlag
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		return nil, fmt.Errorf("read config file %s: %w", configPath, err)
	}

	cfg := &Config{ConfigPath: configPath}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", configPath, err)
	}

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func validate(cfg *Config) error {
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

	for i, r := range cfg.Routes {
		if r.PathPrefix == "" {
			cfg.Routes[i].PathPrefix = "/"
		}
		if r.Upstream != "" {
			if _, err := url.Parse(r.Upstream); err != nil {
				return fmt.Errorf("route %d: invalid upstream URL %q: %w", i, r.Upstream, err)
			}
		}
	}

	return nil
}
