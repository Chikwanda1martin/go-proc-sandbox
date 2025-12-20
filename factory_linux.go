// +build linux

package sandbox

func newPlatformSandbox(config *Config) (Sandbox, error) {
	return NewLinuxSandbox(config)
}
