package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStderr(fn func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	fn()
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()
	return buf.String()
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input   string
		want    Level
		wantErr bool
	}{
		{"error", LevelError, false},
		{"ERROR", LevelError, false},
		{"Error", LevelError, false},
		{"info", LevelInfo, false},
		{"INFO", LevelInfo, false},
		{"Info", LevelInfo, false},
		{"debug", LevelDebug, false},
		{"DEBUG", LevelDebug, false},
		{"Debug", LevelDebug, false},
		{"", LevelError, false},
		{"  error  ", LevelError, false},
		{"invalid", LevelError, true},
		{"warning", LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseLevel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLogger_Info(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		format   string
		args     []any
		wantOut  bool
		wantText string
	}{
		{"error level blocks info", LevelError, "hello %s", []any{"world"}, false, ""},
		{"info level allows info", LevelInfo, "hello %s", []any{"world"}, true, "[info] hello world"},
		{"debug level allows info", LevelDebug, "hello %s", []any{"world"}, true, "[info] hello world"},
		{"info level no args", LevelInfo, "hello", nil, true, "[info] hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Logger{level: tt.level}
			out := captureStderr(func() {
				l.Info(tt.format, tt.args...)
			})
			if tt.wantOut {
				if !strings.Contains(out, tt.wantText) {
					t.Fatalf("expected output to contain %q, got %q", tt.wantText, out)
				}
			} else {
				if out != "" {
					t.Fatalf("expected no output, got %q", out)
				}
			}
		})
	}
}

func TestLogger_Debug(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		format   string
		args     []any
		wantOut  bool
		wantText string
	}{
		{"error level blocks debug", LevelError, "debug %s", []any{"msg"}, false, ""},
		{"info level blocks debug", LevelInfo, "debug %s", []any{"msg"}, false, ""},
		{"debug level allows debug", LevelDebug, "debug %s", []any{"msg"}, true, "[debug] debug msg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Logger{level: tt.level}
			out := captureStderr(func() {
				l.Debug(tt.format, tt.args...)
			})
			if tt.wantOut {
				if !strings.Contains(out, tt.wantText) {
					t.Fatalf("expected output to contain %q, got %q", tt.wantText, out)
				}
			} else {
				if out != "" {
					t.Fatalf("expected no output, got %q", out)
				}
			}
		})
	}
}

func TestLogger_Error(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		format   string
		args     []any
		wantText string
	}{
		{"error level prints error", LevelError, "err %s", []any{"msg"}, "[error] err msg"},
		{"info level prints error", LevelInfo, "err %s", []any{"msg"}, "[error] err msg"},
		{"debug level prints error", LevelDebug, "err %s", []any{"msg"}, "[error] err msg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Logger{level: tt.level}
			out := captureStderr(func() {
				l.Error(tt.format, tt.args...)
			})
			if !strings.Contains(out, tt.wantText) {
				t.Fatalf("expected output to contain %q, got %q", tt.wantText, out)
			}
		})
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// Save and restore Default level
	origLevel := Default.level
	defer func() { Default.level = origLevel }()

	t.Run("Info at error level", func(t *testing.T) {
		Default.level = LevelError
		out := captureStderr(func() {
			Info("pkg info")
		})
		if out != "" {
			t.Fatalf("expected no output, got %q", out)
		}
	})

	t.Run("Info at info level", func(t *testing.T) {
		Default.level = LevelInfo
		out := captureStderr(func() {
			Info("pkg info")
		})
		if !strings.Contains(out, "[info] pkg info") {
			t.Fatalf("expected info output, got %q", out)
		}
	})

	t.Run("Debug at debug level", func(t *testing.T) {
		Default.level = LevelDebug
		out := captureStderr(func() {
			Debug("pkg debug")
		})
		if !strings.Contains(out, "[debug] pkg debug") {
			t.Fatalf("expected debug output, got %q", out)
		}
	})

	t.Run("Error always prints", func(t *testing.T) {
		Default.level = LevelError
		out := captureStderr(func() {
			Error("pkg error")
		})
		if !strings.Contains(out, "[error] pkg error") {
			t.Fatalf("expected error output, got %q", out)
		}
	})
}

func TestSetLevel(t *testing.T) {
	origLevel := Default.level
	defer func() { Default.level = origLevel }()

	SetLevel(LevelDebug)
	if Default.level != LevelDebug {
		t.Fatalf("expected Default.level = LevelDebug, got %v", Default.level)
	}

	SetLevel(LevelInfo)
	if Default.level != LevelInfo {
		t.Fatalf("expected Default.level = LevelInfo, got %v", Default.level)
	}
}
