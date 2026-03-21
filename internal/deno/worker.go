package deno

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"
)

// spawnRunnerProcess creates and starts a new Deno runner process
func (p *Pool) spawnRunnerProcess() (*Runner, error) {
	id := atomic.AddInt64(&p.nextRunnerID, 1)
	runnerID := fmt.Sprintf("runner-%d", id)

	// Create socket path
	socketDir := filepath.Join(os.TempDir(), "backd")
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	socketPath := filepath.Join(socketDir, fmt.Sprintf("%s.sock", runnerID))

	// Prepare environment variables
	env := []string{
		fmt.Sprintf("BACKD_SOCKET_PATH=%s", socketPath),
		"BACKD_INTERNAL_URL=http://127.0.0.1:9191",
		fmt.Sprintf("BACKD_FUNCTIONS_ROOT=%s", p.config.FunctionRoot),
	}

	// Create the command with runner script
	runnerScriptPath := filepath.Join(os.TempDir(), "backd", "runner.ts")
	cmd := exec.CommandContext(p.ctx, "deno", "run",
		"--allow-net",
		"--allow-read="+p.config.FunctionRoot+",/tmp/backd",
		"--allow-env",
		"--allow-write=/tmp/backd",
		"--v8-flags=--max-old-space-size=256",
		runnerScriptPath)
	cmd.Env = append(os.Environ(), env...)

	// Create pipes for stdout/stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start deno process: %w", err)
	}

	runner := &Runner{
		ID:         runnerID,
		Process:    cmd.Process,
		SocketPath: socketPath,
		Ready:      false,
		LastUsed:   time.Now(),
	}

	// Wait for READY signal in a separate goroutine
	readyChan := make(chan error, 1)
	go func() {
		readyChan <- p.waitForReadySignal(stdout, stderr)
	}()

	// Wait for ready signal or timeout
	select {
	case err := <-readyChan:
		if err != nil {
			runner.Process.Kill()
			return nil, fmt.Errorf("runner failed to become ready: %w", err)
		}
	case <-time.After(10 * time.Second):
		runner.Process.Kill()
		return nil, fmt.Errorf("runner ready timeout")
	}

	return runner, nil
}

// waitForReadySignal waits for the runner to send READY signal
func (p *Pool) waitForReadySignal(stdout, stderr io.ReadCloser) error {
	defer stdout.Close()
	defer stderr.Close()

	// Read stdout looking for READY signal
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "READY" {
			return nil
		}
	}

	// If we get here, the process likely failed
	// Read stderr to get error information
	errorOutput, _ := io.ReadAll(stderr)
	return fmt.Errorf("process never became ready. stderr: %s", string(errorOutput))
}

// isRunnerAlive checks if a runner process is still alive
func (p *Pool) isRunnerAlive(runner *Runner) bool {
	if runner.Process == nil {
		return false
	}

	// Check if process is still running
	err := runner.Process.Signal(nil)
	return err == nil
}

// removeDeadRunner removes a dead runner from the pool
func (p *Pool) removeDeadRunner(runner *Runner) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove from pool
	for i, r := range p.runners {
		if r.ID == runner.ID {
			p.runners = append(p.runners[:i], p.runners[i+1:]...)
			break
		}
	}

	// Clean up resources
	if runner.Process != nil {
		runner.Process.Kill()
	}
	if runner.SocketPath != "" {
		os.Remove(runner.SocketPath)
	}
}

// replaceDeadRunner spawns a replacement for a dead runner
func (p *Pool) replaceDeadRunner(deadRunner *Runner) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Only replace if we're below max capacity
	if len(p.runners) >= p.max {
		return nil
	}

	// Spawn replacement
	newRunner, err := p.spawnRunner()
	if err != nil {
		return fmt.Errorf("failed to spawn replacement runner: %w", err)
	}

	p.runners = append(p.runners, newRunner)
	return nil
}

// monitorRunnerHealth monitors the health of all runners
func (p *Pool) monitorRunnerHealth() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.checkRunnerHealth()
		}
	}
}

// checkRunnerHealth checks the health of all runners
func (p *Pool) checkRunnerHealth() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, runner := range p.runners {
		if !p.isRunnerAlive(runner) {
			// Remove dead runner
			go func(r *Runner) {
				p.removeDeadRunner(r)
				p.replaceDeadRunner(r)
			}(runner)
		}
	}
}
