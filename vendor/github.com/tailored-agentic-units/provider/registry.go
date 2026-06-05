package provider

import (
	"fmt"
	"sync"

	"github.com/tailored-agentic-units/protocol/config"
)

// Factory is a function that creates a Provider from configuration.
// Provider implementations register their factory function to enable dynamic provider creation.
type Factory func(c *config.ProviderConfig) (Provider, error)

// registry maintains the global provider factory registry.
// It is thread-safe for concurrent registration and provider creation.
type registry struct {
	factories map[string]Factory
	mu        sync.RWMutex
}

// register is the global provider factory registry.
var register = &registry{
	factories: make(map[string]Factory),
}

// Register registers a provider factory with the given name.
// Thread-safe for concurrent registration.
func Register(name string, factory Factory) {
	register.mu.Lock()
	defer register.mu.Unlock()
	register.factories[name] = factory
}

// Create creates a Provider instance from configuration.
// Returns an error if the provider name is not registered.
// Thread-safe for concurrent provider creation.
func Create(c *config.ProviderConfig) (Provider, error) {
	register.mu.RLock()
	factory, exists := register.factories[c.Name]
	register.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown provider: %s", c.Name)
	}

	return factory(c)
}

// ListProviders returns a list of all registered provider names.
// Thread-safe for concurrent access.
func ListProviders() []string {
	register.mu.RLock()
	defer register.mu.RUnlock()

	names := make([]string, 0, len(register.factories))
	for name := range register.factories {
		names = append(names, name)
	}
	return names
}
