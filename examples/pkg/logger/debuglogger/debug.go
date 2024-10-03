package debuglogger

import (
	"fmt"
	"github.com/apperia-de/slogscope/examples/pkg/logger"
	"log/slog"
	"time"
)

const (
	PackageName = "github.com/apperia-de/slogscope/examples/pkg/logger/debuglogger"
	LogInterval = 1 * 250 * time.Millisecond
	LogLevel    = slog.LevelDebug
)

type Logger logger.DefaultLogger

func New() logger.Logger {
	return (*Logger)(logger.New(LogLevel, LogInterval, PackageName))
}

func (l *Logger) Start() {
	if l.Done != nil {
		return
	}
	l.Done = make(chan struct{})
	go func() {
		for {
			select {
			case t := <-time.Tick(l.LogInterval):
				idx := t.UnixNano() % int64(len(l.Messages))
				slog.Log(nil, l.LogLevel, l.Messages[idx], slog.Time("date", t))
			case <-l.Done:
				slog.Log(nil, 108, fmt.Sprintf("shutting down package: %q", l.PackageName))
				return
			}
		}
	}()
}

func (l *Logger) Stop() {
	close(l.Done)
}
