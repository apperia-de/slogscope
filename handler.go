package slogscope

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Handler struct {
	*slogscope
}

// NewHandler creates a new slog.Handler
func NewHandler(h slog.Handler, opts *HandlerOptions) *Handler {
	o := HandlerOptions{}
	//	EnableFileWatcher: false,
	//	ConfigFile:        nil,
	//	Config:            nil,
	//	Debug:             false,
	//}

	if opts != nil {
		o = *opts
	}

	//if opts == nil {
	//	defaultOpts = &HandlerOptions{
	//		EnableFileWatcher: true,
	//		ConfigFile:        nil,
	//		Config:            nil,
	//		Debug:             false,
	//	}
	//}

	if o.ConfigFile == nil {
		filename := defaultConfigFile
		o.ConfigFile = &filename
	}

	logger := slog.New(NewNilHandler())
	switch h.(type) {
	case nil:
		panic("slog.Handler must not be nil")
	case *Handler:
		panic("slog.Handler must not be of type *Handler")
	default:
		// If debug mode is enabled, we use the given log Handler also for internal log messages.
		if o.Debug {
			logger = slog.New(h)
		}
	}

	ss := &slogscope{logger: logger, slogh: h, opts: &o}
	defer ss.initHandler()

	// We load the HandlerOptions.Config from a config file if no HandlerOptions.Config is provided.
	if ss.opts.Config == nil && ss.opts.ConfigFile != nil {
		ss.loadConfig()
	}

	ssHndl := &Handler{
		slogscope: ss,
	}
	ssHndl.h = ssHndl

	return ssHndl
}

func (h *Handler) Enabled(_ context.Context, lvl slog.Level) bool {
	globalLevel := h.getLogLevel(h.opts.Config.LogLevel)
	cInfo := getCallerInfo(5)
	if v, ok := h.pkgMap.Load(cInfo.PackageName); ok {
		p := v.(*pkg)
		if lvl >= p.logLevel.Level() {
			h.logger.Debug(fmt.Sprintf("use package log level=%q for package=%q", lvl, p.name))
			return true
		}
		return false //return lvl >= p.logLevel.Level()
	}
	h.logger.Debug(fmt.Sprintf("use global log level=%q for package=%q", globalLevel, cInfo.PackageName))
	return lvl >= globalLevel
}

func (h *Handler) Handle(ctx context.Context, rec slog.Record) error {
	return h.slogh.Handle(ctx, rec)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.slogh.WithAttrs(attrs)
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return h.slogh.WithGroup(name)
}

// GetConfig returns the current configuration, which may be adjusted and then used with UseConfig(cfg Config).
func (h *Handler) GetConfig() Config {
	return *h.opts.Config
}

// UseConfig takes a new Config and immediately applies it to the current configuration.
// It also disables any active file watcher.
func (h *Handler) UseConfig(cfg Config) {
	h.mu.Lock()
	h.opts.EnableFileWatcher = false
	h.opts.Config = &cfg
	h.mu.Unlock()

	h.initHandler()

	h.logger.Debug(fmt.Sprintf("using config: %#v", *h.opts.Config))
}

// UseConfigTemporarily takes a new Config and immediately applies it to the current configuration.
// In contrast to UseConfig(cfg	Config), this function automatically reverts to the state before calling the method,
// after revert amount of time has elapsed.
func (h *Handler) UseConfigTemporarily(cfg Config, revert time.Duration) {
	h.mu.Lock()
	oldCfg := h.GetConfig()
	enableFileWatcher := h.opts.EnableFileWatcher

	h.opts.EnableFileWatcher = false
	h.opts.Config = &cfg
	h.mu.Unlock()

	h.initHandler()

	go func() {
		<-time.After(revert)
		if enableFileWatcher {
			h.UseConfigFile()
		} else {
			h.UseConfig(oldCfg)
		}
	}()
	h.logger.Debug(fmt.Sprintf("using config: %#v", *h.opts.Config))
}

// UseConfigFile takes a filename as an argument that will be used for watching a config file for changes.
// If no such filename is given, the Handler uses the already existing ConfigFile from the HandlerOptions or,
// if not present, falls back to the default config file (specified via defaultConfigFile).
func (h *Handler) UseConfigFile(cfgFile ...string) {
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

	h.loadConfig().initHandler()
	h.logger.Debug(fmt.Sprintf("using config file (%s): %#v", *h.opts.ConfigFile, *h.opts.Config))
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
