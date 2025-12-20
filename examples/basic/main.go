package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/BaseMax/go-proc-sandbox"
)

func main() {
	fmt.Println("Go Process Sandbox Examples")
	fmt.Println("============================")

	// Example 1: Simple command execution
	example1()

	// Example 2: Command with timeout
	example2()

	// Example 3: Command with memory limit
	example3()

	// Example 4: Command with CPU limit
	example4()

	// Example 5: Command with I/O redirection
	example5()
}

func example1() {
	fmt.Println("\n=== Example 1: Simple Command Execution ===")

	config := &sandbox.Config{
		Timeout:     5 * time.Second,
		MemoryLimit: 100 * 1024 * 1024, // 100 MB
		CPULimit:    50,                 // 50% of one core
	}

	sb, err := sandbox.New(config)
	if err != nil {
		log.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sb.Cleanup()

	result, err := sb.Run(context.Background(), "echo", "Hello from sandbox!")
	if err != nil && result.Error == nil {
		log.Printf("Warning: %v", err)
	}

	fmt.Printf("Exit Code: %d\n", result.ExitCode)
	fmt.Printf("Execution Time: %v\n", result.ExecutionTime)
	fmt.Printf("Timed Out: %v\n", result.TimedOut)
}

func example2() {
	fmt.Println("\n=== Example 2: Command with Timeout ===")

	config := &sandbox.Config{
		Timeout:     2 * time.Second,
		MemoryLimit: 100 * 1024 * 1024,
	}

	sb, err := sandbox.New(config)
	if err != nil {
		log.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sb.Cleanup()

	// This command will timeout
	result, err := sb.Run(context.Background(), "sleep", "10")
	if err != nil && result.Error == nil {
		log.Printf("Warning: %v", err)
	}

	fmt.Printf("Exit Code: %d\n", result.ExitCode)
	fmt.Printf("Execution Time: %v\n", result.ExecutionTime)
	fmt.Printf("Timed Out: %v\n", result.TimedOut)
}

func example3() {
	fmt.Println("\n=== Example 3: Command with Memory Limit ===")

	config := &sandbox.Config{
		Timeout:     10 * time.Second,
		MemoryLimit: 10 * 1024 * 1024, // 10 MB - very low limit
		CPULimit:    100,
	}

	sb, err := sandbox.New(config)
	if err != nil {
		log.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sb.Cleanup()

	// Try to run a command - actual memory allocation test would need specific program
	result, err := sb.Run(context.Background(), "echo", "Testing memory limits")
	if err != nil && result.Error == nil {
		log.Printf("Warning: %v", err)
	}

	fmt.Printf("Exit Code: %d\n", result.ExitCode)
	fmt.Printf("Execution Time: %v\n", result.ExecutionTime)
	fmt.Printf("Memory Exceeded: %v\n", result.MemoryExceeded)
}

func example4() {
	fmt.Println("\n=== Example 4: Command with CPU Limit ===")

	config := &sandbox.Config{
		Timeout:     10 * time.Second,
		MemoryLimit: 100 * 1024 * 1024,
		CPULimit:    25, // Limit to 25% of one core
	}

	sb, err := sandbox.New(config)
	if err != nil {
		log.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sb.Cleanup()

	result, err := sb.Run(context.Background(), "echo", "CPU limited execution")
	if err != nil && result.Error == nil {
		log.Printf("Warning: %v", err)
	}

	fmt.Printf("Exit Code: %d\n", result.ExitCode)
	fmt.Printf("Execution Time: %v\n", result.ExecutionTime)
}

func example5() {
	fmt.Println("\n=== Example 5: Command with I/O Redirection ===")

	var stdout, stderr bytes.Buffer

	config := &sandbox.Config{
		Timeout:     5 * time.Second,
		MemoryLimit: 100 * 1024 * 1024,
		Stdout:      &stdout,
		Stderr:      &stderr,
		Env:         append(os.Environ(), "CUSTOM_VAR=sandbox_value"),
	}

	sb, err := sandbox.New(config)
	if err != nil {
		log.Fatalf("Failed to create sandbox: %v", err)
	}
	defer sb.Cleanup()

	result, err := sb.Run(context.Background(), "sh", "-c", "echo 'stdout output' && echo 'stderr output' >&2")
	if err != nil && result.Error == nil {
		log.Printf("Warning: %v", err)
	}

	fmt.Printf("Exit Code: %d\n", result.ExitCode)
	fmt.Printf("Execution Time: %v\n", result.ExecutionTime)
	fmt.Printf("Stdout: %s", stdout.String())
	fmt.Printf("Stderr: %s", stderr.String())
}
