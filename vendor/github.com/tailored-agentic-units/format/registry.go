package format

import (
	"fmt"
	"sync"
)

// Factory is a constructor function that creates a new Format instance.
type Factory func() (Format, error)

type registry struct {
	factories map[string]Factory
	mu        sync.RWMutex
}

var register = &registry{
	factories: make(map[string]Factory),
}

// Register adds a format factory to the global registry under the given name.
func Register(name string, factory Factory) {
	register.mu.Lock()
	defer register.mu.Unlock()
	register.factories[name] = factory
}

// Create looks up a format by name in the global registry and returns a new instance.
func Create(name string) (Format, error) {
	register.mu.RLock()
	factory, exists := register.factories[name]
	register.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown format: %s", name)
	}

	return factory()
}

// ListFormats returns the names of all registered formats.
func ListFormats() []string {
	register.mu.RLock()
	defer register.mu.RUnlock()

	names := make([]string, 0, len(register.factories))
	for name := range register.factories {
		names = append(names, name)
	}
	return names
}
