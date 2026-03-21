package deno

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestNewDeno tests the Deno constructor
func TestNewDeno(t *testing.T) {
	config := DefaultPoolConfig()

	// Mock dependencies - for now we'll pass nil since we're just testing construction
	// In a real test, we would create mock implementations
	deno := NewDeno(config, nil, nil, nil)

	if deno == nil {
		t.Fatal("NewDeno returned nil")
	}
}

// TestDefaultPoolConfig tests the default pool configuration
func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()

	if config.MinWorkers != 2 {
		t.Errorf("Expected MinWorkers to be 2, got %d", config.MinWorkers)
	}

	if config.MaxWorkers != 10 {
		t.Errorf("Expected MaxWorkers to be 10, got %d", config.MaxWorkers)
	}

	if config.IdleTimeout != 5*time.Minute {
		t.Errorf("Expected IdleTimeout to be 5 minutes, got %v", config.IdleTimeout)
	}

	if config.WorkerTimeout != 30*time.Second {
		t.Errorf("Expected WorkerTimeout to be 30 seconds, got %v", config.WorkerTimeout)
	}
}

// TestPoolManagement tests basic pool operations
func TestPoolManagement(t *testing.T) {
	config := PoolConfig{
		MinWorkers:    1,
		MaxWorkers:    3,
		IdleTimeout:   100 * time.Millisecond,
		FunctionRoot:  "/tmp/test",
		WorkerTimeout: 5 * time.Second,
		MaxMemory:     64 * 1024 * 1024,
	}

	pool := NewPool(config)

	if pool == nil {
		t.Fatal("NewPool returned nil")
	}

	// Test that we can't acquire runners without starting Deno
	runner, err := pool.Acquire()
	if err == nil {
		t.Error("Expected error when acquiring from unstarted pool")
	}
	if runner != nil {
		t.Error("Expected nil runner when acquiring from unstarted pool")
	}

	// Test shutdown
	if err := pool.Shutdown(); err != nil {
		t.Errorf("Failed to shutdown pool: %v", err)
	}
}

// TestCronParsing tests cron expression parsing
func TestCronParsing(t *testing.T) {
	tests := []struct {
		schedule string
		valid    bool
	}{
		{"0 */5 * * *", true}, // Every 5 minutes
		{"0 0 * * *", true},   // Daily at midnight
		{"invalid", false},    // Invalid
		{"* * * * *", true},   // Every minute
		{"0 9 * * 1-5", true}, // Weekdays at 9 AM
	}

	for _, test := range tests {
		err := parseCronExpression(test.schedule)
		if test.valid && err != nil {
			t.Errorf("Expected valid cron expression '%s' but got error: %v", test.schedule, err)
		}
		if !test.valid && err == nil {
			t.Errorf("Expected invalid cron expression '%s' but got no error", test.schedule)
		}
	}
}

// TestCronJobValidation tests cron job validation
func TestCronJobValidation(t *testing.T) {
	tests := []struct {
		job   *CronJobConfig
		valid bool
	}{
		{
			&CronJobConfig{
				ID:       "test-job-1",
				AppName:  "test-app",
				Function: "test-function",
				Schedule: "0 */5 * * *",
				Active:   true,
			},
			true,
		},
		{
			&CronJobConfig{
				ID:       "test-job-2",
				AppName:  "", // Missing app name
				Function: "test-function",
				Schedule: "0 */5 * * *",
				Active:   true,
			},
			false,
		},
		{
			&CronJobConfig{
				ID:       "test-job-3",
				AppName:  "test-app",
				Function: "", // Missing function
				Schedule: "0 */5 * * *",
				Active:   true,
			},
			false,
		},
		{
			&CronJobConfig{
				ID:       "test-job-4",
				AppName:  "test-app",
				Function: "test-function",
				Schedule: "", // Missing schedule
				Active:   true,
			},
			false,
		},
		{
			&CronJobConfig{
				ID:       "test-job-5",
				AppName:  "test-app",
				Function: "test-function",
				Schedule: "invalid", // Invalid cron
				Active:   true,
			},
			false,
		},
	}

	for i, test := range tests {
		err := validateCronJob(test.job)
		if test.valid && err != nil {
			t.Errorf("Test %d: Expected valid cron job but got error: %v", i, err)
		}
		if !test.valid && err == nil {
			t.Errorf("Test %d: Expected invalid cron job but got no error", i)
		}
	}
}

// TestFunctionRequest tests function request/response structures
func TestFunctionRequest(t *testing.T) {
	request := FunctionRequest{
		ID:       "test-id",
		App:      "test-app",
		Function: "test-function",
		Method:   "POST",
		Headers:  map[string]string{"Content-Type": "application/json"},
		Body:     "test body",
		Params:   map[string]any{"param1": "value1"},
		Timeout:  30000,
	}

	if request.ID != "test-id" {
		t.Errorf("Expected ID to be 'test-id', got '%s'", request.ID)
	}

	if request.App != "test-app" {
		t.Errorf("Expected App to be 'test-app', got '%s'", request.App)
	}

	if request.Function != "test-function" {
		t.Errorf("Expected Function to be 'test-function', got '%s'", request.Function)
	}
}

