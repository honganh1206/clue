package lifecycle

import (
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Automatic recovery: Server can restart without user intervention
// This works more like a process supervisor for the desktop application
// Not a service manager for CLI usage
// Desktop/Tray App (lifecycle.Run())
// ├── Server Detection (IsServerRunning())
// ├── Server Spawning (SpawnServer())
// └── Server Monitoring/Restart Loop
func Run() {
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	var wg sync.WaitGroup
	// var done chan int
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
		// _, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		// defer cancel()

		// TODO: Adding server/tray shutdown logic here
		// cancel()

		slog.Info("shutdown complete")
	}()
	// if IsServerRunning(ctx) {
	// 	slog.Info("Detected another instance of clue running, exiting")
	// 	os.Exit(1)
	// } else {
	// CLIName is a global var storing the process name clue?
	// done, err := SpawnServer(ctx, CLIName)
	// if err != nil {
	// 	slog.Error(fmt.Sprintf("Failed to spawn clue server %s", err))
	// 	// done = make(chan int, 1)
	// 	// done <- 1
	// }
	// }

	// slog.Info("Waiting for clue server to shutdown...")
	// if done != nil {
	// 	<-done
	// }
	wg.Wait()
	slog.Info("clue app exiting")
}
