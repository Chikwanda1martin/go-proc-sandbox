package sandbox

// New creates a new sandbox instance for the current OS
// The implementation is platform-specific and defined in factory_*.go files
func New(config *Config) (Sandbox, error) {
	return newPlatformSandbox(config)
}
