package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

var CLIName = "clue"

func SpawnServer(ctx context.Context, command string) (chan int, error) {
	done := make(chan int)

	go func() {
		for {
			slog.Info("starting the server...")
			cmd, err := start(ctx, command)
			if err != nil {
				// TODO: Keep track of crash count
				continue
			}
			// Wait for command to exit, also for stdin and stdout
			cmd.Wait()
			var code int
			if cmd.ProcessState != nil {
				code = cmd.ProcessState.ExitCode()
			}

			select {
			case <-ctx.Done():
				slog.Info(fmt.Sprintf("server shutdown with exit code %d", code))
				done <- code
				return
			default:
				break
			}
		}
	}()

	return done, nil
}

// Start the server which can be executed by the serve external command
func start(ctx context.Context, command string) (*exec.Cmd, error) {
	cmd := getCmd(ctx, getCLIFullPath(command))
	// TODO: Log output of the server
	// TODO: Rotate log file here
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			err := terminate(cmd)
			if err != nil {
				slog.Warn("error trying to gracefully terminate server", "err", err)
				return cmd.Process.Kill()
			}

			// Poll the process status
			// Give the process a chance to exit gracefully
			// before being forcefully killed
			tick := time.NewTicker(10 * time.Millisecond)
			defer tick.Stop()

			for {
				select {
				case <-tick.C:
					exited, err := isProcessExited(cmd.Process.Pid)
					if err != nil {
						return err
					}

					if exited {
						return nil
					}
				case <-time.After(5 * time.Second):
					slog.Warn("graceful server shutdown timeout, killing", "pid", cmd.Process.Pid)
					return cmd.Process.Kill()
				}
			}
		}
		return nil
	}
	return cmd, nil
}

func getCLIFullPath(command string) string {
	var cmdPath string
	// Can it get the exe running in app/ ?
	// No guarantee that the path is still pointing to the correct exe per docs
	// appExe, err := os.Executable()

	// if err == nil {
	// 	// TODO: Check in tray and bin dir
	// }

	// Look through PATH env variable
	// cmdPath, err = exec.LookPath(command)
	// if err == nil {
	// 	_, err := os.Stat(cmdPath)
	// }
	pwd, err := os.Getwd()
	if err == nil {
		cmdPath = filepath.Join(pwd, command)
		_, err = os.Stat(cmdPath)
		if err == nil {
			print(cmdPath)
			return cmdPath
		}
	}
	return command
}

// TODO: Specific for unix, windows need something else
func getCmd(ctx context.Context, cmd string) *exec.Cmd {
	return exec.CommandContext(ctx, cmd, "serve")
}

// TODO: Specific for unix, windows need something else
func terminate(cmd *exec.Cmd) error {
	return cmd.Process.Signal(os.Interrupt)
}

func isProcessExited(pid int) (bool, error) {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, fmt.Errorf("failed to find process: %v", err)
	}

	// No signal is sent, but error checking is still performed
	// To check the existence of a process id
	err = proc.Signal(syscall.Signal(0))
	if err != nil {
		if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
			return true, nil
		}

		return false, fmt.Errorf("error signaling process: %v", err)
	}

	return false, nil
}
