package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxnguyen95/cfddns/internal/app"
)

func main() {
	once := flag.Bool("once", false, "run one sync cycle and exit")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, logger, *once); err != nil {
		logger.Error("application failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
