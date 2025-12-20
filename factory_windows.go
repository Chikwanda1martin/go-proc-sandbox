// +build windows

package sandbox

func newPlatformSandbox(config *Config) (Sandbox, error) {
	return NewWindowsSandbox(config)
}
