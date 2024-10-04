package slogscope_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/apperia-de/slogscope"
)

var (
	buf    bytes.Buffer
	oldCfg = slogscope.Config{
		LogLevel: slogscope.LogLevelDebug,
	}
	newCfg = slogscope.Config{
		LogLevel: slogscope.LogLevelError,
	}
	testConfigFile = "test/data/slogscope.test_config.yml"
)

func countLogMessageByLogLevel(buf bytes.Buffer, logLevel string) int {
	regex := regexp.MustCompile(`level=(\w+)`)
	cnt := 0
	for line, err := buf.ReadString('\n'); err == nil; line, err = buf.ReadString('\n') {
		if regex.FindStringSubmatch(line)[1] == strings.ToUpper(logLevel) {
			cnt++
		}
	}
	return cnt
}

func setupHandlerWithConfigFile(cfgFile string) *slogscope.Handler {
	return slogscope.NewHandler(slog.NewTextHandler(&buf, nil),
		&slogscope.HandlerOptions{
			EnableFileWatcher: true,
			ConfigFile:        &cfgFile,
		},
	)
}

func setupHandlerWithConfig(cfg slogscope.Config) *slogscope.Handler {
	return slogscope.NewHandler(slog.NewTextHandler(&buf, nil),
		&slogscope.HandlerOptions{
			Config: &cfg,
		},
	)
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
