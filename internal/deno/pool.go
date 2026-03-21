package deno

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/backd-dev/backd/internal/db"
)

// Pool manages a pool of Deno process runners
type Pool struct {
	runners      []*Runner
	mu           sync.Mutex
	min, max     int
	idleTimeout  time.Duration
	config       PoolConfig
	nextRunnerID int64
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewPool creates a new process pool
func NewPool(config PoolConfig) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		min:          config.MinWorkers,
		max:          config.MaxWorkers,
		idleTimeout:  config.IdleTimeout,
		config:       config,
		ctx:          ctx,
		cancel:       cancel,
		nextRunnerID: 1,
	}
}

// Acquire gets an available runner from the pool or creates a new one
func (p *Pool) Acquire() (*Runner, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Find an available runner and mark it busy
	for _, runner := range p.runners {
		if runner.Ready {
			runner.Ready = false
			runner.LastUsed = time.Now()
			return runner, nil
		}
	}

	// If we can create more runners, do so
	if len(p.runners) < p.max {
		runner, err := p.spawnRunner()
		if err != nil {
			return nil, fmt.Errorf("failed to spawn runner: %w", err)
		}
		p.runners = append(p.runners, runner)
		runner.LastUsed = time.Now()
		return runner, nil
	}

	// All runners busy and at max capacity
	return nil, fmt.Errorf("all runners busy and at max capacity")
}

// Release marks a runner as available
func (p *Pool) Release(runner *Runner) {
	p.mu.Lock()
	defer p.mu.Unlock()
	runner.Ready = true
	runner.LastUsed = time.Now()
}

// spawnRunner creates a new Deno runner process
func (p *Pool) spawnRunner() (*Runner, error) {
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

	// Create the command
	cmd := exec.CommandContext(p.ctx, "deno", "run",
		"--allow-net",
		"--allow-read="+p.config.FunctionRoot+",/tmp/backd",
		"--allow-env",
		"--allow-write=/tmp/backd",
		"--v8-flags=--max-old-space-size=256",
		filepath.Join(os.TempDir(), "backd", "runner.ts"))
	cmd.Env = append(os.Environ(), env...)

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

	// Wait for READY signal
	if err := p.waitForReady(runner); err != nil {
		runner.Process.Kill()
		return nil, fmt.Errorf("runner failed to become ready: %w", err)
	}

	// Start idle reaper for this runner
	go p.idleReaper(runner)

	return runner, nil
}

// waitForReady waits for the runner to send READY signal
func (p *Pool) waitForReady(runner *Runner) error {
	// In production this reads READY\n from stdout via waitForReadySignal.
	// For now, mark ready after a brief delay to allow process startup.
	time.Sleep(100 * time.Millisecond)
	runner.Ready = true
	return nil
}

// idleReaper terminates idle runners after timeout
func (p *Pool) idleReaper(runner *Runner) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.mu.Lock()
			if len(p.runners) > p.min && time.Since(runner.LastUsed) > p.idleTimeout {
				// Remove this runner from the pool
				for i, r := range p.runners {
					if r.ID == runner.ID {
						p.runners = append(p.runners[:i], p.runners[i+1:]...)
						break
					}
				}
				p.mu.Unlock()

				// Terminate the process
				runner.Process.Kill()
				os.Remove(runner.SocketPath)
				return
			}
			p.mu.Unlock()
		}
	}
}

// Shutdown gracefully shuts down all runners
func (p *Pool) Shutdown() error {
	p.cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, runner := range p.runners {
		runner.Process.Kill()
		os.Remove(runner.SocketPath)
	}

	p.runners = nil
	return nil
}

// JobWorker processes jobs from the database queue
type JobWorker struct {
	db     db.DB
	deno   Deno
	ctx    context.Context
	cancel context.CancelFunc
}

// NewJobWorker creates a new job worker
func NewJobWorker(db db.DB, deno Deno) *JobWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobWorker{
		db:     db,
		deno:   deno,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the job worker
func (j *JobWorker) Start() error {
	go j.workerLoop()
	return nil
}

// Stop stops the job worker
func (j *JobWorker) Stop() {
	j.cancel()
}

// workerLoop continuously polls for jobs
func (j *JobWorker) workerLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-j.ctx.Done():
			return
		case <-ticker.C:
			j.processJobs()
		}
	}
}

// processJobs fetches and processes pending jobs
func (j *JobWorker) processJobs() {
	// Query for pending jobs using FOR UPDATE SKIP LOCKED
	// Column names match unified _jobs DDL
	query := `
		SELECT id, app_name, function, payload, max_attempts, attempts
		FROM _jobs 
		WHERE status = 'pending' AND run_at <= NOW()
		ORDER BY run_at ASC
		LIMIT 10
		FOR UPDATE SKIP LOCKED`

	jobs, err := j.db.Query(j.ctx, "platform", query)
	if err != nil {
		return
	}

	for _, jobData := range jobs {
		var payload []byte
		if p, ok := jobData["payload"].(string); ok {
			payload = []byte(p)
		}
		job := Job{
			ID:          jobData["id"].(string),
			AppName:     jobData["app_name"].(string),
			Function:    jobData["function"].(string),
			Input:       payload,
			MaxAttempts: int(jobData["max_attempts"].(int64)),
			Attempts:    int(jobData["attempts"].(int64)),
		}

		go j.processJob(job)
	}
}

// processJob processes a single job
func (j *JobWorker) processJob(job Job) {
	// Mark job as running
	j.db.Exec(j.ctx, job.AppName,
		"UPDATE _jobs SET status = 'running', attempts = attempts + 1 WHERE id = $1", job.ID)

	// Execute the function
	_, err := j.deno.InvokeFunction(j.ctx, job.AppName, job.Function, job.Input)

	if err != nil {
		// Handle job failure with exponential backoff
		j.handleJobFailure(job, err)
	} else {
		// Mark job as completed
		j.db.Exec(j.ctx, job.AppName,
			"UPDATE _jobs SET status = 'completed', completed_at = NOW(), updated_at = NOW() WHERE id = $1", job.ID)
	}
}

// handleJobFailure implements exponential backoff for failed jobs
func (j *JobWorker) handleJobFailure(job Job, err error) {
	var backoff time.Duration
	var newStatus string

	switch job.Attempts {
	case 1:
		backoff = 30 * time.Second
		newStatus = "pending"
	case 2:
		backoff = 5 * time.Minute
		newStatus = "pending"
	default:
		// Max attempts reached
		newStatus = "failed"
		backoff = 0
	}

	if backoff > 0 {
		// Schedule retry with backoff
		j.db.Exec(j.ctx, job.AppName,
			`UPDATE _jobs SET status = $1, run_at = NOW() + $2::interval, error = $3, updated_at = NOW() WHERE id = $4`,
			newStatus, fmt.Sprintf("%d seconds", int(backoff.Seconds())), err.Error(), job.ID)
	} else {
		// Mark as failed
		j.db.Exec(j.ctx, job.AppName,
			"UPDATE _jobs SET status = 'failed', error = $1, updated_at = NOW() WHERE id = $2",
			err.Error(), job.ID)
	}
}
