package sandbox

import (
	"bytes"
	"context"
	"runtime"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	config := &Config{
		Timeout:     5 * time.Second,
		MemoryLimit: 100 * 1024 * 1024,
		CPULimit:    50,
	}

	sb, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sb.Cleanup()

	if sb == nil {
		t.Fatal("Expected non-nil sandbox")
	}
}

func TestSimpleCommand(t *testing.T) {
	config := &Config{
		Timeout:     5 * time.Second,
		MemoryLimit: 100 * 1024 * 1024,
	}

	sb, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sb.Cleanup()

	result, err := sb.Run(context.Background(), "echo", "test")
	if err != nil && result.Error == nil {
		t.Logf("Warning during execution: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.TimedOut {
		t.Error("Command should not timeout")
	}
}

func TestTimeout(t *testing.T) {
	config := &Config{
		Timeout:     1 * time.Second,
		MemoryLimit: 100 * 1024 * 1024,
	}

	sb, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sb.Cleanup()

	result, err := sb.Run(context.Background(), "sleep", "10")
	if err != nil && result.Error == nil {
		t.Logf("Expected error or timeout: %v", err)
	}

	if !result.TimedOut {
		t.Error("Command should have timed out")
	}

	if result.ExecutionTime < 1*time.Second {
		t.Errorf("Execution time should be at least 1 second, got %v", result.ExecutionTime)
	}
}

func TestIORedirection(t *testing.T) {
	var stdout, stderr bytes.Buffer

	config := &Config{
		Timeout:     5 * time.Second,
		MemoryLimit: 100 * 1024 * 1024,
		Stdout:      &stdout,
		Stderr:      &stderr,
	}

	sb, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sb.Cleanup()

	var cmd string
	var args []string

	// Platform-specific command
	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/C", "echo stdout test && echo stderr test 1>&2"}
	} else {
		cmd = "sh"
		args = []string{"-c", "echo 'stdout test' && echo 'stderr test' >&2"}
	}

	result, err := sb.Run(context.Background(), cmd, args...)
	if err != nil && result.Error == nil {
		t.Logf("Warning during execution: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if stdout.Len() == 0 {
		t.Error("Expected stdout output")
	}

	if stderr.Len() == 0 {
		t.Error("Expected stderr output")
	}
}

func TestWorkingDirectory(t *testing.T) {
	config := &Config{
		Timeout:     5 * time.Second,
		MemoryLimit: 100 * 1024 * 1024,
		WorkingDir:  "/tmp",
	}

	sb, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sb.Cleanup()

	var stdout bytes.Buffer
	config.Stdout = &stdout

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "cd"
	} else {
		cmd = "pwd"
	}

	result, err := sb.Run(context.Background(), cmd)
	if err != nil && result.Error == nil {
		t.Logf("Warning during execution: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
}

func TestEnvironmentVariables(t *testing.T) {
	var stdout bytes.Buffer

	config := &Config{
		Timeout:     5 * time.Second,
		MemoryLimit: 100 * 1024 * 1024,
		Stdout:      &stdout,
		Env:         []string{"TEST_VAR=test_value"},
	}

	sb, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sb.Cleanup()

	var cmd string
	var args []string

	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/C", "echo %TEST_VAR%"}
	} else {
		cmd = "sh"
		args = []string{"-c", "echo $TEST_VAR"}
	}

	result, err := sb.Run(context.Background(), cmd, args...)
	if err != nil && result.Error == nil {
		t.Logf("Warning during execution: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
}

func TestCleanup(t *testing.T) {
	config := &Config{
		Timeout:     5 * time.Second,
		MemoryLimit: 100 * 1024 * 1024,
	}

	sb, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	err = sb.Cleanup()
	if err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}
}

func TestNilConfig(t *testing.T) {
	sb, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create sandbox with nil config: %v", err)
	}
	defer sb.Cleanup()

	if sb == nil {
		t.Fatal("Expected non-nil sandbox")
	}
}

func TestExitCode(t *testing.T) {
	config := &Config{
		Timeout:     5 * time.Second,
		MemoryLimit: 100 * 1024 * 1024,
	}

	sb, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sb.Cleanup()

	var cmd string
	var args []string

	if runtime.GOOS == "windows" {
		cmd = "cmd"
		args = []string{"/C", "exit 42"}
	} else {
		cmd = "sh"
		args = []string{"-c", "exit 42"}
	}

	result, err := sb.Run(context.Background(), cmd, args...)
	// Exit code 42 is expected, not an error in our context
	if err != nil && result.Error == nil {
		t.Logf("Note: non-zero exit code produces error: %v", err)
	}

	if result.ExitCode != 42 {
		t.Errorf("Expected exit code 42, got %d", result.ExitCode)
	}
}
