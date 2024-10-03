package logger

import (
	"log/slog"
	"time"
)

type Logger interface {
	Start()
	Stop()
}

type DefaultLogger struct {
	Messages    []string
	PackageName string
	LogLevel    slog.Level
	LogInterval time.Duration
	Done        chan struct{}
}

func New(logLvl slog.Level, logIntvl time.Duration, pkgName string) *DefaultLogger {
	return &DefaultLogger{
		Messages: []string{
			"Something unimportant has happened",
			"Boring stuff happened",
			"Not needed to be collected",
		},
		PackageName: pkgName,
		LogLevel:    logLvl,
		LogInterval: logIntvl,
	}
}
