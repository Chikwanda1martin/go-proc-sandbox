// +build darwin !linux,!windows

package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// DefaultSandbox implements Sandbox interface for macOS and other Unix systems using ulimit
type DefaultSandbox struct {
	config *Config
}

// NewDefaultSandbox creates a new default sandbox
func NewDefaultSandbox(config *Config) (*DefaultSandbox, error) {
	if config == nil {
		config = &Config{}
	}

	// Set defaults
	if config.CPULimit == 0 {
		config.CPULimit = 100
	}
	if config.MemoryLimit == 0 {
		config.MemoryLimit = 512 * 1024 * 1024 // 512 MB default
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxProcesses == 0 {
		config.MaxProcesses = 50
	}

	sandbox := &DefaultSandbox{
		config: config,
	}

	return sandbox, nil
}

// Run executes a command in the sandbox
func (s *DefaultSandbox) Run(ctx context.Context, command string, args ...string) (*Result, error) {
	result := &Result{}
	startTime := time.Now()

	// Create command context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, command, args...)

	// Setup resource limits using setrlimit
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Set working directory
	if s.config.WorkingDir != "" {
		cmd.Dir = s.config.WorkingDir
	}

	// Set environment variables
	if len(s.config.Env) > 0 {
		cmd.Env = s.config.Env
	} else {
		cmd.Env = os.Environ()
	}

	// Setup I/O
	if s.config.Stdin != nil {
		cmd.Stdin = s.config.Stdin
	}
	if s.config.Stdout != nil {
		cmd.Stdout = s.config.Stdout
	}
	if s.config.Stderr != nil {
		cmd.Stderr = s.config.Stderr
	}

	// Start the process
	err := cmd.Start()
	if err != nil {
		result.Error = fmt.Errorf("failed to start process: %w", err)
		return result, err
	}

	// Set resource limits on the process group
	// Note: This is done after start as we need the PID
	if s.config.MemoryLimit > 0 {
		// Set memory limit (data segment)
		rlimit := syscall.Rlimit{
			Cur: uint64(s.config.MemoryLimit),
			Max: uint64(s.config.MemoryLimit),
		}
		// Note: Setting limits on already running process has limited effect
		// This is a best-effort approach
		syscall.Setrlimit(syscall.RLIMIT_DATA, &rlimit)
		syscall.Setrlimit(syscall.RLIMIT_AS, &rlimit)
	}

	if s.config.MaxProcesses > 0 {
		// Set process limit
		rlimit := syscall.Rlimit{
			Cur: uint64(s.config.MaxProcesses),
			Max: uint64(s.config.MaxProcesses),
		}
		syscall.Setrlimit(syscall.RLIMIT_NPROC, &rlimit)
	}

	// Wait for completion
	err = cmd.Wait()
	result.ExecutionTime = time.Since(startTime)

	// Check if timeout occurred
	if cmdCtx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.Error = fmt.Errorf("execution timeout exceeded")
		// Kill the process group
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	// Get exit code
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
		
		// Check if process was killed by signal (possibly OOM)
		if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			if ws.Signaled() && ws.Signal() == syscall.SIGKILL {
				// Could be OOM or timeout
				if !result.TimedOut {
					result.MemoryExceeded = true
				}
			}
		}
	}

	return result, nil
}

// Cleanup releases resources used by the sandbox
func (s *DefaultSandbox) Cleanup() error {
	// No resources to cleanup for this implementation
	return nil
}
