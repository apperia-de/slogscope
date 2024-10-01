package slogscope_test

import (
	"bytes"
	"encoding/json"
	"github.com/apperia-de/slogscope"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
	"testing"
	"testing/slogtest"
)

var testConfigFile = "test/data/slogscope.test_config.yml"

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
		var buf bytes.Buffer
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
		var buf bytes.Buffer
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
		var buf bytes.Buffer
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
		assert.Equal(t, "INFO", cfg.LogLevel)
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
		var buf bytes.Buffer
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
}

// testResults is a helper function for the slogtest.TestHandler function
func testResults(t *testing.T, buf *bytes.Buffer) func() []map[string]any {
	return func() []map[string]any {
		var ms []map[string]any
		for _, line := range bytes.Split(buf.Bytes(), []byte{'\n'}) {
			if len(line) == 0 {
				continue
			}
			var m map[string]any
			if err := json.Unmarshal(line, &m); err != nil {
				t.Fatal(err) // In a real test, use t.Fatal.
			}
			ms = append(ms, m)
		}
		return ms
	}
}
