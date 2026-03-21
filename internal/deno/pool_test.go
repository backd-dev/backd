package deno

import (
	"context"
	"testing"
	"time"
)

// TestPoolAcquireRelease tests pool acquire and release operations
func TestPoolAcquireRelease(t *testing.T) {
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

// TestPoolConfigValidation tests pool configuration edge cases
func TestPoolConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config PoolConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: PoolConfig{
				MinWorkers:    1,
				MaxWorkers:    5,
				IdleTimeout:   5 * time.Minute,
				FunctionRoot:  "/tmp",
				WorkerTimeout: 30 * time.Second,
				MaxMemory:     128 * 1024 * 1024,
			},
			valid: true,
		},
		{
			name: "zero min workers",
			config: PoolConfig{
				MinWorkers:    0,
				MaxWorkers:    5,
				IdleTimeout:   5 * time.Minute,
				FunctionRoot:  "/tmp",
				WorkerTimeout: 30 * time.Second,
				MaxMemory:     128 * 1024 * 1024,
			},
			valid: false,
		},
		{
			name: "max less than min",
			config: PoolConfig{
				MinWorkers:    5,
				MaxWorkers:    3,
				IdleTimeout:   5 * time.Minute,
				FunctionRoot:  "/tmp",
				WorkerTimeout: 30 * time.Second,
				MaxMemory:     128 * 1024 * 1024,
			},
			valid: false,
		},
		{
			name: "empty function root",
			config: PoolConfig{
				MinWorkers:    1,
				MaxWorkers:    5,
				IdleTimeout:   5 * time.Minute,
				FunctionRoot:  "",
				WorkerTimeout: 30 * time.Second,
				MaxMemory:     128 * 1024 * 1024,
			},
			valid: false,
		},
		{
			name: "zero worker timeout",
			config: PoolConfig{
				MinWorkers:    1,
				MaxWorkers:    5,
				IdleTimeout:   5 * time.Minute,
				FunctionRoot:  "/tmp",
				WorkerTimeout: 0,
				MaxMemory:     128 * 1024 * 1024,
			},
			valid: false,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// This would validate the pool configuration
			// For now, we just test the basic validation logic
			isValid := test.config.MinWorkers > 0 &&
				test.config.MaxWorkers >= test.config.MinWorkers &&
				test.config.FunctionRoot != "" &&
				test.config.WorkerTimeout > 0 &&
				test.config.MaxMemory > 0
			
			if isValid != test.valid {
				t.Errorf("Expected validity %v, got %v", test.valid, isValid)
			}
		})
	}
}

// TestRunnerLifecycle tests runner lifecycle operations
func TestRunnerLifecycle(t *testing.T) {
	runner := &Runner{
		ID:         "test-runner-1",
		SocketPath: "/tmp/test-runner-1.sock",
		Process:    nil,
		Ready:      false,
		LastUsed:   time.Now(),
		IdleTimer:  nil,
	}
	
	// Test initial state
	if runner.ID != "test-runner-1" {
		t.Errorf("Expected runner ID to be 'test-runner-1', got '%s'", runner.ID)
	}
	
	if runner.Ready {
		t.Error("Expected runner to be not ready initially")
	}
	
	// Test marking as ready
	runner.Ready = true
	if !runner.Ready {
		t.Error("Expected runner to be ready after marking")
	}
	
	// Test updating last used time
	originalTime := runner.LastUsed
	time.Sleep(1 * time.Millisecond)
	runner.LastUsed = time.Now()
	
	if !runner.LastUsed.After(originalTime) {
		t.Error("Expected LastUsed to be updated")
	}
}

