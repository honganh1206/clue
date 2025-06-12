package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func Run(ctx context.Context, cancel context.CancelFunc) error {
	// TODO: Pass this ctx to the agent
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	// shutdownErr := make(chan error)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// TODO: When adding tray, add here

	go func() {
		slog.Debug("starting callback loop")
		for {
			select {
			// case err := <-shutdownErr:
			case <-quit:
				s := <-quit
				slog.Debug("shutting down due to signals: %s", s.String())
			}
		}
	}()

	cancel()
	slog.Info("Waiting for clue to shutdown...")
	// TODO: Handle shutdown error channel?
	slog.Info("clue exiting...")

	return nil
}
