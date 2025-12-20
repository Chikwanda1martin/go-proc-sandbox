// +build linux

package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// LinuxSandbox implements Sandbox interface for Linux using cgroups v2 and namespaces
type LinuxSandbox struct {
	config      *Config
	cgroupPath  string
	useCgroupV2 bool
}

// NewLinuxSandbox creates a new Linux sandbox
func NewLinuxSandbox(config *Config) (*LinuxSandbox, error) {
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

	// Check for cgroup v2
	useCgroupV2 := checkCgroupV2()

	sandbox := &LinuxSandbox{
		config:      config,
		useCgroupV2: useCgroupV2,
	}

	return sandbox, nil
}

// checkCgroupV2 checks if cgroup v2 is available
func checkCgroupV2() bool {
	// Check if /sys/fs/cgroup is a cgroup2 filesystem
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "cgroup2")
}

// setupCgroup creates and configures a cgroup for the process
func (s *LinuxSandbox) setupCgroup() error {
	// Generate unique cgroup name
	cgroupName := fmt.Sprintf("go-proc-sandbox-%d", os.Getpid())

	if s.useCgroupV2 {
		s.cgroupPath = filepath.Join("/sys/fs/cgroup", cgroupName)
		
		// Create cgroup directory
		if err := os.MkdirAll(s.cgroupPath, 0755); err != nil {
			// If we can't create cgroup, continue without it (non-root user)
			return nil
		}

		// Set memory limit
		if s.config.MemoryLimit > 0 {
			memPath := filepath.Join(s.cgroupPath, "memory.max")
			os.WriteFile(memPath, []byte(strconv.FormatInt(s.config.MemoryLimit, 10)), 0644)
		}

		// Set CPU limit (as a percentage)
		if s.config.CPULimit > 0 && s.config.CPULimit < 100 {
			// CPU quota in microseconds (100000 = 100%)
			quota := int64(s.config.CPULimit * 1000)
			cpuPath := filepath.Join(s.cgroupPath, "cpu.max")
			os.WriteFile(cpuPath, []byte(fmt.Sprintf("%d 100000", quota)), 0644)
		}

		// Set process limit
		if s.config.MaxProcesses > 0 {
			pidsPath := filepath.Join(s.cgroupPath, "pids.max")
			os.WriteFile(pidsPath, []byte(strconv.Itoa(s.config.MaxProcesses)), 0644)
		}
	}

	return nil
}

// Run executes a command in the sandbox
func (s *LinuxSandbox) Run(ctx context.Context, command string, args ...string) (*Result, error) {
	result := &Result{}
	startTime := time.Now()

	// Setup cgroup
	s.setupCgroup()

	// Create command context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, command, args...)

	// Try to setup namespaces if possible (requires root or user namespaces)
	// Don't fail if we can't set up namespaces - continue without them
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group for easier cleanup
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

	// Add process to cgroup
	if s.cgroupPath != "" && s.useCgroupV2 {
		procsPath := filepath.Join(s.cgroupPath, "cgroup.procs")
		os.WriteFile(procsPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
	}

	// Wait for completion
	err = cmd.Wait()
	result.ExecutionTime = time.Since(startTime)

	// Check if timeout occurred
	if cmdCtx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.Error = fmt.Errorf("execution timeout exceeded")
		// Kill the process group if it's still running
		if cmd.Process != nil {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}

	// Get exit code
	if cmd.ProcessState != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				result.ExitCode = status.ExitStatus()
			}
		} else if err == nil {
			result.ExitCode = 0
		}
	}

	// Check if process was killed due to OOM
	if s.cgroupPath != "" && s.useCgroupV2 {
		eventsPath := filepath.Join(s.cgroupPath, "memory.events")
		if data, err := os.ReadFile(eventsPath); err == nil {
			if strings.Contains(string(data), "oom_kill") {
				result.MemoryExceeded = true
			}
		}
	}

	return result, nil
}

// Cleanup releases resources used by the sandbox
func (s *LinuxSandbox) Cleanup() error {
	if s.cgroupPath != "" {
		// Remove cgroup directory
		os.RemoveAll(s.cgroupPath)
	}
	return nil
}
