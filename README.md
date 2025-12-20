# go-proc-sandbox

A Go-based sandbox runner that limits CPU, memory, filesystem access, and execution time for spawned processes. Uses OS-level primitives (cgroups, job objects, namespaces where available) for safe local execution.

## Features

- **Cross-platform**: Supports Linux, Windows, and macOS/Unix systems
- **CPU Limiting**: Control CPU usage percentage per process
- **Memory Limiting**: Set maximum memory usage
- **Execution Timeout**: Automatic termination after timeout
- **Process Limiting**: Limit number of child processes
- **I/O Redirection**: Capture stdout/stderr or provide stdin
- **Environment Control**: Custom environment variables
- **Working Directory**: Set custom working directory

## Platform-Specific Implementation

### Linux
- Uses **cgroups v2** for resource limiting (CPU, memory, process count)
- Supports **namespaces** (mount, PID) for isolation
- OOM killer detection for memory violations

### Windows
- Uses **Job Objects** for resource limiting
- Memory and process count limits enforced
- Automatic cleanup on job closure

### macOS/Unix
- Uses **setrlimit** for resource limiting
- Process group management
- Best-effort resource constraints

## Installation

```bash
go get github.com/BaseMax/go-proc-sandbox
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/BaseMax/go-proc-sandbox"
)

func main() {
    // Create sandbox configuration
    config := &sandbox.Config{
        Timeout:     5 * time.Second,
        MemoryLimit: 100 * 1024 * 1024, // 100 MB
        CPULimit:    50,                 // 50% of one core
        MaxProcesses: 10,
    }

    // Create sandbox instance (auto-detects OS)
    sb, err := sandbox.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer sb.Cleanup()

    // Run a command
    result, err := sb.Run(context.Background(), "echo", "Hello, Sandbox!")
    if err != nil {
        log.Printf("Execution error: %v", err)
    }

    fmt.Printf("Exit Code: %d\n", result.ExitCode)
    fmt.Printf("Execution Time: %v\n", result.ExecutionTime)
    fmt.Printf("Timed Out: %v\n", result.TimedOut)
}
```

## Configuration Options

```go
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
```

## Usage Examples

### Example 1: Timeout Handling

```go
config := &sandbox.Config{
    Timeout: 2 * time.Second,
}

sb, _ := sandbox.New(config)
defer sb.Cleanup()

result, _ := sb.Run(context.Background(), "sleep", "10")
fmt.Printf("Timed Out: %v\n", result.TimedOut) // true
```

### Example 2: Memory Limiting

```go
config := &sandbox.Config{
    MemoryLimit: 50 * 1024 * 1024, // 50 MB
    Timeout:     10 * time.Second,
}

sb, _ := sandbox.New(config)
defer sb.Cleanup()

result, _ := sb.Run(context.Background(), "your-memory-intensive-app")
fmt.Printf("Memory Exceeded: %v\n", result.MemoryExceeded)
```

### Example 3: I/O Redirection

```go
var stdout, stderr bytes.Buffer

config := &sandbox.Config{
    Timeout: 5 * time.Second,
    Stdout:  &stdout,
    Stderr:  &stderr,
}

sb, _ := sandbox.New(config)
defer sb.Cleanup()

sb.Run(context.Background(), "echo", "output")
fmt.Printf("Output: %s\n", stdout.String())
```

### Example 4: Custom Environment

```go
config := &sandbox.Config{
    Timeout: 5 * time.Second,
    Env:     []string{"CUSTOM_VAR=value", "PATH=/usr/bin"},
}

sb, _ := sandbox.New(config)
defer sb.Cleanup()

sb.Run(context.Background(), "printenv", "CUSTOM_VAR")
```

## Result Structure

```go
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
```

## Running Examples

```bash
cd examples/basic
go run main.go
```

## Running Tests

```bash
go test -v
```

## Security Considerations

- This library is designed for **safe local execution**, not as a container replacement
- Root privileges may be required for full isolation on Linux (namespaces, cgroups)
- Without root, the sandbox provides best-effort resource limiting
- Always validate and sanitize command inputs
- Be cautious with filesystem access and network settings

## Limitations

### Linux
- Full cgroup v2 features require root or proper permissions
- Namespace isolation requires appropriate capabilities
- Some features may be limited in containerized environments

### Windows
- Job object features depend on Windows version
- Some limits may not be enforced in all scenarios

### macOS/Unix
- Resource limits are best-effort using setrlimit
- Less strict than Linux cgroups or Windows job objects
- Memory limiting may not prevent all allocations

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Author

Copyright (c) 2024, Max Base
