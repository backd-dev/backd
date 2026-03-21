package deno

import (
	"context"
	"fmt"
	"time"

	"github.com/backd-dev/backd/internal/db"
	"github.com/robfig/cron/v3"
)

// registerAppCronJobs registers cron jobs for a specific app
func (d *denoImpl) registerAppCronJobs(ctx context.Context, appName string) error {
	// This would load cron jobs from app configuration
	// For now, we'll implement a placeholder that could be extended later

	// Example of how cron jobs would be registered:
	// c := cron.New()
	// c.AddFunc("0 */5 * * *", func() {
	//     d.insertCronJob(appName, "my_function", "{}")
	// })
	// c.Start()

	return nil
}

// insertCronJob inserts a job triggered by cron
func (d *denoImpl) insertCronJob(appName, functionName, payload string) error {
	jobID := db.NewXID()

	err := d.db.Exec(context.Background(), appName,
		`INSERT INTO _jobs (id, function, payload, trigger, status, run_at, created_at)
		 VALUES ($1, $2, $3, 'cron', 'pending', NOW(), NOW())`,
		jobID, functionName, payload)

	if err != nil {
		return fmt.Errorf("failed to insert cron job: %w", err)
	}

	return nil
}

// CronJob represents a cron job configuration
type CronJobConfig struct {
	ID       string    `json:"id"`
	AppName  string    `json:"app_name"`
	Function string    `json:"function"`
	Schedule string    `json:"schedule"` // Cron expression
	Active   bool      `json:"active"`
	Created  time.Time `json:"created"`
}

// parseCronExpression validates a cron expression
func parseCronExpression(schedule string) error {
	_, err := cron.ParseStandard(schedule)
	return err
}

// validateCronJob validates a cron job configuration
func validateCronJob(job *CronJobConfig) error {
	if job.AppName == "" {
		return fmt.Errorf("app name is required")
	}

	if job.Function == "" {
		return fmt.Errorf("function name is required")
	}

	if job.Schedule == "" {
		return fmt.Errorf("schedule is required")
	}

	if job.ID == "" {
		return fmt.Errorf("job ID is required")
	}

	if err := parseCronExpression(job.Schedule); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	return nil
}

// addCronJob adds a cron job to the scheduler
func (d *denoImpl) addCronJob(job *CronJobConfig) error {
	if err := validateCronJob(job); err != nil {
		return err
	}

	_, err := d.cron.AddFunc(job.Schedule, func() {
		// Insert job when cron fires
		err := d.insertCronJob(job.AppName, job.Function, "{}")
		if err != nil {
			// Log error but don't crash the cron scheduler
			// In a real implementation, this would use proper logging
		}
	})

	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	return nil
}

// removeCronJob removes a cron job from the scheduler
func (d *denoImpl) removeCronJob(jobID string) error {
	// This would require tracking cron entry IDs
	// For now, this is a placeholder
	return nil
}

// listCronJobs returns all active cron jobs
func (d *denoImpl) listCronJobs() ([]*CronJobConfig, error) {
	// This would return the list of configured cron jobs
	// For now, this is a placeholder
	return []*CronJobConfig{}, nil
}

// startCronScheduler starts the cron scheduler
func (d *denoImpl) startCronScheduler() {
	if d.cron != nil {
		d.cron.Start()
	}
}

// stopCronScheduler stops the cron scheduler
func (d *denoImpl) stopCronScheduler() {
	if d.cron != nil {
		d.cron.Stop()
	}
}