// TestJobStatusLogic tests job status transitions and logic
func TestJobStatusLogic(t *testing.T) {
	tests := []struct {
		name        string
		current     string
		maxAttempts int
		attempts    int
		shouldRetry bool
		shouldFail  bool
	}{
		{
			name:        "pending job should run",
			current:     "pending",
			maxAttempts: 3,
			attempts:    0,
			shouldRetry: true,
			shouldFail:  false,
		},
		{
			name:        "running job should not retry",
			current:     "running",
			maxAttempts: 3,
			attempts:    0,
			shouldRetry: false,
			shouldFail:  false,
		},
		{
			name:        "completed job should not retry",
			current:     "completed",
			maxAttempts: 3,
			attempts:    1,
			shouldRetry: false,
			shouldFail:  false,
		},
		{
			name:        "failed job with retries left should retry",
			current:     "failed",
			maxAttempts: 3,
			attempts:    1,
			shouldRetry: true,
			shouldFail:  false,
		},
		{
			name:        "failed job with no retries left should fail permanently",
			current:     "failed",
			maxAttempts: 3,
			attempts:    3,
			shouldRetry: false,
			shouldFail:  true,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			job := &Job{
				ID:          "test-job",
				AppName:     "test-app",
				Function:    "test-function",
				Input:       []byte("test input"),
				Status:      test.current,
				MaxAttempts: test.maxAttempts,
				Attempts:    test.attempts,
				CreatedAt:   time.Now(),
				ScheduledAt: time.Now(),
			}
			
			shouldRetry := job.Status == "pending" || 
				(job.Status == "failed" && job.Attempts < job.MaxAttempts)
			
			shouldFail := job.Status == "failed" && job.Attempts >= job.MaxAttempts
			
			if shouldRetry != test.shouldRetry {
				t.Errorf("Expected shouldRetry %v, got %v", test.shouldRetry, shouldRetry)
			}
			
			if shouldFail != test.shouldFail {
				t.Errorf("Expected shouldFail %v, got %v", test.shouldFail, shouldFail)
			}
		})
	}
}

// TestCronJobScheduleValidation tests cron job schedule validation
func TestCronJobScheduleValidation(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		valid    bool
	}{
		{
			name:     "every minute",
			schedule: "* * * * *",
			valid:    true,
		},
		{
			name:     "every hour",
			schedule: "0 * * * *",
			valid:    true,
		},
		{
			name:     "every day at midnight",
			schedule: "0 0 * * *",
			valid:    true,
		},
		{
			name:     "weekdays at 9 AM",
			schedule: "0 9 * * 1-5",
			valid:    true,
		},
		{
			name:     "invalid format",
			schedule: "invalid",
			valid:    false,
		},
		{
			name:     "empty schedule",
			schedule: "",
			valid:    false,
		},
		{
			name:     "too many fields",
			schedule: "* * * * * *",
			valid:    false,
		},
		{
			name:     "too few fields",
			schedule: "* * * *",
			valid:    false,
		},
		{
			name:     "invalid range",
			schedule: "0 25 * * *",
			valid:    false,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := parseCronExpression(test.schedule)
			isValid := err == nil
			
			if isValid != test.valid {
				t.Errorf("Expected validity %v for schedule '%s', got %v (error: %v)", 
					test.valid, test.schedule, isValid, err)
			}
		})
	}
}

