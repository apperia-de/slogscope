# slogscope

[![Go Reference](https://pkg.go.dev/badge/github.com/apperia-de/slogscope.svg)](https://pkg.go.dev/github.com/apperia-de/slogscope)
![Go Report Card](https://goreportcard.com/badge/github.com/apperia-de/slogscope)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/apperia-de/slogscope?style=flat)
![GitHub Licence](https://img.shields.io/github/license/apperia-de/slogscope)

`slogscope` is a custom `slog` Handler for Go, designed to allow fine-grained control over log levels on a per-package
basis. This package provides an easy way to define different logging levels for individual packages, making it simpler
to manage and debug large applications with varying levels of verbosity.

### Features

- **Package-Specific Log Levels:** Assign different log levels to specific packages within your Go application.
- **Seamless Integration with slog:** Built on top of Go'ss standard `slog` package, ensuring compatibility and ease of
  use.
- **Flexible and Configurable:** Configure the log levels per package dynamically, without needing to change global
  settings.
- **Optimized for Large Applications:** Ideal for projects with many modules where selective logging is crucial.
- **Automatically reflect Config changes:** Making live adjustments easier than ever before on a per-package basis.

### Installation

To install `slogscope`, simply run:

```bash
go get github.com/apperia-de/slogscope
```

### Usage

Import `slogscope` in your project and configure the log levels per package as needed:

```go
package main

import (
	"github.com/apperia-de/slogscope"
	"log/slog"
	"os"
)

func main() {
	// Example setup for custom log levels per package.
	Handler := slogscope.NewHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}), nil)
	logger := slog.New(Handler)
	slog.SetDefault(logger)

	slog.Info("Hello World!")
}
```

### Custom `slogscope.HandlerOptions`

This example uses a given `slogscope.Config` directly instead of loading it from a config file.

```go
package main

import (
	"github.com/apperia-de/slogscope"
	"log/slog"
	"os"
)

func main() {
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
	// It uses the given config directly instead from a config file.
	Handler := slogscope.NewHandler(slog.NewTextHandler(os.Stdout, nil), &slogscope.HandlerOptions{
		EnableFileWatcher: false,
		Config:            &cfg,
		ConfigFile:        nil,
	})
	logger := slog.New(Handler)
	logger.Info("Hello World!")
}

```

### Configuration

The default configuration uses the `slogscope.yml` in your project root directory for package wise log level
configuration.
Altering the config file during runtime causes the logger to reflect those changes across all packages.

You can configure `slogscope` to map different log levels to specific packages using either the `slogscope.yml` config
file method, or passing the current `slogscope.Config` via `Handler.SetConfig(cfg slogscope.Config)`. The default
behavior is to inherit the global log level if no package-specific level is set.

### Contributing

Contributions are welcome! Feel free to open issues or submit pull requests to help improve `slogscope`.

### Related Projects

- [awesome-slog](https://github.com/go-slog/awesome-slog): Collection of log/slog related projects.

### Acknowledgments

This project was inspired by a [blog post](https://www.dolthub.com/blog/2024-09-13-package-scoped-logging-in-go-log4j/)
from [@zachmu](https://github.com/zachmu) and my own need for this feature.

### License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
