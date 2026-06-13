package slogscope

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInternalHierarchicalMatching(t *testing.T) {
	ss := &slogscope{}
	ss.pkgMap = map[string]slog.Level{
		"github.com/apperia-de/slogscope_test/pkg/parent":              slog.LevelError,
		"github.com/apperia-de/slogscope_test/pkg/parent/sub/specific": slog.LevelDebug,
	}

	tests := []struct {
		pkgName     string
		expectedLvl slog.Level
		expectedOk  bool
	}{
		// Exact parent match
		{"github.com/apperia-de/slogscope_test/pkg/parent", slog.LevelError, true},
		// Subpackage inherits parent
		{"github.com/apperia-de/slogscope_test/pkg/parent/sub", slog.LevelError, true},
		{"github.com/apperia-de/slogscope_test/pkg/parent/sub/child", slog.LevelError, true},
		// Specific child override takes precedence
		{"github.com/apperia-de/slogscope_test/pkg/parent/sub/specific", slog.LevelDebug, true},
		{"github.com/apperia-de/slogscope_test/pkg/parent/sub/specific/child", slog.LevelDebug, true},
		// No match
		{"github.com/apperia-de/slogscope_test/pkg/other", 0, false},
		{"github.com/apperia-de/slogscope_test", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.pkgName, func(t *testing.T) {
			ss.pkgMapMu.RLock()
			curr := tt.pkgName
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

			assert.Equal(t, tt.expectedOk, hasOverride)
			assert.Equal(t, tt.expectedLvl, overrideLvl)
		})
	}
}

func TestWildcardNormalization(t *testing.T) {
	h := NewHandler(NewNilHandler(), &HandlerOptions{
		Config: &Config{
			LogLevel: "INFO",
			Packages: []Package{
				{Name: "pkg/a/...", LogLevel: "DEBUG"},
				{Name: "pkg/b/*", LogLevel: "WARN"},
				{Name: "pkg/c", LogLevel: "ERROR"},
			},
		},
	})

	h.pkgMapMu.RLock()
	defer h.pkgMapMu.RUnlock()

	assert.Contains(t, h.pkgMap, "pkg/a")
	assert.Contains(t, h.pkgMap, "pkg/b")
	assert.Contains(t, h.pkgMap, "pkg/c")
	assert.NotContains(t, h.pkgMap, "pkg/a/...")
	assert.NotContains(t, h.pkgMap, "pkg/b/*")

	assert.Equal(t, slog.LevelDebug, h.pkgMap["pkg/a"])
	assert.Equal(t, slog.LevelWarn, h.pkgMap["pkg/b"])
	assert.Equal(t, slog.LevelError, h.pkgMap["pkg/c"])
}
