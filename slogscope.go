package slogscope

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Constants for debug mode and defaults values.
const (
	debugMode         = false
	defaultLogLevel   = LogLevelInfo
	defaultConfigFile = "slogscope.yml"
)

// Available log levels for the Config.
const (
	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
)

// slogscope contains all required handler configurations.
type slogscope struct {
	h      *handler
	slogh  slog.Handler
	opts   *HandlerOptions
	pkgMap sync.Map
	mu     sync.Mutex
	doneCh chan struct{}
}

// pkg contains information about the package name and corresponding log level.
type pkg struct {
	name     string
	logLevel slog.Level
}

// callInfo represents the result of the call to getCallerInfo(skip int).
type callInfo struct {
	FuncName    string `json:"funcName,omitempty"`
	PackageName string `json:"packageName"`
	Filename    string `json:"filename"`
	FilePath    string `json:"filePath"`
	LineNo      int    `json:"lineNo"`
	Source      string `json:"source"`
}

// initConfigFileWatcher watches the specified config file for changes and reflects them instantly in their
// log response during program runtime without restarting.
func (s *slogscope) initConfigFileWatcher() chan struct{} {
	if !checkFileExists(*s.opts.ConfigFile) {
		debugf("config file %q is missing! -> file watcher is disabled.", *s.opts.ConfigFile)
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		debug(err)
		return nil
	}

	// Add the config file to watch.
	err = watcher.Add(*s.opts.ConfigFile)
	if err != nil {
		debug(err)
		return nil
	}

	doneCh := make(chan struct{})
	// Start listening for events.
	go func() {
		debug("config file watcher started.")
		initHandlerWithoutWatcher := func(msg string) {
			debugf("config file (%s) was modified.", *s.opts.ConfigFile)
			opts := *s.opts
			opts.EnableFileWatcher = false
			opts.ConfigFile = nil
			s.initHandler(s.slogh, opts)
		}

		closeWatcher := func() {
			if err := watcher.Close(); err != nil {
				debug("config file watcher error: " + err.Error())
				return
			}
			debug("config file watcher stopped.")
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				switch {
				case event.Has(fsnotify.Remove):
					initHandlerWithoutWatcher("config file (%s) has been removed.")
					closeWatcher()
					return
				case event.Has(fsnotify.Rename):
					initHandlerWithoutWatcher("config file (%s) has been renamed.")
					closeWatcher()
					return
				case event.Has(fsnotify.Write):
					debug("config file (%s) was modified.", event.Name)
					s.initHandler(s.slogh, *s.opts)
					closeWatcher()
					return
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				debug("config file watcher error:", err)
			case <-doneCh:
				closeWatcher()
				return
			}
		}
	}()

	return doneCh
}

// initHandler initializes the slogscope instance depending on the given HandlerOptions.
// If opts.EnableFileWatcher == true, the handler will try to load the config from a config file,
// specified by HandlerOptions.ConfigFile (fallback filename is defaultConfigFile), and if that fails,
// it uses a default Config with "INFO" as global log level.
func (s *slogscope) initHandler(slogh slog.Handler, opts HandlerOptions) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.slogh = slogh
	s.opts = &opts

	if opts.EnableFileWatcher && opts.ConfigFile == nil {
		cfgFile := defaultConfigFile
		s.opts.ConfigFile = &cfgFile
	}

	// Config file takes precedence over the config object.
	if s.opts.ConfigFile != nil {
		s.opts.Config = s.loadConfig(*s.opts.ConfigFile)
		debug("loaded config from file:", *s.opts.ConfigFile)
	} else {
		debug("using given config:", *s.opts.Config)
	}

	s.pkgMap.Clear()
	for _, v := range s.opts.Config.Packages {
		p := &pkg{
			name:     v.Name,
			logLevel: s.getLogLevel(v.LogLevel),
		}
		s.pkgMap.Store(p.name, p)
	}

	if s.doneCh != nil {
		close(s.doneCh)
		s.doneCh = nil
	}
	if opts.EnableFileWatcher {
		s.doneCh = s.initConfigFileWatcher()
	}
}

