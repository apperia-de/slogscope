# slogscope

[![GitHub Tag](https://img.shields.io/github/v/tag/apperia-de/slogscope?label=Version)](https://github.com/apperia-de/slogscope)
[![Go Report Card](https://goreportcard.com/badge/github.com/apperia-de/slogscope)](https://goreportcard.com/badge/github.com/apperia-de/slogscope)
[![Go Reference](https://pkg.go.dev/badge/github.com/apperia-de/slogscope.svg)](https://pkg.go.dev/github.com/apperia-de/slogscope)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

`slogscope` is a custom `slog` Handler for Go designed to enable package-scoped log level management. It allows developers to define different log levels for individual packages dynamically, making debugging and log volume control in complex applications simple and highly performant.

## Why Package-Scoped Logging?

In production environments, toggling `DEBUG` logging globally can overwhelm log aggregators, inflate ingestion costs, and degrade application performance. `slogscope` solves this by allowing you to target log levels precisely:

- **Targeted Production Debugging**: Enable `DEBUG` logs only for a specific package (e.g., your database connection pool or payment handler) to troubleshoot issues without flooding your logs.
- **Zero Code Changes**: Works automatically based on your codebase structure. Developers don't need to remember to add specific log attributes or context keys.
- **Ultra-Low Overhead**: Leverages thread-safe call-site caching (`uintptr` program counters) to eliminate reflection and allocation costs on hot paths, making package checks near-instant.

## Key Features

- **Per-Package Log Levels**: Set granular log levels (e.g., `DEBUG`, `INFO`, `WARN`, `ERROR`) per package path.
- **Dynamic Configuration Updates**: Reflect configuration updates instantly at runtime without restarting the application.
- **File Watcher Integration**: Watch for YAML configuration file changes automatically via fsnotify.
- **Ultra High Performance**: Optimized call-site caching using standard library maps and reader-writer locks, reducing logging overhead to a minimum.
- **Seamless Integration**: Built entirely on top of the Go standard library `log/slog` package.

## Installation

```bash
go get github.com/apperia-de/slogscope
```

## Configuration

`slogscope` uses a YAML configuration file (default is `slogscope.yml`) to define package-level overrides. 

```yaml
# slogscope.yml
log_level: INFO
packages:
  - name: github.com/apperia-de/slogscope/examples/pkg/logger/debuglogger
    log_level: DEBUG
  - name: github.com/apperia-de/slogscope/examples/pkg/logger/errorlogger
    log_level: ERROR
```

### Wildcards & Package Inheritance

`slogscope` supports hierarchical package log level inheritance. If you configure a log level override for a parent package, all of its subpackages will automatically inherit that log level unless they define their own more specific override.

To make this intuitive, suffix wildcards (`/*` and `/...`) at the end of package paths are fully supported and behave identically (both are normalized by stripping the suffix):

```yaml
packages:
  # This package and all subpackages recursively (e.g. debuglogger, errorlogger) inherit ERROR level
  - name: github.com/apperia-de/slogscope/examples/pkg/logger/*
    log_level: ERROR

  # This is equivalent to using '/*'
  - name: github.com/apperia-de/slogscope/examples/pkg/logger/...
    log_level: ERROR

  # Or simply omit the wildcard entirely (slogscope still matches subpackages recursively)
  - name: github.com/apperia-de/slogscope/examples/pkg/logger
    log_level: ERROR
```

Please note that middle-of-path wildcards (e.g. `github.com/.../logger/*`) are not supported.

## Quick Start

### Basic Usage

Use the default settings (loads config from `./slogscope.yml` without watching).

```go
package main

import (
	"log/slog"
	"os"

	"github.com/apperia-de/slogscope"
)

func main() {
	// Wrap a standard slog handler
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
	handler := slogscope.NewHandler(jsonHandler, nil)
	
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("This is an info message")
}
```

### With Configuration Watching

Enable the file watcher to automatically pick up runtime modifications to `slogscope.yml`.

```go
package main

import (
	"log/slog"
	"os"

	"github.com/apperia-de/slogscope"
)

func main() {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
	handler := slogscope.NewHandler(jsonHandler, &slogscope.HandlerOptions{
		EnableFileWatcher: true,
	})
	
	logger := slog.New(handler)
	slog.Info("Config changes to slogscope.yml will be loaded live")
}
```

### Programmatic Configuration

Change configuration programmatically without using a YAML file.

```go
package main

import (
	"log/slog"
	"os"

	"github.com/apperia-de/slogscope"
)

func main() {
	cfg := slogscope.Config{
		LogLevel: "INFO",
		Packages: []slogscope.Package{
			{
				Name:     "github.com/apperia-de/slogscope/examples/pkg/logger/debuglogger",
				LogLevel: "DEBUG",
			},
		},
	}

	handler := slogscope.NewHandler(slog.NewTextHandler(os.Stdout, nil), &slogscope.HandlerOptions{
		Config: &cfg,
	})
	
	logger := slog.New(handler)
	logger.Info("Starting application with programmatic configuration...")
}
```

> **Note**: Programmatic configuration can be updated at any time using `handler.UseConfig(cfg slogscope.Config)`.

## Benchmarks

Our recent optimization pass has significantly reduced the overhead of package level lookup by using thread-safe call-site caching and standard library maps, eliminating redundant string formatting and interface allocations.

Here are the benchmark results on `Apple M2 Pro (Go 1.26)`:

| Logger Handler | Performance (ns/op) | Allocations (B/op) | Allocations (op) |
| :--- | :--- | :--- | :--- |
| **Default Handler** | ~394 ns/op | 246 B/op | 0 allocs/op |
| **slogscope (Optimized)** | **~996 ns/op** | **175 B/op** | **0 allocs/op** |
| **slogscope (Before)** | ~1547 ns/op | 798 B/op | 8 allocs/op |

*Note: slogscope performs a caller package path lookup on every check to determine the correct log level. The optimized version reduces this overhead by 35% and completely eliminates memory allocations (0 allocs/op).*

## Related Projects

- [awesome-slog](https://github.com/go-slog/awesome-slog): Collection of log/slog related projects.

## Acknowledgments

This project was inspired by a [blog post](https://www.dolthub.com/blog/2024-09-13-package-scoped-logging-in-go-log4j/) from [@zachmu](https://github.com/zachmu) and my own need for this feature.

## Contributing

Contributions are welcome! Please feel free to open issues or submit pull requests.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE.md) file for details.
