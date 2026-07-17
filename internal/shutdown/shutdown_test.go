package shutdown

import (
	"errors"
	"syscall"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	m := New(5 * time.Second)
	if m == nil {
		t.Fatal("expected Manager, got nil")
	}
	if m.timeout != 5*time.Second {
		t.Fatalf("expected timeout 5s, got %v", m.timeout)
	}
	if m.ctx == nil {
		t.Fatal("expected context to be set")
	}
	if m.cancel == nil {
		t.Fatal("expected cancel to be set")
	}
}

func TestRegister(t *testing.T) {
	m := New(5 * time.Second)

	called := false
	m.Register(func() error {
		called = true
		return nil
	})

	if len(m.onShutdown) != 1 {
		t.Fatalf("expected 1 registered function, got %d", len(m.onShutdown))
	}

	// Verify the function works by calling DoShutdown
	m.DoShutdown()
	if !called {
		t.Fatal("expected registered function to be called")
	}
}

func TestRegister_Multiple(t *testing.T) {
	m := New(5 * time.Second)

	var calls []int
	m.Register(func() error {
		calls = append(calls, 1)
		return nil
	})
	m.Register(func() error {
		calls = append(calls, 2)
		return nil
	})
	m.Register(func() error {
		calls = append(calls, 3)
		return nil
	})

	if len(m.onShutdown) != 3 {
		t.Fatalf("expected 3 registered functions, got %d", len(m.onShutdown))
	}

	m.DoShutdown()

	if len(calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(calls))
	}
	if calls[0] != 1 || calls[1] != 2 || calls[2] != 3 {
		t.Fatalf("expected sequential calls [1,2,3], got %v", calls)
	}
}

func TestDoShutdown_Success(t *testing.T) {
	m := New(5 * time.Second)

	m.Register(func() error {
		return nil
	})

	err := m.DoShutdown()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDoShutdown_Error(t *testing.T) {
	m := New(5 * time.Second)

	expectedErr := errors.New("cleanup failed")
	m.Register(func() error {
		return expectedErr
	})

	err := m.DoShutdown()
	if err != expectedErr {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDoShutdown_MultipleErrors(t *testing.T) {
	m := New(5 * time.Second)

	lastErr := errors.New("last error")
	m.Register(func() error {
		return errors.New("first error")
	})
	m.Register(func() error {
		return lastErr
	})

	err := m.DoShutdown()
	if err != lastErr {
		t.Fatalf("expected last error %v, got %v", lastErr, err)
	}
}

func TestWait_Signal(t *testing.T) {
	m := New(5 * time.Second)

	done := make(chan struct{})
	go func() {
		m.Wait()
		close(done)
	}()

	// Give Wait() time to set up signal handling
	time.Sleep(100 * time.Millisecond)

	// Send SIGTERM to ourselves
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Wait() to return")
	}
}

func TestWait_SIGINT(t *testing.T) {
	m := New(5 * time.Second)

	done := make(chan struct{})
	go func() {
		m.Wait()
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)

	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Wait() to return")
	}
}

func TestContextCancellation(t *testing.T) {
	m := New(5 * time.Second)

	// Verify context is not cancelled initially
	select {
	case <-m.ctx.Done():
		t.Fatal("context should not be cancelled initially")
	default:
		// OK
	}

	// Cancel the context
	m.cancel()

	select {
	case <-m.ctx.Done():
		// OK
	case <-time.After(time.Second):
		t.Fatal("context should be cancelled")
	}
}
