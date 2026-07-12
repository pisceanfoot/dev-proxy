package watcher

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a .env file for changes and triggers a reload callback.
type Watcher struct {
	fsWatcher  *fsnotify.Watcher
	filePath   string
	reloadFunc func() error
	mu         sync.Mutex
}

// New creates a new Watcher for the given .env file path and reload function.
func New(filePath string, reloadFunc func() error) (*Watcher, error) {
	w := &Watcher{
		filePath:   filePath,
		reloadFunc: reloadFunc,
	}

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}
	w.fsWatcher = fw

	return w, nil
}

// Start begins watching the .env file for changes.
func (w *Watcher) Start() error {
	dir := w.filePath
	if dir == "" {
		dir = ".env"
	}

	err := w.fsWatcher.Add(dir)
	if err != nil {
		return fmt.Errorf("add watch for %s: %w", dir, err)
	}

	go w.watchLoop()
	return nil
}

func (w *Watcher) watchLoop() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				fmt.Println("[dev-proxy] .env file changed, reloading...")
				if err := w.reloadFunc(); err != nil {
					fmt.Printf("[dev-proxy] reload error: %v\n", err)
				} else {
					fmt.Println("[dev-proxy] config reloaded successfully")
				}
			}
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("[dev-proxy] watcher error: %v\n", err)
		}
	}
}

// Close stops the watcher and releases resources.
func (w *Watcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.fsWatcher != nil {
		return w.fsWatcher.Close()
	}
	return nil
}
