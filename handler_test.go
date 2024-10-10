package slogscope_test

import (
	"log/slog"
	"os"
	"testing"
	"testing/slogtest"
	"time"

	"github.com/apperia-de/slogscope"
	"github.com/stretchr/testify/assert"
)

func TestNewHandler(t *testing.T) {
	t.Run("wrapped slog.Handler must not be nil", func(t *testing.T) {
		testFunc := func() { slogscope.NewHandler(nil, nil) }
		assert.Panics(t, testFunc, "nil handler should have raised a panic")
	})

	t.Run("wrapped slog.Handler must not be of type *slogscope.Handler", func(t *testing.T) {
		testFunc := func() { slogscope.NewHandler(slogscope.NewHandler(slogscope.NewNilHandler(), nil), nil) }
		assert.Panics(t, testFunc, "wrapped handler of type *slogscope.Handler should have raised a panic")
	})

	t.Run("test if the given HandlerOptions.Config takes precedence over HandlerOptions.ConfigFile.", func(t *testing.T) {
		buf.Reset()
		h := slogscope.NewHandler(slog.NewJSONHandler(&buf, nil), &slogscope.HandlerOptions{
			EnableFileWatcher: false,
			ConfigFile:        &testConfigFile,
			Config: &slogscope.Config{
				LogLevel: "DEBUG",
			},
			Debug: false,
		})
		if err := slogtest.TestHandler(h, testResults(t, &buf)); err != nil {
			t.Fatal(err)
		}
		cfg := h.GetConfig()
		assert.Equal(t, "DEBUG", cfg.LogLevel)
		assert.Equal(t, []slogscope.Package(nil), cfg.Packages)
	})

	t.Run("test slogscope.Handler with a wrapped slog.JSONHandler and given Config", func(t *testing.T) {
		buf.Reset()
		h := slogscope.NewHandler(slog.NewJSONHandler(&buf, nil), &slogscope.HandlerOptions{
			EnableFileWatcher: false,
			ConfigFile:        nil,
			Config: &slogscope.Config{
				LogLevel: "INFO",
			},
			Debug: false,
		})
		if err := slogtest.TestHandler(h, testResults(t, &buf)); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test slogscope.Handler with a wrapped slog.JSONHandler with Config from config file (slogscope.test_config.yml)", func(t *testing.T) {
		buf.Reset()
		h := slogscope.NewHandler(slog.NewJSONHandler(&buf, nil), &slogscope.HandlerOptions{
			EnableFileWatcher: false,
			ConfigFile:        &testConfigFile,
			Config:            nil,
			Debug:             false,
		})
		if err := slogtest.TestHandler(h, testResults(t, &buf)); err != nil {
			t.Fatal(err)
		}
		cfg := h.GetConfig()
		assert.Equal(t, slogscope.LogLevelDebug, cfg.LogLevel)
	})

	t.Run("test default config file is missing", func(t *testing.T) {
		var missingConfigFile = "test/data/default_config_is_missing.yml"

		h := slogscope.NewHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}), &slogscope.HandlerOptions{
			EnableFileWatcher: false,
			ConfigFile:        &missingConfigFile,
		})
		cfg := h.GetConfig()
		assert.Equal(t, "INFO", cfg.LogLevel)
		assert.Equal(t, []slogscope.Package(nil), cfg.Packages)
	})

	t.Run("test with debug mode enabled", func(t *testing.T) {
		buf.Reset()
		_ = slogscope.NewHandler(slog.NewTextHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}), &slogscope.HandlerOptions{
			EnableFileWatcher: false,
			Debug:             true,
		})
		line, err := buf.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		assert.Contains(t, line, "msg=\"debug mode enabled\"")
	})

	t.Run("test config file gets changed, renamed or removed", func(t *testing.T) {
		data, err := os.ReadFile(testConfigFile)
		assert.NoError(t, err)
		err = os.WriteFile(testConfigFile+"_tmp", data, 0644)
		assert.NoError(t, err)
		h := setupHandlerWithConfigFile(testConfigFile + "_tmp")
		cfg := h.GetConfig()
		assert.Equal(t, slogscope.LogLevelDebug, cfg.LogLevel)
		err = os.WriteFile(testConfigFile+"_tmp", []byte("log_level: INFO"), 0644)
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		cfg = h.GetConfig()
		assert.Equal(t, slogscope.LogLevelInfo, cfg.LogLevel)
		// Rename config file
		err = os.Rename(testConfigFile+"_tmp", testConfigFile+"_tmp2")
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		// Config should not have changed if a config file gets renamed
		cfg = h.GetConfig()
		assert.Equal(t, slogscope.LogLevelInfo, cfg.LogLevel)
		// Use the renamed config file again for watching changes and then remove it.
		h.UseConfigFile(testConfigFile + "_tmp2")
		// Remove config file
		err = os.Remove(testConfigFile + "_tmp2")
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		cfg = h.GetConfig()
		assert.Equal(t, slogscope.LogLevelInfo, cfg.LogLevel)
	})
}

