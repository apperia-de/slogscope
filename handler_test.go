package slogscope_test

import (
	"bytes"
	"github.com/apperia-de/slogscope"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"regexp"
	"strings"
	"testing"
	"time"
)

var (
	oldCfg = slogscope.Config{
		LogLevel: slogscope.LogLevelDebug,
	}
	newCfg = slogscope.Config{
		LogLevel: slogscope.LogLevelError,
	}
)

func TestHandler_GetConfig(t *testing.T) {
	var buf bytes.Buffer
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
	var buf bytes.Buffer
	h := slogscope.NewHandler(slog.NewTextHandler(&buf, nil), &slogscope.HandlerOptions{Config: &oldCfg})
	l := slog.New(h)
	l.Debug("Debug message printed")
	l.Error("Error message printed")
	cfg := h.GetConfig()
	assert.Equal(t, slogscope.LogLevelDebug, cfg.LogLevel)

	// Switch config for one second and print again both log messages.
	h.UseConfigTemporarily(newCfg, time.Second)
	l.Debug("Debug message not printed")
	l.Error("Error message printed")

	// Get new config and assert that the global log level has changed from DEBUG to ERROR.
	cfg = h.GetConfig()
	assert.Equal(t, slogscope.LogLevelError, cfg.LogLevel)
	// Sleep some time before printing another debug message.
	time.Sleep(time.Second + time.Millisecond*100)
	l.Debug("Debug message printed")
	l.Error("Error message printed")

	// Get config again and verify that it is the old one and assert that the
	// global log level has changed back from ERROR to DEBUG.
	cfg = h.GetConfig()
	assert.Equal(t, slogscope.LogLevelDebug, cfg.LogLevel)

	// Assert that two DEBUG and three ERROR messages have been logged in total.
	assert.Equal(t, 2, countLogMessageByLogLevel(buf, slogscope.LogLevelDebug))
	assert.Equal(t, 3, countLogMessageByLogLevel(buf, slogscope.LogLevelError))
}

func TestHandler_UseConfigFile(t *testing.T) {
	var (
		buf     bytes.Buffer
		cfgFile = "test/data/slogscope.test_config.yml"
	)

	h := slogscope.NewHandler(slog.NewTextHandler(&buf, nil),
		&slogscope.HandlerOptions{
			Config: &newCfg,
		},
	)

	l := slog.New(h)
	l.Info("Info message printed")
	l.Error("Error message printed")
	cfg := h.GetConfig()
	assert.Equal(t, slogscope.LogLevelError, cfg.LogLevel)

	h.UseConfigFile(cfgFile)
	cfg = h.GetConfig()
	assert.Equal(t, slogscope.LogLevelInfo, cfg.LogLevel)
	l.Info("Info message printed")
	l.Error("Error message printed")

	// Assert that two DEBUG and three ERROR messages have been logged in total.
	assert.Equal(t, 1, countLogMessageByLogLevel(buf, slogscope.LogLevelInfo))
	assert.Equal(t, 2, countLogMessageByLogLevel(buf, slogscope.LogLevelError))
}

func countLogMessageByLogLevel(buf bytes.Buffer, logLevel string) int {
	regex := regexp.MustCompile(`level=(\w+)`)
	cnt := 0
	for line, err := buf.ReadString('\n'); err == nil; line, err = buf.ReadString('\n') {
		if regex.FindStringSubmatch(line)[1] == strings.ToUpper(logLevel) {
			cnt++
		}
		//fmt.Print(line)
	}
	return cnt
}
