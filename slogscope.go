package slogscope

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
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

// callSiteInfo represents the resolved log override settings for a call site.
type callSiteInfo struct {
	pkgName     string
	overrideLvl slog.Level
	hasOverride bool
}

// slogscope contains all required Handler configurations.
type slogscope struct {
	h         *Handler
	slogh     slog.Handler
	opts      *HandlerOptions
	logLvl    atomic.Int32 // Global log level (slog.Level)
	debug     atomic.Bool  // Debug mode flag
	pkgMapMu  sync.RWMutex
	pkgMap    map[string]slog.Level // Map from package name to its log level
	pcCacheMu sync.RWMutex
	pcCache   map[uintptr]callSiteInfo // Cache for PC to call site mapping
	//lvlMap sync.Map
	mu     sync.Mutex
	doneCh chan struct{}
	logger *slog.Logger
}

// initConfigFileWatcher watches the specified config file for changes and reflects them instantly in their
// log response during program runtime without restarting.
func (ss *slogscope) initConfigFileWatcher() chan struct{} {
	if !checkFileExists(ss.opts.ConfigFile) {
		ss.logger.Debug(fmt.Sprintf("config file %q does not exists! -> file watcher is disabled.", ss.opts.ConfigFile))
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		ss.logger.Debug(err.Error())
		return nil
	}

	// Add the config file to watch.
	err = watcher.Add(ss.opts.ConfigFile)
	if err != nil {
		ss.logger.Debug(err.Error())
		return nil
	}

	doneCh := make(chan struct{})
	// Start listening for events.
	go func() {
		ss.logger.Debug(fmt.Sprintf("started file watcher for config file (%s).", ss.opts.ConfigFile))

		closeWatcher := func() {
			if err := watcher.Close(); err != nil {
				ss.logger.Debug(fmt.Sprintf("file watcher error for config file (%s): %s.", ss.opts.ConfigFile, err.Error()))
				return
			}
			ss.logger.Debug(fmt.Sprintf("stopped file watcher for config file (%s).", ss.opts.ConfigFile))
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
				ss.logger.Debug(fmt.Sprintf("file watcher error for config file (%s): %s.", ss.opts.ConfigFile, err.Error()))
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
			Packages: nil,
		}

		// Create a config file if it does not already exist.
		if !checkFileExists(ss.opts.ConfigFile) {
			ss.opts.Config.Packages = ss.createPackageList()

			data, err := yaml.Marshal(ss.opts.Config)
			if err != nil {
				ss.logger.Error(err.Error())
			}
			err = os.WriteFile(ss.opts.ConfigFile, data, 0644)
			if err != nil {
				ss.logger.Error(err.Error())
			}
		}
	}

	ss.logger.Debug("use config:", "config", *ss.opts.Config)

	// Set global log level and debug flag
	ss.logLvl.Store(int32(ss.h.GetLogLevel(ss.opts.Config.LogLevel)))
	ss.debug.Store(ss.opts.Debug)

	ss.pkgMapMu.Lock()
	ss.pkgMap = make(map[string]slog.Level)
	for _, v := range ss.opts.Config.Packages {
		name := v.Name
		name = strings.TrimSuffix(name, "/...")
		name = strings.TrimSuffix(name, "/*")
		ss.pkgMap[name] = ss.h.GetLogLevel(v.LogLevel)
	}
	ss.pkgMapMu.Unlock()

	// Clear the call-site cache on configuration updates to apply new overrides
	ss.pcCacheMu.Lock()
	ss.pcCache = make(map[uintptr]callSiteInfo)
	ss.pcCacheMu.Unlock()

	if ss.doneCh != nil {
		close(ss.doneCh)
		ss.doneCh = nil
	}

	if ss.opts.EnableFileWatcher && ss.opts.ConfigFile != "" {
		ss.doneCh = ss.initConfigFileWatcher()
	}
}

func (ss *slogscope) loadConfig() *slogscope {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if !checkFileExists(ss.opts.ConfigFile) {
		ss.logger.Debug(fmt.Sprintf("config file (%s) does not exists! -> file watcher is disabled.", ss.opts.ConfigFile))
		ss.opts.Config = nil
		return ss
	}

	data, err := os.ReadFile(ss.opts.ConfigFile)
	if err != nil {
		ss.logger.Debug(fmt.Sprintf("error reading config file (%s): %s", ss.opts.ConfigFile, err.Error()))
		ss.opts.Config = nil
		return ss
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		ss.logger.Debug(fmt.Sprintf("error unmarshalling config file (%s): %s", ss.opts.ConfigFile, err.Error()))
		ss.opts.Config = nil
		return ss
	}
	ss.opts.Config = &cfg
	ss.logger.Debug(fmt.Sprintf("config file (%s) loaded.", ss.opts.ConfigFile))
	return ss
}

// resolveCallSite resolves the package name and finds the best hierarchical log level override
// for a given program counter, caching the result to keep subsequent checks near-instant.
func (ss *slogscope) resolveCallSite(pc uintptr) callSiteInfo {
	ss.pcCacheMu.RLock()
	if ss.pcCache != nil {
		if info, ok := ss.pcCache[pc]; ok {
			ss.pcCacheMu.RUnlock()
			return info
		}
	}
	ss.pcCacheMu.RUnlock()

	// 1. Resolve package name
	funcName := runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}
	firstDot := strings.IndexByte(funcName[lastSlash:], '.') + lastSlash
	pkgName := funcName[:firstDot]

	// 2. Perform prefix walking to find nearest hierarchical override
	ss.pkgMapMu.RLock()
	curr := pkgName
	var overrideLvl slog.Level
	var hasOverride bool
	for {
		if lvl, ok := ss.pkgMap[curr]; ok {
			overrideLvl = lvl
			hasOverride = true
			break
		}
		idx := strings.LastIndexByte(curr, '/')
		if idx < 0 {
			break
		}
		curr = curr[:idx]
	}
	ss.pkgMapMu.RUnlock()

	info := callSiteInfo{
		pkgName:     pkgName,
		overrideLvl: overrideLvl,
		hasOverride: hasOverride,
	}

	ss.pcCacheMu.Lock()
	if ss.pcCache == nil {
		ss.pcCache = make(map[uintptr]callSiteInfo)
	}
	ss.pcCache[pc] = info
	ss.pcCacheMu.Unlock()
	return info
}

// checkFileExists returns true if a file exists at that location on disk.
func checkFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

// createPackageList returns a list of the project package names retrieved via go list command
func (ss *slogscope) createPackageList() []Package {
	var packages []Package

	cmd := exec.Command("go", "list", "./...")
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput
	err := cmd.Run()
	if err != nil {
		ss.logger.Error(err.Error())
		return packages
	}

	pkgPaths := strings.Split(cmdOutput.String(), "\n")

	for _, pkgPath := range pkgPaths {
		if pkgPath != "" {
			packages = append(packages, Package{
				Name:     pkgPath,
				LogLevel: "ERROR",
			})
		}
	}

	return packages
}