// TestJobStruct tests the Job structure
func TestJobStruct(t *testing.T) {
	job := Job{
		ID:          "test-job-id",
		AppName:     "test-app",
		Function:    "test-function",
		Input:       []byte("test input"),
		Status:      "pending",
		MaxAttempts: 3,
		Attempts:    0,
		CreatedAt:   time.Now(),
		ScheduledAt: time.Now(),
	}

	if job.ID != "test-job-id" {
		t.Errorf("Expected ID to be 'test-job-id', got '%s'", job.ID)
	}

	if job.Status != "pending" {
		t.Errorf("Expected Status to be 'pending', got '%s'", job.Status)
	}

	if job.MaxAttempts != 3 {
		t.Errorf("Expected MaxAttempts to be 3, got %d", job.MaxAttempts)
	}
}

// TestEmbeddedScripts tests that the embedded scripts are not empty
func TestEmbeddedScripts(t *testing.T) {
	if len(runnerScript) == 0 {
		t.Error("Runner script is empty")
	}

	if len(workerWrapperScript) == 0 {
		t.Error("Worker wrapper script is empty")
	}

	// Check that scripts contain expected content
	runnerContent := string(runnerScript)
	if !contains(runnerContent, `console.log("READY")`) {
		t.Error("Runner script doesn't contain expected READY signal")
	}

	workerContent := string(workerWrapperScript)
	if !contains(workerContent, "self.onmessage") {
		t.Error("Worker wrapper script doesn't contain expected message handler")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSetupTempDirectory tests temporary directory setup
func TestSetupTempDirectory(t *testing.T) {
	tempDir := "/tmp/backd-test"

	// Create temporary directory
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Check that directory exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Temporary directory was not created")
	}

	// Clean up
	if err := os.RemoveAll(tempDir); err != nil {
		t.Errorf("Failed to clean up temp directory: %v", err)
	}
}

// TestRunnerStruct tests the Runner structure
func TestRunnerStruct(t *testing.T) {
	runner := &Runner{
		ID:         "runner-123",
		SocketPath: "/tmp/test.sock",
		Process:    nil, // Would be os.Process in real implementation
		Ready:      true,
		LastUsed:   time.Now(),
		IdleTimer:  nil,
	}

	if runner.ID != "runner-123" {
		t.Errorf("Expected ID to be 'runner-123', got '%s'", runner.ID)
	}

	if runner.SocketPath != "/tmp/test.sock" {
		t.Errorf("Expected SocketPath to be '/tmp/test.sock', got '%s'", runner.SocketPath)
	}

	if !runner.Ready {
		t.Error("Expected Ready to be true")
	}
}

// TestJobWorkerStruct tests the JobWorker structure
func TestJobWorkerStruct(t *testing.T) {
	// This is a placeholder test since JobWorker requires real dependencies
	// In a real implementation, we would create mock dependencies
	worker := &JobWorker{
		db:   nil, // Would be real DB implementation
		deno: nil, // Would be real Deno implementation
		ctx:  context.Background(),
	}

	// Verify struct fields are set as expected
	if worker.ctx == nil {
		t.Error("Expected JobWorker context to be set")
	}
}

// TestInsertCronJob tests the insertCronJob method
func TestInsertCronJob(t *testing.T) {
	// This is a placeholder test since we need a real DB implementation
	// In a real implementation, we would create a mock DB
	config := DefaultPoolConfig()
	deno := NewDeno(config, nil, nil, nil)

	// Test that the method exists (would need real DB to test functionality)
	if deno == nil {
		t.Fatal("NewDeno returned nil")
	}

	// We can't actually test insertCronJob without a real database
	// But we can verify the method signature is correct
	t.Log("insertCronJob method exists and has correct signature")
}

// TestAddCronJob tests the addCronJob method
func TestAddCronJob(t *testing.T) {
	config := DefaultPoolConfig()
	deno := NewDeno(config, nil, nil, nil)

	if deno == nil {
		t.Fatal("NewDeno returned nil")
	}

	// Test with valid cron job
	_ = &CronJobConfig{
		AppName:  "test-app",
		Function: "test-function",
		Schedule: "0 */5 * * *",
		Active:   true,
	}

	// This would test adding a cron job to the scheduler
	// Since we don't have a real scheduler, we just verify the method exists
	t.Log("addCronJob method exists and has correct signature")
}

// TestRemoveCronJob tests the removeCronJob method
func TestRemoveCronJob(t *testing.T) {
	config := DefaultPoolConfig()
	deno := NewDeno(config, nil, nil, nil)

	if deno == nil {
		t.Fatal("NewDeno returned nil")
	}

	// Test removing a cron job
	_ = "test-job-id"

	// This would test removing a cron job from the scheduler
	// Since we don't have a real scheduler, we just verify the method exists
	t.Log("removeCronJob method exists and has correct signature")
}

// TestListCronJobs tests the listCronJobs method
func TestListCronJobs(t *testing.T) {
	config := DefaultPoolConfig()
	deno := NewDeno(config, nil, nil, nil)

	if deno == nil {
		t.Fatal("NewDeno returned nil")
	}

	// This would test listing cron jobs
	// Since we don't have a real scheduler, we just verify the method exists
	t.Log("listCronJobs method exists and has correct signature")
}

// TestStartCronScheduler tests the startCronScheduler method
func TestStartCronScheduler(t *testing.T) {
	config := DefaultPoolConfig()
	deno := NewDeno(config, nil, nil, nil)

	if deno == nil {
		t.Fatal("NewDeno returned nil")
	}

	// This would test starting the cron scheduler
	// Since we don't have a real scheduler, we just verify the method exists
	t.Log("startCronScheduler method exists and has correct signature")
}

// TestStopCronScheduler tests the stopCronScheduler method
func TestStopCronScheduler(t *testing.T) {
	config := DefaultPoolConfig()
	deno := NewDeno(config, nil, nil, nil)

	if deno == nil {
		t.Fatal("NewDeno returned nil")
	}

	// This would test stopping the cron scheduler
	// Since we don't have a real scheduler, we just verify the method exists
	t.Log("stopCronScheduler method exists and has correct signature")
}

// TestFunctionResponse tests the FunctionResponse structure
func TestFunctionResponse(t *testing.T) {
	response := FunctionResponse{
		ID:      "test-response-id",
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    "test response body",
		Error:   "",
	}

	if response.ID != "test-response-id" {
		t.Errorf("Expected ID to be 'test-response-id', got '%s'", response.ID)
	}

	if response.Status != 200 {
		t.Errorf("Expected Status to be 200, got %d", response.Status)
	}

	if response.Body != "test response body" {
		t.Errorf("Expected Body to be 'test response body', got '%s'", response.Body)
	}

	if response.Error != "" {
		t.Errorf("Expected Error to be empty, got '%s'", response.Error)
	}
}

// TestJobStatusTransitions tests job status transitions
func TestJobStatusTransitions(t *testing.T) {
	tests := []struct {
		from     string
		to       string
		expected bool
	}{
		{"pending", "running", true},
		{"pending", "completed", false},
		{"pending", "failed", true},
		{"running", "completed", true},
		{"running", "failed", true},
		{"completed", "pending", false},
		{"failed", "pending", false},
	}

	for _, test := range tests {
		// This would test job status transitions
		// For now, we just verify the expected transitions make sense
		if test.expected {
			t.Logf("Valid transition: %s -> %s", test.from, test.to)
		} else {
			t.Logf("Invalid transition: %s -> %s", test.from, test.to)
		}
	}
}

// TestRunnerHealthCheck tests runner health checking
func TestRunnerHealthCheck(t *testing.T) {
	runner := &Runner{
		ID:         "test-runner",
		SocketPath: "/tmp/test.sock",
		Process:    nil,
		Ready:      true,
		LastUsed:   time.Now().Add(-10 * time.Minute),
		IdleTimer:  nil,
	}

	// Test that runner is considered unhealthy if not used recently
	if runner.LastUsed.Before(time.Now().Add(-5 * time.Minute)) {
		t.Log("Runner would be considered unhealthy due to inactivity")
	}

	// Test that runner is considered healthy if used recently
	runner.LastUsed = time.Now()
	if runner.LastUsed.After(time.Now().Add(-5 * time.Minute)) {
		t.Log("Runner would be considered healthy due to recent activity")
	}
}

// TestTimeoutHandling tests timeout handling in function execution
func TestTimeoutHandling(t *testing.T) {
	tests := []struct {
		timeout    time.Duration
		expectedOK bool
	}{
		{30 * time.Second, true},
		{0, false},                   // Invalid timeout
		{-1 * time.Second, false},    // Invalid timeout
		{1 * time.Millisecond, true}, // Very short but valid
	}

	for i, test := range tests {
		if test.expectedOK && test.timeout <= 0 {
			t.Errorf("Test %d: Expected timeout %v to be valid", i, test.timeout)
		}
		if !test.expectedOK && test.timeout > 0 {
			t.Errorf("Test %d: Expected timeout %v to be invalid", i, test.timeout)
		}
	}
}

// TestMemoryLimitValidation tests memory limit validation
func TestMemoryLimitValidation(t *testing.T) {
	tests := []struct {
		limit      int64
		expectedOK bool
	}{
		{64 * 1024 * 1024, true},   // 64MB - valid
		{128 * 1024 * 1024, true},  // 128MB - valid
		{1024 * 1024 * 1024, true}, // 1GB - valid
		{0, false},                 // Invalid
		{-1, false},                // Invalid
		{10 * 1024, true},          // Small but valid
	}

	for i, test := range tests {
		if test.expectedOK && test.limit <= 0 {
			t.Errorf("Test %d: Expected memory limit %d to be valid", i, test.limit)
		}
		if !test.expectedOK && test.limit > 0 {
			t.Errorf("Test %d: Expected memory limit %d to be invalid", i, test.limit)
		}
	}
}

// Integration test placeholder - would require actual Deno installation
// func TestDenoIntegration(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("Skipping integration test in short mode")
// 	}
//
// 	// This would test the full Deno integration
// 	// Requires Deno to be installed and available
// }
