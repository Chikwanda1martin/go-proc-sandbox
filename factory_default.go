// +build darwin !linux,!windows

package sandbox

func newPlatformSandbox(config *Config) (Sandbox, error) {
	return NewDefaultSandbox(config)
}
