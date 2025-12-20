package sandbox

import (
	"context"
	"io"
	"time"
)

// Config defines the sandbox configuration
type Config struct {
	// CPU limit in percentage (0-100 per core)
	CPULimit int

	// Memory limit in bytes
	MemoryLimit int64

	// Execution timeout
	Timeout time.Duration

	// Working directory for the process
	WorkingDir string

	// Allowed directories for read access
	AllowedDirs []string

	// Read-only directories
	ReadOnlyDirs []string

	// Environment variables
	Env []string

	// Stdin for the process
	Stdin io.Reader

	// Stdout for the process
	Stdout io.Writer

	// Stderr for the process
	Stderr io.Writer

	// NetworkAccess enables/disables network access
	NetworkAccess bool

	// MaxProcesses limits the number of processes
	MaxProcesses int
}

// Result contains the execution result
type Result struct {
	// Exit code of the process
	ExitCode int

	// Execution time
	ExecutionTime time.Duration

	// Whether the process was killed due to timeout
	TimedOut bool

	// Whether the process exceeded memory limit
	MemoryExceeded bool

	// Error if any
	Error error
}

// Sandbox defines the interface for process sandboxing
type Sandbox interface {
	// Run executes a command in the sandbox
	Run(ctx context.Context, command string, args ...string) (*Result, error)

	// Cleanup releases any resources used by the sandbox
	Cleanup() error
}
