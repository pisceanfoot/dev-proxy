package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- parseDotEnv ---

func TestParseDotEnv_PlainKeyValue(t *testing.T) {
	f := writeTempEnv(t, "KEY=value\n")
	env, err := parseDotEnv(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["KEY"] != "value" {
		t.Fatalf("want KEY=value, got %q", env["KEY"])
	}
}

func TestParseDotEnv_DoubleQuotedValue(t *testing.T) {
	f := writeTempEnv(t, `KEY="value with spaces"`+"\n")
	env, err := parseDotEnv(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["KEY"] != "value with spaces" {
		t.Fatalf("want 'value with spaces', got %q", env["KEY"])
	}
}

func TestParseDotEnv_SingleQuotedValue(t *testing.T) {
	f := writeTempEnv(t, "KEY='hello world'\n")
	env, err := parseDotEnv(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["KEY"] != "hello world" {
		t.Fatalf("want 'hello world', got %q", env["KEY"])
	}
}

func TestParseDotEnv_ExportPrefix(t *testing.T) {
	f := writeTempEnv(t, "export KEY=exported\n")
	env, err := parseDotEnv(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["KEY"] != "exported" {
		t.Fatalf("want KEY=exported, got %q", env["KEY"])
	}
}

func TestParseDotEnv_CommentLinesSkipped(t *testing.T) {
	f := writeTempEnv(t, "# this is a comment\nKEY=value\n")
	env, err := parseDotEnv(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := env["# this is a comment"]; ok {
		t.Fatal("comment line should not produce a key")
	}
	if env["KEY"] != "value" {
		t.Fatalf("want KEY=value, got %q", env["KEY"])
	}
}

func TestParseDotEnv_EmptyLinesSkipped(t *testing.T) {
	f := writeTempEnv(t, "\n\nKEY=val\n\n")
	env, err := parseDotEnv(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env) != 1 {
		t.Fatalf("want 1 entry, got %d", len(env))
	}
}

func TestParseDotEnv_EmptyValue(t *testing.T) {
	f := writeTempEnv(t, "KEY=\n")
	env, err := parseDotEnv(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v, ok := env["KEY"]; !ok || v != "" {
		t.Fatalf("want KEY='', got ok=%v val=%q", ok, v)
	}
}

func TestParseDotEnv_MissingFileReturnsEmptyMap(t *testing.T) {
	env, err := parseDotEnv("/nonexistent/path/.env")
	if err != nil {
		t.Fatalf("want no error for missing file, got: %v", err)
	}
	if len(env) != 0 {
		t.Fatalf("want empty map, got %v", env)
	}
}

// --- expandVars ---

func TestExpandVars_SimpleVar(t *testing.T) {
	env := map[string]string{"MY_URL": "http://upstream:8080"}
	out, err := expandVars([]byte("url: ${MY_URL}"), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "url: http://upstream:8080" {
		t.Fatalf("want expanded URL, got %q", string(out))
	}
}

func TestExpandVars_DefaultUsedWhenUnset(t *testing.T) {
	env := map[string]string{}
	out, err := expandVars([]byte("url: ${MISSING_VAR:-http://localhost:3000}"), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "url: http://localhost:3000" {
		t.Fatalf("want default, got %q", string(out))
	}
}

func TestExpandVars_EnvVarBeatsDefault(t *testing.T) {
	env := map[string]string{"MY_URL": "http://real-upstream:9090"}
	out, err := expandVars([]byte("url: ${MY_URL:-http://localhost:3000}"), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "url: http://real-upstream:9090" {
		t.Fatalf("want env value, got %q", string(out))
	}
}

func TestExpandVars_OSEnvUsedForUnknownKey(t *testing.T) {
	t.Setenv("TEST_EXPAND_OS_KEY", "from-os")
	env := map[string]string{} // key not in merged map
	out, err := expandVars([]byte("val: ${TEST_EXPAND_OS_KEY}"), env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "val: from-os" {
		t.Fatalf("want OS env value, got %q", string(out))
	}
}

func TestExpandVars_MissingVarNoDefaultIsError(t *testing.T) {
	env := map[string]string{}
	_, err := expandVars([]byte("url: ${TOTALLY_MISSING}"), env)
	if err == nil {
		t.Fatal("want error for missing var with no default, got nil")
	}
	if !strings.Contains(err.Error(), "TOTALLY_MISSING") {
		t.Fatalf("want var name in error, got: %v", err)
	}
}

func TestExpandVars_NoTokensUnchanged(t *testing.T) {
	input := []byte("url: http://localhost:8080\npath_prefix: /api")
	out, err := expandVars(input, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != string(input) {
		t.Fatalf("want unchanged output, got %q", string(out))
	}
}

// --- mergeEnv ---

func TestMergeEnv_OSEnvOverridesDotEnv(t *testing.T) {
	t.Setenv("MERGE_TEST_KEY", "from-os")
	dotEnv := map[string]string{"MERGE_TEST_KEY": "from-dotenv"}
	merged := mergeEnv(dotEnv)
	if merged["MERGE_TEST_KEY"] != "from-os" {
		t.Fatalf("want OS env to win, got %q", merged["MERGE_TEST_KEY"])
	}
}

func TestMergeEnv_DotEnvUsedWhenOSMissing(t *testing.T) {
	os.Unsetenv("MERGE_ONLY_DOTENV_KEY")
	dotEnv := map[string]string{"MERGE_ONLY_DOTENV_KEY": "from-dotenv"}
	merged := mergeEnv(dotEnv)
	if merged["MERGE_ONLY_DOTENV_KEY"] != "from-dotenv" {
		t.Fatalf("want dotenv value, got %q", merged["MERGE_ONLY_DOTENV_KEY"])
	}
}

// --- config.Load with env interpolation ---

func TestLoad_WithEnvFileInterpolation(t *testing.T) {
	// Write a minimal valid config using interpolation.
	cfgContent := `
server:
  listen_ports: [8080]
upstreams:
  backend:
    url: ${TEST_LOAD_UPSTREAM:-http://localhost:9999}
hosts:
  - match: "*"
    upstream: backend
    routes:
      - path_prefix: /
`
	envContent := "TEST_LOAD_UPSTREAM=http://env-upstream:1234\n"
	cfgPath := writeTempFile(t, "config-*.yaml", cfgContent)
	envPath := writeTempFile(t, ".env-*", envContent)

	cfg, err := Load(cfgPath, envPath)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.Upstreams["backend"].URL != "http://env-upstream:1234" {
		t.Fatalf("want env-upstream URL, got %q", cfg.Upstreams["backend"].URL)
	}
}

func TestLoad_WithMissingEnvFileFallbackToDefault(t *testing.T) {
	cfgContent := `
server:
  listen_ports: [8080]
upstreams:
  backend:
    url: ${TEST_LOAD_FALLBACK_UPSTREAM:-http://fallback:7777}
hosts:
  - match: "*"
    upstream: backend
    routes:
      - path_prefix: /
`
	cfgPath := writeTempFile(t, "config-*.yaml", cfgContent)

	cfg, err := Load(cfgPath, "/nonexistent/.env")
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.Upstreams["backend"].URL != "http://fallback:7777" {
		t.Fatalf("want fallback URL, got %q", cfg.Upstreams["backend"].URL)
	}
}

// --- helpers ---

func writeTempEnv(t *testing.T, content string) string {
	t.Helper()
	return writeTempFile(t, ".env-*", content)
}

func writeTempFile(t *testing.T, pattern, content string) string {
	t.Helper()
	dir := t.TempDir()
	f, err := os.CreateTemp(dir, filepath.Base(pattern))
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}
