package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Manager handles graceful shutdown on SIGINT/SIGTERM.
type Manager struct {
	mu          sync.Mutex
	cancel      context.CancelFunc
	ctx         context.Context
	timeout     time.Duration
	onShutdown  []func() error
}

// New creates a new Shutdown manager with the given timeout for draining in-flight requests.
func New(timeout time.Duration) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		cancel:  cancel,
		ctx:     ctx,
		timeout: timeout,
	}
}

// Register adds a cleanup function to be called during shutdown.
func (m *Manager) Register(fn func() error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onShutdown = append(m.onShutdown, fn)
}

// Wait blocks until a SIGINT or SIGTERM signal is received.
func (m *Manager) Wait() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\n[dev-proxy] Shutting down...")

	m.cancel()
}

// DoShutdown runs all registered cleanup functions sequentially.
func (m *Manager) DoShutdown() error {
	var lastErr error
	for _, fn := range m.onShutdown {
		if err := fn(); err != nil {
			fmt.Printf("[dev-proxy] shutdown cleanup error: %v\n", err)
			lastErr = err
		}
	}
	return lastErr
}
