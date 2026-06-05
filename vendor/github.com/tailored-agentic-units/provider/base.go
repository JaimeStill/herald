package provider

// BaseProvider provides common functionality for provider implementations.
// It stores the provider name and base URL, and provides default accessors.
// Provider implementations typically embed BaseProvider to inherit this functionality.
type BaseProvider struct {
	name    string
	baseURL string
}

// NewBaseProvider creates a new BaseProvider with the given name and base URL.
// This is typically called by provider constructors to initialize common fields.
func NewBaseProvider(name, baseURL string) *BaseProvider {
	return &BaseProvider{
		name:    name,
		baseURL: baseURL,
	}
}

// Name returns the provider's identifier.
func (p *BaseProvider) Name() string {
	return p.name
}

// BaseURL returns the provider's base URL.
// Provider implementations use this to construct full endpoint URLs.
func (p *BaseProvider) BaseURL() string {
	return p.baseURL
}
