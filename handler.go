package slogscope

import (
	"context"
	"log/slog"
	"time"
)

type handler struct {
	*slogscope
}

// NewHandler creates a new slog.Handler
func NewHandler(h slog.Handler, opts *HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &HandlerOptions{
			EnableFileWatcher: true,
			Config:            nil,
		}
	}

	if opts.EnableFileWatcher && opts.ConfigFile == nil {
		filename := defaultConfigFile
		opts.ConfigFile = &filename
	}

	// We need to create an instance of the logger for internal logging purposes
	// to prevent nil pointer exception during setup.
	// Will be replaced later with the handler from the given HandlerFunc hFn.
	ssc := &slogscope{}
	defer ssc.initHandler(h, *opts)
	sccHndl := &handler{
		slogscope: ssc,
	}
	sccHndl.h = sccHndl
	return sccHndl
}

func (h *handler) Enabled(_ context.Context, lvl slog.Level) bool {
	globalLevel := h.getLogLevel(h.opts.Config.LogLevel)
	cInfo := getCallerInfo(5)
	if v, ok := h.pkgMap.Load(cInfo.PackageName); ok {
		p := v.(*pkg)
		debugf("use package log level=%q for package=%q", p.logLevel, p.name)
		return lvl >= p.logLevel.Level()
	}
	debugf("use default log level=%q for package=%q", globalLevel, cInfo.PackageName)
	return lvl >= globalLevel
}

func (h *handler) Handle(ctx context.Context, rec slog.Record) error {
	return h.slogh.Handle(ctx, rec)
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.slogh.WithAttrs(attrs)
}

func (h *handler) WithGroup(name string) slog.Handler {
	return h.slogh.WithGroup(name)
}

// GetConfig returns the current configuration, which may be adjusted and then used with UseConfig(cfg Config).
func (h *handler) GetConfig() Config {
	return *h.opts.Config
}

// UseConfig takes a new Config and immediately applies it to the current configuration.
// It also disables any active file watcher.
func (h *handler) UseConfig(cfg Config) {
	h.mu.Lock()
	h.opts.EnableFileWatcher = false
	h.opts.Config = &cfg
	h.mu.Unlock()

	h.initHandler(h.slogh, *h.opts)
}

// UseConfigTemporarily takes a new Config and immediately applies it to the current configuration.
// In contrast to UseConfig(cfg	Config), this function automatically reverts to the state before calling the method,
// after revert amount of time has elapsed.
func (h *handler) UseConfigTemporarily(cfg Config, revert time.Duration) {
	h.mu.Lock()
	oldCfg := h.GetConfig()
	enableFileWatcher := h.opts.EnableFileWatcher

	h.opts.EnableFileWatcher = false
	h.opts.Config = &cfg
	h.mu.Unlock()

	h.initHandler(h.slogh, *h.opts)

	go func(enableFileWatcher bool) {
		<-time.After(revert)
		if enableFileWatcher {
			h.UseConfigFile()
		} else {
			h.UseConfig(oldCfg)
		}
	}(enableFileWatcher)
}

// UseConfigFile takes a filename as an argument that will be used for watching a config file for changes.
// If no such filename is given, the handler uses the already existing ConfigFile from the HandlerOptions or,
// if not present, falls back to the default config file (specified via defaultConfigFile).
func (h *handler) UseConfigFile(cfgFile ...string) {
	h.mu.Lock()
	if len(cfgFile) == 1 && cfgFile[0] != "" {
		h.opts.ConfigFile = &cfgFile[0]
	}

	// If we have zero or more than one config files and the current HandlerOptions
	// doesn't contain a ConfigFile option, we use the default config filename.
	if h.opts.ConfigFile == nil {
		filename := defaultConfigFile
		h.opts.ConfigFile = &filename
	}
	h.opts.EnableFileWatcher = true
	h.mu.Unlock()

	h.initHandler(h.slogh, *h.opts)
}

type nilHandler struct{}

// NewNilHandler provides a nil slog.Handler for silencing slog.Log() calls.
func NewNilHandler() slog.Handler {
	return &nilHandler{}
}

func (h *nilHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

func (h *nilHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (h *nilHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *nilHandler) WithGroup(_ string) slog.Handler {
	return h
}