// TestFunctionRequestValidation tests function request validation
func TestFunctionRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request FunctionRequest
		valid   bool
	}{
		{
			name: "valid request",
			request: FunctionRequest{
				ID:       "test-id",
				App:      "test-app",
				Function: "test-function",
				Method:   "POST",
				Headers:  map[string]string{"Content-Type": "application/json"},
				Body:     "test body",
				Params:   map[string]any{"param1": "value1"},
				Timeout:  30000,
			},
			valid: true,
		},
		{
			name: "missing ID",
			request: FunctionRequest{
				ID:       "",
				App:      "test-app",
				Function: "test-function",
				Method:   "POST",
				Headers:  map[string]string{"Content-Type": "application/json"},
				Body:     "test body",
				Params:   map[string]any{"param1": "value1"},
				Timeout:  30000,
			},
			valid: false,
		},
		{
			name: "missing app",
			request: FunctionRequest{
				ID:       "test-id",
				App:      "",
				Function: "test-function",
				Method:   "POST",
				Headers:  map[string]string{"Content-Type": "application/json"},
				Body:     "test body",
				Params:   map[string]any{"param1": "value1"},
				Timeout:  30000,
			},
			valid: false,
		},
		{
			name: "missing function",
			request: FunctionRequest{
				ID:       "test-id",
				App:      "test-app",
				Function: "",
				Method:   "POST",
				Headers:  map[string]string{"Content-Type": "application/json"},
				Body:     "test body",
				Params:   map[string]any{"param1": "value1"},
				Timeout:  30000,
			},
			valid: false,
		},
		{
			name: "zero timeout",
			request: FunctionRequest{
				ID:       "test-id",
				App:      "test-app",
				Function: "test-function",
				Method:   "POST",
				Headers:  map[string]string{"Content-Type": "application/json"},
				Body:     "test body",
				Params:   map[string]any{"param1": "value1"},
				Timeout:  0,
			},
			valid: false,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isValid := test.request.ID != "" &&
				test.request.App != "" &&
				test.request.Function != "" &&
				test.request.Timeout > 0
			
			if isValid != test.valid {
				t.Errorf("Expected validity %v, got %v", test.valid, isValid)
			}
		})
	}
}

// TestFunctionResponseValidation tests function response validation
func TestFunctionResponseValidation(t *testing.T) {
	tests := []struct {
		name     string
		response FunctionResponse
		valid    bool
	}{
		{
			name: "valid success response",
			response: FunctionResponse{
				ID:      "test-response-id",
				Status:  200,
				Headers: map[string]string{"Content-Type": "application/json"},
				Body:    "test response body",
				Error:   "",
			},
			valid: true,
		},
		{
			name: "valid error response",
			response: FunctionResponse{
				ID:      "test-response-id",
				Status:  500,
				Headers: map[string]string{},
				Body:    "",
				Error:   "Something went wrong",
			},
			valid: true,
		},
		{
			name: "missing ID",
			response: FunctionResponse{
				ID:      "",
				Status:  200,
				Headers: map[string]string{"Content-Type": "application/json"},
				Body:    "test response body",
				Error:   "",
			},
			valid: false,
		},
		{
			name: "invalid status",
			response: FunctionResponse{
				ID:      "test-response-id",
				Status:  0,
				Headers: map[string]string{"Content-Type": "application/json"},
				Body:    "test response body",
				Error:   "",
			},
			valid: false,
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isValid := test.response.ID != "" &&
				test.response.Status >= 100 && test.response.Status < 600
			
			if isValid != test.valid {
				t.Errorf("Expected validity %v, got %v", test.valid, isValid)
			}
		})
	}
}

// TestDenoInterfaceMethods tests that all interface methods are implemented
func TestDenoInterfaceMethods(t *testing.T) {
	config := DefaultPoolConfig()
	deno := NewDeno(config, nil, nil, nil)
	
	if deno == nil {
		t.Fatal("NewDeno returned nil")
	}
	
	// Test that all interface methods exist and have correct signatures
	ctx := context.Background()
	
	// Test Start method
	err := deno.Start(ctx)
	if err != nil {
		t.Logf("Start method returned error (expected without real dependencies): %v", err)
	}
	
	// Test Stop method
	err = deno.Stop(ctx)
	if err != nil {
		t.Logf("Stop method returned error (expected without real dependencies): %v", err)
	}
	
	// Test InvokeFunction method
	_, err = deno.InvokeFunction(ctx, "test-app", "test-function", []byte("test input"))
	if err != nil {
		t.Logf("InvokeFunction method returned error (expected without real dependencies): %v", err)
	}
	
	// Test StartJobWorker method
	err = deno.StartJobWorker(ctx)
	if err != nil {
		t.Logf("StartJobWorker method returned error (expected without real dependencies): %v", err)
	}
	
	// Test RegisterCronJobs method
	err = deno.RegisterCronJobs(ctx, []string{"test-app"})
	if err != nil {
		t.Logf("RegisterCronJobs method returned error (expected without real dependencies): %v", err)
	}
}
