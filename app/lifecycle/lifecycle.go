package lifecycle

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Manage lifecycle of all internal services
func Run() error {
	var wg sync.WaitGroup
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	// shutdownErr := make(chan error)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	wg.Add(1)
	go func() {
		defer wg.Done()
		// When this goroutine exits, it means we are shutting down
		// Better cancel the main context of the app

		slog.Debug("starting shutdown handler")
		// No for loop here since we are dealing with one shutdown trigger only
		// We only need for loop if we handle multiple types of events life configReload, healthCheck, etc.
		select {
		case s := <-quit:
			slog.Info("received signal", "signal", s)
			slog.Info("initiating graceful shutdown...")
		}

		// Context for inflight requests to complete
		// TODO: Replace the _ with shutdownCtx when adding shutting down logic for server/tray
		_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// TODO: Adding server/tray shutdown logic here

		slog.Info("shutdown complete")
	}()

	wg.Wait()

	return nil
}
