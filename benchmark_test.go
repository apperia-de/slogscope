package slogscope_test

import (
	"bytes"
	"github.com/apperia-de/slogscope"
	"log/slog"
	"testing"
)

func BenchmarkSlogScopeHandlerLogging(b *testing.B) {
	var buf bytes.Buffer
	h := slogscope.NewHandler(slog.NewJSONHandler(&buf, nil), nil)
	logger := slog.New(h)
	for i := 0; i < b.N; i++ {
		logger.Info("INFO LOG MESSAGE")
	}
}

func BenchmarkDefaultHandlerLogging(b *testing.B) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(h)
	for i := 0; i < b.N; i++ {
		logger.Info("INFO LOG MESSAGE")
	}
}

func BenchmarkNilHandlerLogging(b *testing.B) {
	h := slogscope.NewNilHandler()
	logger := slog.New(h)
	for i := 0; i < b.N; i++ {
		logger.Info("INFO LOG MESSAGE")
	}
}
