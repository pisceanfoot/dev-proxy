package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNew_MultiplePathsAccepted verifies the watcher is created successfully
// with a list of file paths.
func TestNew_MultiplePathsAccepted(t *testing.T) {
	w, err := New([]string{"a.yaml", "b.env"}, func() error { return nil }, nil, nil)
	if err != nil {
		t.Fatalf("want no error creating watcher, got: %v", err)
	}
	defer w.Close()
}

// TestStart_NonExistentPathSkippedSilently verifies that a path that doesn't
// exist in the filePaths list does not cause Start to return an error.
func TestStart_NonExistentPathSkippedSilently(t *testing.T) {
	w, err := New([]string{"/nonexistent/path/.env"}, func() error { return nil }, nil, nil)
	if err != nil {
		t.Fatalf("want no error creating watcher, got: %v", err)
	}
	defer w.Close()

	// Start should not error even though the path doesn't exist.
	if err := w.Start(); err != nil {
		t.Fatalf("want no error starting watcher with non-existent path, got: %v", err)
	}
}

// TestStart_ExistingFileTriggersReload verifies that a write to a watched file
// triggers the reload callback.
func TestStart_ExistingFileTriggersReload(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "dev-proxy.yaml")
	if err := os.WriteFile(configPath, []byte("initial"), 0644); err != nil {
		t.Fatalf("create config file: %v", err)
	}

	reloaded := make(chan struct{}, 1)
	w, err := New(
		[]string{configPath},
		func() error {
			select {
			case reloaded <- struct{}{}:
			default:
			}
			return nil
		},
		nil, nil,
	)
	if err != nil {
		t.Fatalf("want no error, got: %v", err)
	}
	defer w.Close()

	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Trigger a write event.
	if err := os.WriteFile(configPath, []byte("updated"), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	select {
	case <-reloaded:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("want reload callback triggered within 2s, timed out")
	}
}

// TestStart_MixedExistingAndMissingPaths verifies the watcher works when some
// paths exist (config) and some don't (.env) — the common startup scenario.
func TestStart_MixedExistingAndMissingPaths(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "dev-proxy.yaml")
	if err := os.WriteFile(configPath, []byte("initial"), 0644); err != nil {
		t.Fatalf("create config file: %v", err)
	}
	missingEnvPath := filepath.Join(dir, ".env") // does not exist

	reloaded := make(chan struct{}, 1)
	w, err := New(
		[]string{configPath, missingEnvPath},
		func() error {
			select {
			case reloaded <- struct{}{}:
			default:
			}
			return nil
		},
		nil, nil,
	)
	if err != nil {
		t.Fatalf("want no error, got: %v", err)
	}
	defer w.Close()

	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Write to the config file — should still trigger reload.
	if err := os.WriteFile(configPath, []byte("updated"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	select {
	case <-reloaded:
		// success — watcher still works despite missing .env
	case <-time.After(2 * time.Second):
		t.Fatal("want reload triggered by config change, timed out")
	}
}
