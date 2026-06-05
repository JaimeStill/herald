package observability

import (
	"fmt"
	"log/slog"
	"sync"
)

var (
	observers = map[string]Observer{
		"noop": NoOpObserver{},
		"slog": NewSlogObserver(slog.Default()),
	}
	mutex sync.RWMutex
)

// GetObserver returns a registered observer by name.
// Pre-registered observers: "noop" (NoOpObserver) and "slog" (default logger).
func GetObserver(name string) (Observer, error) {
	mutex.RLock()
	defer mutex.RUnlock()

	obs, exists := observers[name]
	if !exists {
		return nil, fmt.Errorf("unknown observer: %s", name)
	}
	return obs, nil
}

// RegisterObserver adds or replaces a named observer in the global registry.
func RegisterObserver(name string, observer Observer) {
	mutex.Lock()
	defer mutex.Unlock()

	observers[name] = observer
}
