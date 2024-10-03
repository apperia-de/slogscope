package main

import (
	"fmt"
	"github.com/apperia-de/slogscope"
	"github.com/apperia-de/slogscope/examples/pkg/app"
	"log/slog"
	"os"
	"os/signal"
	"time"
)

func init() {
	h := slogscope.NewHandler(slog.NewTextHandler(os.Stderr, nil), &slogscope.HandlerOptions{EnableFileWatcher: true})
	l := slog.New(h)
	slog.SetDefault(l)
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	fmt.Println("Press CTRL-C to exit")
	loggers := app.New().GetLoggers()
	for _, l := range loggers {
		l.Start()
	}

	<-c
	slog.Log(nil, 108, "Shutdown example app...")
	for _, l := range loggers {
		l.Stop()
	}
	time.Sleep(100 * time.Millisecond)
	slog.Log(nil, 108, "... done!")
}
