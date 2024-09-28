package slogscope

import (
	"context"
	"log/slog"
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

func (h *handler) GetConfig() Config {
	return *h.opts.Config
}

// UseConfig takes a new Config and immediately applies it to the current configuration.
// It also disables any active file watcher.
func (h *handler) UseConfig(cfg Config) {
	h.mu.Lock()
	h.opts.EnableFileWatcher = false
	h.opts.Config = &cfg
	h.opts.ConfigFile = nil
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
