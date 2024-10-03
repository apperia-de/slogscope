package app

import (
	"github.com/apperia-de/slogscope/examples/pkg/logger"
	"github.com/apperia-de/slogscope/examples/pkg/logger/debuglogger"
	"github.com/apperia-de/slogscope/examples/pkg/logger/errorlogger"
	"github.com/apperia-de/slogscope/examples/pkg/logger/infologger"
	"github.com/apperia-de/slogscope/examples/pkg/logger/warnlogger"
)

type App struct{}

func New() *App {
	return &App{}
}

func (a *App) GetLoggers() []logger.Logger {
	var loggers []logger.Logger
	loggers = append(loggers, debuglogger.New(), infologger.New(), warnlogger.New(), errorlogger.New())

	return loggers
}
