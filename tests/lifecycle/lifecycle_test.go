package lifecycle_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/JaimeStill/herald/pkg/lifecycle"
)

func TestNotReadyBeforeStartup(t *testing.T) {
	lc := lifecycle.New()
	if lc.Ready() {
		t.Error("should not be ready before WaitForStartup")
	}
}

func TestReadyAfterStartup(t *testing.T) {
	lc := lifecycle.New()
	lc.WaitForStartup()

	if !lc.Ready() {
		t.Error("should be ready after WaitForStartup")
	}
}

func TestStartupHooksExecute(t *testing.T) {
	lc := lifecycle.New()

	var count atomic.Int32
	for range 3 {
		lc.OnStartup(func() {
			count.Add(1)
		})
	}

	lc.WaitForStartup()

	if got := count.Load(); got != 3 {
		t.Errorf("startup hooks: got %d, want 3", got)
	}
}

func TestShutdownHooksExecute(t *testing.T) {
	lc := lifecycle.New()

	var cleaned atomic.Bool
	lc.OnShutdown(func() {
		<-lc.Context().Done()
		cleaned.Store(true)
	})

	lc.WaitForStartup()

	if err := lc.Shutdown(5 * time.Second); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	if !cleaned.Load() {
		t.Error("shutdown hook did not execute")
	}
}

func TestShutdownTimeout(t *testing.T) {
	lc := lifecycle.New()

	lc.OnShutdown(func() {
		<-lc.Context().Done()
		time.Sleep(500 * time.Millisecond)
	})

	lc.WaitForStartup()

	err := lc.Shutdown(50 * time.Millisecond)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestContextCancelledOnShutdown(t *testing.T) {
	lc := lifecycle.New()
	lc.WaitForStartup()

	if err := lc.Shutdown(5 * time.Second); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	select {
	case <-lc.Context().Done():
	default:
		t.Error("context should be cancelled after shutdown")
	}
}