func TestHandler_GetConfig(t *testing.T) {
	h := slogscope.NewHandler(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}), &slogscope.HandlerOptions{
		Config: &slogscope.Config{
			LogLevel: "DEBUG",
		},
	})

	cfg := h.GetConfig()
	assert.Equal(t, slogscope.LogLevelDebug, cfg.LogLevel)
	assert.Equal(t, []slogscope.Package(nil), cfg.Packages)
}

func TestHandler_UseConfig(t *testing.T) {
	h := slogscope.NewHandler(slogscope.NewNilHandler(), &slogscope.HandlerOptions{Config: &oldCfg})
	cfg := h.GetConfig()
	assert.Equal(t, slogscope.LogLevelDebug, cfg.LogLevel)
	h.UseConfig(newCfg)
	cfg = h.GetConfig()
	assert.Equal(t, slogscope.LogLevelError, cfg.LogLevel)
}

func TestHandler_UseConfigTemporarily(t *testing.T) {
	var (
		h *slogscope.Handler
		l *slog.Logger
	)

	t.Run("test with previous settings reset to slogscope.HandlerOptions.Config", func(t *testing.T) {
		buf.Reset()
		h = setupHandlerWithConfig(oldCfg)

		l = slog.New(h)
		l.Debug("Debug message printed")
		l.Error("Error message printed")
		cfg := h.GetConfig()
		assert.Equal(t, slogscope.LogLevelDebug, cfg.LogLevel)

		// Switch config for one second and print again both log messages.
		h.UseConfigTemporarily(newCfg, time.Millisecond*100)

		l.Debug("Debug message not printed")
		l.Error("Error message printed")
		// Get new config and assert that the global log level has changed from DEBUG to ERROR.
		cfg = h.GetConfig()
		assert.Equal(t, slogscope.LogLevelError, cfg.LogLevel)
		// Sleep some time before printing another debug message.
		time.Sleep(time.Millisecond * 110)
		l.Debug("Debug message printed")
		l.Error("Error message printed")

		// Get config again and verify that it is the old one and assert that the
		// global log level has changed back from ERROR to DEBUG.
		cfg = h.GetConfig()
		assert.Equal(t, slogscope.LogLevelDebug, cfg.LogLevel)

		// Assert that two DEBUG and three ERROR messages have been logged in total.
		assert.Equal(t, 2, countLogMessageByLogLevel(buf, slogscope.LogLevelDebug))
		assert.Equal(t, 3, countLogMessageByLogLevel(buf, slogscope.LogLevelError))
	})

	t.Run("test with previous settings reset to slogscope.HandlerOptions.ConfigFile", func(t *testing.T) {
		buf.Reset()
		h = setupHandlerWithConfigFile("test/data/slogscope.test_config.yml")

		l = slog.New(h)
		l.Info("Info message printed")
		l.Error("Error message printed")
		cfg := h.GetConfig()
		assert.Equal(t, slogscope.LogLevelDebug, cfg.LogLevel)

		// Switch config for one second and print again both log messages.
		h.UseConfigTemporarily(newCfg, time.Millisecond*100)

		l.Info("Info message not printed")
		l.Error("Error message printed")
		// Get new config and assert that the global log level has changed from DEBUG to ERROR.
		cfg = h.GetConfig()
		assert.Equal(t, slogscope.LogLevelError, cfg.LogLevel)
		// Sleep some time before printing another debug message.
		time.Sleep(time.Millisecond * 110)
		l.Info("Info message printed")
		l.Error("Error message printed")

		// Get config again and verify that it is the old one and assert that the
		// global log level has changed back from ERROR to DEBUG.
		cfg = h.GetConfig()
		assert.Equal(t, slogscope.LogLevelDebug, cfg.LogLevel)

		// Assert that two DEBUG and three ERROR messages have been logged in total.
		assert.Equal(t, 2, countLogMessageByLogLevel(buf, slogscope.LogLevelInfo))
		assert.Equal(t, 3, countLogMessageByLogLevel(buf, slogscope.LogLevelError))
	})
}