func (s *slogscope) loadConfig(cfgFile string) *Config {
	var cfg Config

	// The default configuration for logging in case of the load config from file fails.
	defaultCfg := Config{
		LogLevel: defaultLogLevel,
	}

	data, err := os.ReadFile(cfgFile)
	if err != nil {
		debugf("error reading config file (%s): %s", slog.String("filename", cfgFile), err.Error())
		return &defaultCfg
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		debugf("error unmarshalling config file (%s): %s", slog.String("filename", cfgFile), err.Error())
		return &defaultCfg
	}

	return &cfg
}

// getLogLevel converts string log levels to slog.Level representation.
// Can be one of ["DEBUG", "INFO", "WARN" or "ERROR"].
// Additionally, it accepts the aforementioned strings +/- an integer for representing additional log levels, not
// defined by the log/slog package.
// Example: DEBUG-2 or ERROR+4
func (s *slogscope) getLogLevel(level string) slog.Level {
	levelMap := map[string]slog.Level{
		LogLevelDebug: slog.LevelDebug,
		LogLevelInfo:  slog.LevelInfo,
		LogLevelWarn:  slog.LevelWarn,
		LogLevelError: slog.LevelError,
	}
	level = strings.ToUpper(level)
	matches := regexp.MustCompile(`([a-zA-Z]+)(([+\-])(\d+))?`).FindStringSubmatch(level)

	slogLevel := levelMap[defaultLogLevel]
	if len(matches) != 5 {
		debugf("invalid log level: %q! -> fallback to log level %q", level, defaultLogLevel)
		return slogLevel
	}

	slogLevel, ok := levelMap[matches[1]]
	if !ok {
		debugf("invalid log level: %q! -> fallback to log level %q", level, defaultLogLevel)
		return slogLevel
	}

	if matches[4] != "" {
		nb, _ := strconv.Atoi(matches[4])
		if matches[3] == "-" {
			return slog.Level(int(slogLevel) - nb)
		}
		return slog.Level(int(slogLevel) + nb)
	}
	return slogLevel
}

// getCallerInfo returns the *callInfo for a caller.
// It includes the function name, package name, filename, filepath and line number.
// For convenience, there is also a Source attribute, containing the full filename and line number.
// As in runtime.Caller, the argument skip is the number of stack frames to ascend,
// with 0 identifying the caller of Caller.
func getCallerInfo(skip int) *callInfo {
	pc, file, lineNo, ok := runtime.Caller(skip)
	if !ok {
		return &callInfo{}
	}

	funcName := runtime.FuncForPC(pc).Name()
	filename := path.Base(file) // The Base function returns the last element of the path
	filePath := path.Dir(file)

	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}
	firstDot := strings.IndexByte(funcName[lastSlash:], '.') + lastSlash

	ci := &callInfo{
		FuncName:    funcName[firstDot+1:],
		PackageName: funcName[:firstDot],
		Filename:    filename,
		FilePath:    filePath,
		LineNo:      lineNo,
	}

	ci.Source = fmt.Sprintf("%s/%s:%d", ci.FilePath, ci.Filename, ci.LineNo)

	return ci
}

// checkFileExists returns true if a file exists at that location on disk.
func checkFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func debug(a ...any) {
	if debugMode {
		ci := getCallerInfo(2)
		fmt.Println("-- DEBUG-MODE --")
		fmt.Printf("   source: %s\n", ci.Source)
		fmt.Println(append([]any{"  message:"}, a...)...)
		fmt.Println("------")
	}
}

func debugf(format string, a ...any) {
	if debugMode {
		ci := getCallerInfo(2)
		fmt.Println("-- DEBUG-MODE --")
		fmt.Printf("   source: %s\n", ci.Source)
		fmt.Println(append([]any{"  message:"}, fmt.Sprintf(format, a...))...)
		fmt.Println("------")
	}
}
