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

// slogscope contains all required Handler configurations.
type slogscope struct {
	h      *Handler
	slogh  slog.Handler
	opts   *HandlerOptions
	pkgMap sync.Map
	mu     sync.Mutex
	doneCh chan struct{}
	logger *slog.Logger
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
func (ss *slogscope) initConfigFileWatcher() chan struct{} {
	if !checkFileExists(*ss.opts.ConfigFile) {
		ss.logger.Debug(fmt.Sprintf("config file %q does not exists! -> file watcher is disabled.", *ss.opts.ConfigFile))
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		ss.logger.Debug(err.Error())
		return nil
	}

	// Add the config file to watch.
	err = watcher.Add(*ss.opts.ConfigFile)
	if err != nil {
		ss.logger.Debug(err.Error())
		return nil
	}

	doneCh := make(chan struct{})
	// Start listening for events.
	go func() {
		//ss.logger.Debug("config file watcher started.")
		ss.logger.Debug(fmt.Sprintf("started file watcher for config file (%s).", *ss.opts.ConfigFile))

		closeWatcher := func() {
			if err := watcher.Close(); err != nil {
				ss.logger.Debug(fmt.Sprintf("file watcher error for config file (%s): %s.", *ss.opts.ConfigFile, err.Error()))
				return
			}
			//ss.logger.Debug("config file watcher stopped.")
			ss.logger.Debug(fmt.Sprintf("stopped file watcher for config file (%s).", *ss.opts.ConfigFile))
		}

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				switch {
				case event.Has(fsnotify.Remove):
					ss.logger.Debug(fmt.Sprintf("config file (%s) was removed.", event.Name))
					closeWatcher()
					return
				case event.Has(fsnotify.Rename):
					ss.logger.Debug(fmt.Sprintf("config file (%s) was renamed.", event.Name))
					closeWatcher()
					return
				case event.Has(fsnotify.Write):
					ss.logger.Debug(fmt.Sprintf("config file (%s) was modified.", event.Name))
					ss.loadConfig().initHandler()
					closeWatcher()
					return
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				ss.logger.Debug(fmt.Sprintf("file watcher error for config file (%s): %s.", *ss.opts.ConfigFile, err.Error()))
			case <-doneCh:
				closeWatcher()
				return
			}
		}
	}()

	return doneCh
}

// initHandler initializes the slogscope instance depending on the given HandlerOptions.
// If opts.EnableFileWatcher == true, the Handler will try to load the config from a config file,
// specified by HandlerOptions.ConfigFile (fallback filename is defaultConfigFile), and if that fails,
// it uses a default Config with "INFO" as global log level.
func (ss *slogscope) initHandler() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.opts.Config == nil {
		ss.opts.Config = &Config{
			LogLevel: defaultLogLevel,
		}
	}

	ss.logger.Debug("use config:", "config", *ss.opts.Config)

	ss.pkgMap.Clear()
	for _, v := range ss.opts.Config.Packages {
		p := &pkg{
			name:     v.Name,
			logLevel: ss.getLogLevel(v.LogLevel),
		}
		ss.pkgMap.Store(p.name, p)
	}

	if ss.doneCh != nil {
		close(ss.doneCh)
		ss.doneCh = nil
	}

	if ss.opts.EnableFileWatcher && ss.opts.ConfigFile != nil {
		ss.doneCh = ss.initConfigFileWatcher()
	}
}

func (ss *slogscope) loadConfig() *slogscope {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !checkFileExists(*ss.opts.ConfigFile) {
		ss.logger.Debug(fmt.Sprintf("config file (%s) does not exists! -> file watcher is disabled.", *ss.opts.ConfigFile))
		ss.opts.Config = nil
		return ss
	}

	data, err := os.ReadFile(*ss.opts.ConfigFile)
	if err != nil {
		ss.logger.Debug(fmt.Sprintf("error reading config file (%s): %s", *ss.opts.ConfigFile, err.Error()))
		ss.opts.Config = nil
		return ss
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		ss.logger.Debug(fmt.Sprintf("error unmarshalling config file (%s): %s", *ss.opts.ConfigFile, err.Error()))
		ss.opts.Config = nil
		return ss
	}
	ss.opts.Config = &cfg
	ss.logger.Debug(fmt.Sprintf("config file (%s) loaded.", *ss.opts.ConfigFile))
	return ss
}

// getLogLevel converts string log levels to slog.Level representation.
// Can be one of ["DEBUG", "INFO", "WARN" or "ERROR"].
// Additionally, it accepts the aforementioned strings +/- an integer for representing additional log levels, not
// defined by the log/slog package.
// Example: DEBUG-2 or ERROR+4
func (ss *slogscope) getLogLevel(level string) slog.Level {
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
		ss.logger.Debug(fmt.Sprintf("invalid log level: %q! -> fallback to log level %q", level, defaultLogLevel))
		return slogLevel
	}

	slogLevel, ok := levelMap[matches[1]]
	if !ok {
		ss.logger.Debug(fmt.Sprintf("invalid log level: %q! -> fallback to log level %q", level, defaultLogLevel))
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
