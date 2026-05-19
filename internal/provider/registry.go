package provider

// Factory creates a provider instance from a configuration map.
type Factory func(cfg map[string]any) (any, error)

var registry = map[string]Factory{}

// Register adds a provider factory to the registry.
func Register(name string, factory Factory) {
	registry[name] = factory
}

// Get retrieves a provider factory by name.
func Get(name string) (Factory, bool) {
	f, ok := registry[name]
	return f, ok
}
