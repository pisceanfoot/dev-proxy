package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// isAuxFile returns true if the event targets a Vim/Editor auxiliary file.
func isAuxFile(path string) bool {
	base := filepath.Base(path)
	return base == ".DS_Store" ||
		(base != "." && (base[0] == '.' && len(base) > 4 && base[len(base)-4:] == ".swp")) || // .file.swp
		(base != "." && (base[0] == '.' && len(base) > 4 && base[len(base)-4:] == ".swo")) || // .file.swo
		base == "*.bak" ||
		(len(base) > 1 && base[len(base)-1] == '~')
}

type Watcher struct {
	fsWatcher   *fsnotify.Watcher
	filePaths   map[string]bool // map for fast lookup of target files
	reloadFunc  func() error
	onReloadErr func(error)
	onFSErr     func(error)
	mu          sync.Mutex

	// Track last known modification time per watched path to avoid false reloads.
	lastMtime map[string]time.Time
}

func New(filePaths []string, reloadFunc func() error, onReloadErr func(error), onFSErr func(error)) (*Watcher, error) {
	// Clean and absolute-ify files to ensure robust matching
	targets := make(map[string]bool)
	for _, p := range filePaths {
		if abs, err := filepath.Abs(p); err == nil {
			targets[abs] = true
		} else {
			targets[filepath.Clean(p)] = true
		}
	}

	w := &Watcher{
		filePaths:   targets,
		reloadFunc:  reloadFunc,
		onReloadErr: onReloadErr,
		onFSErr:     onFSErr,
		lastMtime:   make(map[string]time.Time),
	}

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}
	w.fsWatcher = fw
	return w, nil
}

func (w *Watcher) Start() error {
	if len(w.filePaths) == 0 {
		return fmt.Errorf("no file paths to watch")
	}

	// Watch the parent directories of our targets instead of the files directly
	watchedDirs := make(map[string]bool)
	for path := range w.filePaths {
		if path == "" || isAuxFile(path) {
			continue
		}
		dir := filepath.Dir(path)
		if !watchedDirs[dir] {
			if err := w.fsWatcher.Add(dir); err != nil {
				continue // skip missing directories silently
			}
			watchedDirs[dir] = true
		}
	}

	go w.watchLoop()
	return nil
}

func (w *Watcher) watchLoop() {
	const debounce = 100 * time.Millisecond
	// Use a channel-based debouncer approach instead of fighting the Timer directly
	var timerChan <-chan time.Time
	var pendingReload bool

	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// 1. Ensure the event targets one of our specific target files
			absEventPath, err := filepath.Abs(event.Name)
			if err != nil {
				absEventPath = filepath.Clean(event.Name)
			}
			if !w.filePaths[absEventPath] {
				continue
			}

			// 2. Skip auxiliary swap/temp files
			if isAuxFile(event.Name) {
				continue
			}

			// 3. Since Vim deletes/re-creates, we want to capture Writes, Creates,
			// or Chmod events (which often happen at the end of a rename sequence)
			isWriteOp := event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Chmod)
			if !isWriteOp {
				continue
			}

			// 4. Verify file exists and modified time changed
			info, err := os.Stat(event.Name)
			if err != nil || info.IsDir() {
				continue
			}

			w.mu.Lock()
			prev, exists := w.lastMtime[absEventPath]
			mtime := info.ModTime()
			changedContent := !exists || !prev.Equal(mtime)
			if changedContent {
				w.lastMtime[absEventPath] = mtime
			}
			w.mu.Unlock()

			// 5. Trigger debounce if the content actually changed
			if changedContent {
				pendingReload = true
				timerChan = time.After(debounce)
			}

		case <-timerChan:
			// Debounce finished - trigger the reload!
			if pendingReload {
				pendingReload = false
				if err := w.reloadFunc(); err != nil && w.onReloadErr != nil {
					w.onReloadErr(err)
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

func (w *Watcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.fsWatcher != nil {
		return w.fsWatcher.Close()
	}
	return nil
}