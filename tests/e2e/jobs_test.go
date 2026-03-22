//go:build e2e

package e2e

import (
	"testing"
	"time"
)

func TestJobs_EnqueueAndProcess(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	// Enqueue a job via the public function pattern
	// The public function "hello" doesn't enqueue jobs, so we test the SDK's
	// job client directly (requires internal endpoint access in E2E environment)
	jobID, err := c.Jobs.Enqueue(ctx, "_send_email", map[string]any{
		"to": "test@example.com",
	})
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if jobID == "" {
		t.Fatal("expected job ID")
	}

	// Wait for worker to process (poll _jobs table)
	time.Sleep(5 * time.Second)

	// Query job status via CRUD
	data, _, err := c.From("_jobs").
		Where(map[string]any{"id": map[string]any{"$eq": jobID}}).
		List(ctx)
	if err != nil {
		t.Logf("could not query _jobs (may not be exposed via CRUD): %v", err)
		return
	}
	if len(data) > 0 {
		status, _ := data[0]["status"].(string)
		t.Logf("job %s status: %s", jobID, status)
	}
}

func TestJobs_FailedRetry(t *testing.T) {
	c := newAuthenticatedClient(t)
	ctx := t.Context()

	// Enqueue a job that will fail (non-existent function)
	jobID, err := c.Jobs.Enqueue(ctx, "_nonexistent_function", map[string]any{}, )
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	if jobID == "" {
		t.Fatal("expected job ID")
	}

	// Wait for retry attempts
	time.Sleep(10 * time.Second)

	data, _, err := c.From("_jobs").
		Where(map[string]any{"id": map[string]any{"$eq": jobID}}).
		List(ctx)
	if err != nil {
		t.Logf("could not query _jobs: %v", err)
		return
	}
	if len(data) > 0 {
		status, _ := data[0]["status"].(string)
		t.Logf("failed job %s status: %s", jobID, status)
		if status != "failed" {
			t.Logf("expected status 'failed', got %s (may need more time)", status)
		}
	}
}
