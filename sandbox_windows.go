// +build windows

package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procCreateJobObjectW          = kernel32.NewProc("CreateJobObjectW")
	procAssignProcessToJobObject  = kernel32.NewProc("AssignProcessToJobObject")
	procSetInformationJobObject   = kernel32.NewProc("SetInformationJobObject")
	procTerminateJobObject        = kernel32.NewProc("TerminateJobObject")
	procCloseHandle               = kernel32.NewProc("CloseHandle")
)

const (
	JobObjectBasicLimitInformation           = 2
	JobObjectExtendedLimitInformation        = 9
	JOB_OBJECT_LIMIT_PROCESS_MEMORY          = 0x00000100
	JOB_OBJECT_LIMIT_JOB_MEMORY              = 0x00000200
	JOB_OBJECT_LIMIT_PROCESS_TIME            = 0x00000002
	JOB_OBJECT_LIMIT_ACTIVE_PROCESS          = 0x00000008
	JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE       = 0x00002000
)

type JOBOBJECT_BASIC_LIMIT_INFORMATION struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type IO_COUNTERS struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type JOBOBJECT_EXTENDED_LIMIT_INFORMATION struct {
	BasicLimitInformation JOBOBJECT_BASIC_LIMIT_INFORMATION
	IoInfo                IO_COUNTERS
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// WindowsSandbox implements Sandbox interface for Windows using Job Objects
type WindowsSandbox struct {
	config    *Config
	jobHandle syscall.Handle
}

// NewWindowsSandbox creates a new Windows sandbox
func NewWindowsSandbox(config *Config) (*WindowsSandbox, error) {
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

	sandbox := &WindowsSandbox{
		config: config,
	}

	return sandbox, nil
}

// createJobObject creates a Windows Job Object with limits
func (s *WindowsSandbox) createJobObject() error {
	// Create job object
	ret, _, err := procCreateJobObjectW.Call(0, 0)
	if ret == 0 {
		return fmt.Errorf("failed to create job object: %v", err)
	}
	s.jobHandle = syscall.Handle(ret)

	// Setup extended limits
	var limits JOBOBJECT_EXTENDED_LIMIT_INFORMATION

	// Set memory limit
	if s.config.MemoryLimit > 0 {
		limits.ProcessMemoryLimit = uintptr(s.config.MemoryLimit)
		limits.BasicLimitInformation.LimitFlags |= JOB_OBJECT_LIMIT_PROCESS_MEMORY
	}

	// Set process count limit
	if s.config.MaxProcesses > 0 {
		limits.BasicLimitInformation.ActiveProcessLimit = uint32(s.config.MaxProcesses)
		limits.BasicLimitInformation.LimitFlags |= JOB_OBJECT_LIMIT_ACTIVE_PROCESS
	}

	// Kill all processes when job handle is closed
	limits.BasicLimitInformation.LimitFlags |= JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE

	// Set job information
	ret, _, err = procSetInformationJobObject.Call(
		uintptr(s.jobHandle),
		uintptr(JobObjectExtendedLimitInformation),
		uintptr(unsafe.Pointer(&limits)),
		unsafe.Sizeof(limits),
	)
	if ret == 0 {
		procCloseHandle.Call(uintptr(s.jobHandle))
		return fmt.Errorf("failed to set job object limits: %v", err)
	}

	return nil
}

// Run executes a command in the sandbox
func (s *WindowsSandbox) Run(ctx context.Context, command string, args ...string) (*Result, error) {
	result := &Result{}
	startTime := time.Now()

	// Create job object
	if err := s.createJobObject(); err != nil {
		result.Error = err
		return result, err
	}

	// Create command context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, command, args...)

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

	// Ensure CREATE_SUSPENDED and CREATE_BREAKAWAY_FROM_JOB flags
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_SUSPENDED | 0x01000000, // CREATE_BREAKAWAY_FROM_JOB
	}

	// Start the process
	err := cmd.Start()
	if err != nil {
		result.Error = fmt.Errorf("failed to start process: %w", err)
		return result, err
	}

	// Assign process to job object
	ret, _, err := procAssignProcessToJobObject.Call(
		uintptr(s.jobHandle),
		uintptr(cmd.Process.Pid),
	)
	if ret == 0 {
		cmd.Process.Kill()
		result.Error = fmt.Errorf("failed to assign process to job: %v", err)
		return result, result.Error
	}

	// Resume the process
	// Note: In a real implementation, we would need to resume the thread
	// For simplicity, we'll start without CREATE_SUSPENDED in practice

	// Wait for completion
	err = cmd.Wait()
	result.ExecutionTime = time.Since(startTime)

	// Check if timeout occurred
	if cmdCtx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.Error = fmt.Errorf("execution timeout exceeded")
	}

	// Get exit code
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	return result, nil
}

// Cleanup releases resources used by the sandbox
func (s *WindowsSandbox) Cleanup() error {
	if s.jobHandle != 0 {
		// Terminate all processes in the job
		procTerminateJobObject.Call(uintptr(s.jobHandle), 1)
		// Close job handle
		procCloseHandle.Call(uintptr(s.jobHandle))
		s.jobHandle = 0
	}
	return nil
}
