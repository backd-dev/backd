package deno

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/backd-dev/backd/internal/auth"
	"github.com/backd-dev/backd/internal/db"
	"github.com/backd-dev/backd/internal/secrets"
	"github.com/robfig/cron/v3"
)

// PoolConfig defines configuration for the Deno process pool
type PoolConfig struct {
	MinWorkers    int           // Minimum number of worker processes to keep alive
	MaxWorkers    int           // Maximum number of worker processes
	IdleTimeout   time.Duration // How long a worker can be idle before being terminated
	FunctionRoot  string        // Root directory containing function code
	WorkerTimeout time.Duration // Timeout for individual function calls
	MaxMemory     int64         // Maximum memory per worker in bytes
}

// DefaultPoolConfig returns sensible defaults for the pool configuration
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MinWorkers:    2,
		MaxWorkers:    10,
		IdleTimeout:   5 * time.Minute,
		FunctionRoot:  "./functions",
		WorkerTimeout: 30 * time.Second,
		MaxMemory:     256 * 1024 * 1024, // 256MB
	}
}

// Runner represents a single Deno subprocess
type Runner struct {
	ID         string
	Process    *os.Process
	SocketPath string
	Ready      bool
	LastUsed   time.Time
	IdleTimer  *time.Timer
}

// Job represents a job in the queue
type Job struct {
	ID          string    `json:"id"`
	AppName     string    `json:"app_name"`
	Function    string    `json:"function"`
	Input       []byte    `json:"input"`
	Status      string    `json:"status"` // pending, running, completed, failed
	MaxAttempts int       `json:"max_attempts"`
	Attempts    int       `json:"attempts"`
	CreatedAt   time.Time `json:"created_at"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

// CronJob represents a cron job definition
type CronJob struct {
	ID       string    `json:"id"`
	AppName  string    `json:"app_name"`
	Function string    `json:"function"`
	Schedule string    `json:"schedule"` // Cron expression
	Active   bool      `json:"active"`
	Created  time.Time `json:"created"`
}

// Deno interface defines the main Deno runtime operations
type Deno interface {
	// Start initializes the Deno runtime and process pool
	Start(ctx context.Context) error

	// Stop gracefully shuts down the Deno runtime
	Stop(ctx context.Context) error

	// InvokeFunction executes a function in the Deno runtime
	InvokeFunction(ctx context.Context, appName, fnName string, input []byte) ([]byte, error)

	// StartJobWorker starts the job processing worker
	StartJobWorker(ctx context.Context) error

	// RegisterCronJobs registers cron jobs for the given applications
	RegisterCronJobs(ctx context.Context, appNames []string) error
}

// denoImpl implements the Deno interface
type denoImpl struct {
	config    PoolConfig
	pool      *Pool
	db        db.DB
	auth      auth.Auth
	secrets   secrets.Secrets
	cron      *cron.Cron
	jobWorker *JobWorker
}

// NewDeno creates a new Deno runtime instance
func NewDeno(config PoolConfig, db db.DB, auth auth.Auth, secrets secrets.Secrets) Deno {
	return &denoImpl{
		config:  config,
		db:      db,
		auth:    auth,
		secrets: secrets,
		cron:    cron.New(),
	}
}

// Start initializes the Deno runtime and process pool
func (d *denoImpl) Start(ctx context.Context) error {
	// Initialize the process pool
	d.pool = NewPool(d.config)

	// Start internal HTTP server
	go d.startInternalServer()

	return nil
}

// Stop gracefully shuts down the Deno runtime
func (d *denoImpl) Stop(ctx context.Context) error {
	if d.pool != nil {
		if err := d.pool.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown pool: %w", err)
		}
	}

	if d.cron != nil {
		d.cron.Stop()
	}

	if d.jobWorker != nil {
		d.jobWorker.Stop()
	}

	return nil
}

// InvokeFunction executes a function in the Deno runtime
func (d *denoImpl) InvokeFunction(ctx context.Context, appName, fnName string, input []byte) ([]byte, error) {
	if d.pool == nil {
		return nil, fmt.Errorf("pool not initialized")
	}

	runner, err := d.pool.Acquire()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire runner: %w", err)
	}
	defer d.pool.Release(runner)

	// Send request to runner via Unix socket
	return d.invokeViaSocket(ctx, runner, appName, fnName, input)
}

// StartJobWorker starts the job processing worker
func (d *denoImpl) StartJobWorker(ctx context.Context) error {
	d.jobWorker = NewJobWorker(d.db, d)
	return d.jobWorker.Start()
}

// RegisterCronJobs registers cron jobs for the given applications
func (d *denoImpl) RegisterCronJobs(ctx context.Context, appNames []string) error {
	for _, appName := range appNames {
		if err := d.registerAppCronJobs(ctx, appName); err != nil {
			return fmt.Errorf("failed to register cron jobs for app %s: %w", appName, err)
		}
	}

	d.cron.Start()
	return nil
}
