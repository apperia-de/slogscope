package errorlogger

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/apperia-de/slogscope/examples/pkg/logger"
)

const (
	PackageName = "github.com/apperia-de/slogscope/examples/pkg/logger/errorlogger"
	LogInterval = 8 * 250 * time.Millisecond
	LogLevel    = slog.LevelError
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
				slog.Log(context.TODO(), l.LogLevel, l.Messages[idx], slog.Time("date", t))
			case <-l.Done:
				slog.Log(context.TODO(), 108, fmt.Sprintf("shutting down package: %q", l.PackageName))
				return
			}
		}
	}()
}

func (l *Logger) Stop() {
	close(l.Done)
}
