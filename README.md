# slogscope

![Go Report Card](https://goreportcard.com/badge/github.com/apperia-de/slogscope)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/apperia-de/slogscope?style=flat)
![Coverage](assets/coverage-badge.svg)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/apperia-de/slogscope.svg)](https://pkg.go.dev/github.com/apperia-de/slogscope)

`slogscope` is a custom `slog` Handler for Go, designed to allow fine-grained control over log levels on a per-package
basis. This package provides an easy way to define different logging levels for individual packages, making it simpler
to manage and debug large applications with varying levels of verbosity.

## Features

- **Automatically reflect Config changes:** Making live adjustments easier than ever before on a per-package basis.
- **Package-Specific Log Levels:** Assign different log levels to specific packages within your Go application.
- **Seamless Integration with slog:** Built on top of Go's standard `slog` package, ensuring compatibility and ease of
  use.
- **Flexible and Configurable:** Configure the log levels per package dynamically, without needing to change global
  settings.
- **Optimized for Large Applications:** Ideal for projects with many modules where selective logging is crucial.

## Installation

To install `slogscope`, simply run:

```bash
go get github.com/apperia-de/slogscope
```

## Usage

### Default values

By default, without any `slogscope.Handleroptions` set, the package will use the default ConfigFile (`./slogscope.yml`).
```go 
// These are the default options if no HandlerOptions are specified.
defaultFilename := "slogscope.yml"
opts := &slogscope.HandlerOptions{
  EnableFileWatcher: false,
  Config:            nil,
  ConfigFile:        &defaultFilename,
})
```

To let `slogscope` watch for changes of your config file, add this `slogscope.HandlerOptions`:
```go
// If you want the config to be watched for changes, you have to set the following option to true. 
opts := &slogscope.HandlerOptions{
    EnableFileWatcher: true,
})
```

### Examples

#### Creating a default slogscope.Handler with default settings. 

```go
package main

import (
  "github.com/apperia-de/slogscope"
  "log/slog"
  "os"
)

func main() {
  // Example setup for custom log levels per package using the default settings.
  wrappedHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
  handler := slogscope.NewHandler(wrappedHandler, nil)
  logger := slog.New(handler)
  slog.SetDefault(logger)

  // Use it anywhere else in your code ...  
  slog.Info("Hello World!")
}

```
#### Using default config file (`slogscope.yml`) with enabled file watcher.

```go
package main

import (
  "github.com/apperia-de/slogscope"
  "log/slog"
  "os"
)

func main() {
  // Example setup for custom log levels per package using the default settings.
  wrappedHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
  handler := slogscope.NewHandler(wrappedHandler, &slogscope.HandlerOptions{
	  EnableFileWatcher: true,
  })
  logger := slog.New(handler)
  logger.Info("Hello World!")
}

```

#### Custom `slogscope.HandlerOptions` using `slogscope.Config` directly without config file.

This example uses a given `slogscope.Config` directly instead of loading it from a config file.

```go
package main

import (
	"github.com/apperia-de/slogscope"
	"log/slog"
	"os"
)

func main() {
	// Create a custom Config which will be used instead of the default one provided by ConfigFile.
	cfg := slogscope.Config{
		LogLevel: "INFO",
		Packages: []slogscope.Package{
			{
				Name:     "PACKAGE_NAME_YOU_WANT_TO_OVERRIDE_DEFAULT_LOG_LEVEL",
				LogLevel: "DEBUG",
			},
		},
	}

	// Example setup for custom log levels per package, with custom slogscope.HandlerOptions.
	handler := slogscope.NewHandler(slog.NewTextHandler(os.Stdout, nil), &slogscope.HandlerOptions{
		EnableFileWatcher: false,
		Config:            &cfg,
		ConfigFile:        nil,
	})
	logger := slog.New(handler)
	logger.Info("Hello World!")
}

```
> Hint: If you use a Config directly you may chage it programmatically anytime with `slogscope.Handler.UseConfig(cfg slogscope.Config)`.


## Configuration

The default configuration uses the `slogscope.yml` in your project root directory for package wise log level
configuration.
Altering the config file during runtime causes the logger to reflect those changes across all packages.

You can configure `slogscope` to map different log levels to specific packages using either the `slogscope.yml` config
file method, or passing the current `slogscope.Config` via `Handler.SetConfig(cfg slogscope.Config)`. The default
behavior is to inherit the global log level if no package-specific level is set.

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests to help improve `slogscope`.

### Related Projects

- [awesome-slog](https://github.com/go-slog/awesome-slog): Collection of log/slog related projects.

## Acknowledgments

This project was inspired by a [blog post](https://www.dolthub.com/blog/2024-09-13-package-scoped-logging-in-go-log4j/)
from [@zachmu](https://github.com/zachmu) and my own need for this feature.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
