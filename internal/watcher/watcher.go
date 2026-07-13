package watcher

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a config file for changes and triggers a reload callback.
type Watcher struct {
	fsWatcher   *fsnotify.Watcher
	filePath    string
	reloadFunc  func() error
	onReloadErr func(error) // called when reloadFunc returns an error
	onFSErr     func(error) // called when fsnotify reports an internal error
	mu          sync.Mutex
}

// New creates a new Watcher.
// onReloadErr is called (if non-nil) when reloadFunc returns an error.
// onFSErr is called (if non-nil) when the underlying fsnotify watcher reports an error.
func New(filePath string, reloadFunc func() error, onReloadErr func(error), onFSErr func(error)) (*Watcher, error) {
	w := &Watcher{
		filePath:    filePath,
		reloadFunc:  reloadFunc,
		onReloadErr: onReloadErr,
		onFSErr:     onFSErr,
	}

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}
	w.fsWatcher = fw

	return w, nil
}

// Start begins watching the config file for changes.
func (w *Watcher) Start() error {
	path := w.filePath
	if path == "" {
		path = "dev-proxy.yaml"
	}

	if err := w.fsWatcher.Add(path); err != nil {
		return fmt.Errorf("add watch for %s: %w", path, err)
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
				if err := w.reloadFunc(); err != nil {
					if w.onReloadErr != nil {
						w.onReloadErr(err)
					}
				}
			}
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			if w.onFSErr != nil {
				w.onFSErr(err)
			}
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
