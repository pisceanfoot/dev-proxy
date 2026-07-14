package watcher

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches one or more files for changes and triggers a reload callback.
type Watcher struct {
	fsWatcher   *fsnotify.Watcher
	filePaths   []string
	reloadFunc  func() error
	onReloadErr func(error) // called when reloadFunc returns an error
	onFSErr     func(error) // called when fsnotify reports an internal error
	mu          sync.Mutex
}

// New creates a new Watcher that monitors all paths in filePaths.
// Paths that do not exist at the time Start is called are silently skipped —
// this allows an optional .env file to be listed without requiring it to exist.
// onReloadErr is called (if non-nil) when reloadFunc returns an error.
// onFSErr is called (if non-nil) when the underlying fsnotify watcher reports an error.
func New(filePaths []string, reloadFunc func() error, onReloadErr func(error), onFSErr func(error)) (*Watcher, error) {
	w := &Watcher{
		filePaths:   filePaths,
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

// Start begins watching all configured file paths for changes.
// Paths that do not exist are silently skipped (no error).
func (w *Watcher) Start() error {
	if len(w.filePaths) == 0 {
		return fmt.Errorf("no file paths to watch")
	}

	watched := 0
	for _, path := range w.filePaths {
		if path == "" {
			continue
		}
		if err := w.fsWatcher.Add(path); err != nil {
			// Skip missing files silently — .env may not exist.
			continue
		}
		watched++
	}

	if watched == 0 {
		// Nothing to watch — still valid (e.g. all paths missing), but log-worthy.
		// Caller decides whether to treat this as an error; we return nil.
		return nil
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