func TestHandler_UseConfigFile(t *testing.T) {
	var (
		h *slogscope.Handler
		l *slog.Logger
	)

	setup := func(cfgFile string) {
		buf.Reset()
		h = setupHandlerWithConfig(newCfg)

		l = slog.New(h)
		l.Info("Info message printed")
		l.Error("Error message printed")
		currentCfg := h.GetConfig()
		assert.Equal(t, slogscope.LogLevelError, currentCfg.LogLevel)

		if cfgFile != "" {
			h.UseConfigFile(cfgFile)
		} else {
			h.UseConfigFile()
		}
		l.Info("Info message printed")
		l.Error("Error message printed")
	}

	t.Run("test file config without package override", func(t *testing.T) {
		setup("test/data/slogscope.test_config.yml")
		currentCfg := h.GetConfig()
		assert.Equal(t, slogscope.LogLevelDebug, currentCfg.LogLevel)
		// Assert that two DEBUG and three ERROR messages have been logged in total.
		assert.Equal(t, 1, countLogMessageByLogLevel(buf, slogscope.LogLevelInfo))
		assert.Equal(t, 2, countLogMessageByLogLevel(buf, slogscope.LogLevelError))
	})

	t.Run("test file config with package override", func(t *testing.T) {
		setup("test/data/slogscope.test_config_with_test_package.yml")

		currentCfg := h.GetConfig()
		assert.Equal(t, slogscope.LogLevelInfo, currentCfg.LogLevel)
		assert.Equal(t, "github.com/apperia-de/slogscope_test", currentCfg.Packages[0].Name)
		assert.Equal(t, slogscope.LogLevelError, currentCfg.Packages[0].LogLevel)

		// Assert that two DEBUG and three ERROR messages have been logged in total.
		assert.Equal(t, 0, countLogMessageByLogLevel(buf, slogscope.LogLevelInfo))
		assert.Equal(t, 2, countLogMessageByLogLevel(buf, slogscope.LogLevelError))
	})
}

func TestHandler_GetLogLevel(t *testing.T) {
	h := setupHandlerWithConfig(oldCfg)

	tests := []struct {
		level     string
		slogLevel slog.Level
	}{
		{"DEBUG", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"ERROR", slog.LevelError},
		{"ERROR+1", slog.LevelError + 1},
		{"ERROR-1", slog.LevelError - 1},
		{"DEBUG+4", slog.LevelInfo},
		{"DEBUG+8", slog.LevelWarn},
		{"DEBUG+12", slog.LevelError},
		{"DEBUG+100", slog.LevelDebug + 100},
		{"error-100", slog.LevelError - 100},
		{"inFo-10", slog.LevelInfo - 10},
		{"XJDHFIW§R§ü+234'", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			ll := h.GetLogLevel(tt.level)
			assert.Equal(t, tt.slogLevel, ll)
		})
	}
}
