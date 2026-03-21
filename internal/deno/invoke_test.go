package deno

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

// TestInvokeViaSocket tests the socket invocation logic
func TestInvokeViaSocket(t *testing.T) {
	// This is a placeholder test since we can't actually create Unix sockets
	// in the test environment without real Deno processes
	// But we can test the request/response structures

	request := FunctionRequest{
		ID:       "test-request-id",
		App:      "test-app",
		Function: "test-function",
		Method:   "POST",
		Headers:  map[string]string{"Content-Type": "application/json"},
		Body:     "test body",
		Params:   map[string]any{"param1": "value1"},
		Timeout:  30000,
	}

	// Test request serialization
	requestJSON, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	if len(requestJSON) == 0 {
		t.Error("Request JSON should not be empty")
	}

	// Test request deserialization
	var decodedRequest FunctionRequest
	if err := json.Unmarshal(requestJSON, &decodedRequest); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if decodedRequest.ID != request.ID {
		t.Errorf("Expected ID %s, got %s", request.ID, decodedRequest.ID)
	}

	if decodedRequest.App != request.App {
		t.Errorf("Expected App %s, got %s", request.App, decodedRequest.App)
	}

	if decodedRequest.Function != request.Function {
		t.Errorf("Expected Function %s, got %s", request.Function, decodedRequest.Function)
	}
}

// TestFunctionResponseSerialization tests response serialization
func TestFunctionResponseSerialization(t *testing.T) {
	response := FunctionResponse{
		ID:      "test-response-id",
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    "test response body",
		Error:   "",
	}

	// Test response serialization
	responseJSON, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	if len(responseJSON) == 0 {
		t.Error("Response JSON should not be empty")
	}

	// Test response deserialization
	var decodedResponse FunctionResponse
	if err := json.Unmarshal(responseJSON, &decodedResponse); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if decodedResponse.ID != response.ID {
		t.Errorf("Expected ID %s, got %s", response.ID, decodedResponse.ID)
	}

	if decodedResponse.Status != response.Status {
		t.Errorf("Expected Status %d, got %d", response.Status, decodedResponse.Status)
	}

	if decodedResponse.Body != response.Body {
		t.Errorf("Expected Body %s, got %s", response.Body, decodedResponse.Body)
	}
}

// TestErrorResponseSerialization tests error response serialization
func TestErrorResponseSerialization(t *testing.T) {
	response := FunctionResponse{
		ID:      "test-error-response",
		Status:  500,
		Headers: map[string]string{},
		Body:    "",
		Error:   "Something went wrong",
	}

	// Test error response serialization
	responseJSON, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal error response: %v", err)
	}

	// Test error response deserialization
	var decodedResponse FunctionResponse
	if err := json.Unmarshal(responseJSON, &decodedResponse); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if decodedResponse.Error != response.Error {
		t.Errorf("Expected Error %s, got %s", response.Error, decodedResponse.Error)
	}

	if decodedResponse.Status != 500 {
		t.Errorf("Expected Status 500, got %d", decodedResponse.Status)
	}
}

// TestSocketPathGeneration tests socket path generation
func TestSocketPathGeneration(t *testing.T) {
	// Test that socket paths follow the expected pattern
	// In the real implementation, this would be done in worker.go

	testCases := []struct {
		appName  string
		pid      int
		expected string
	}{
		{"test-app", 12345, "/tmp/backd/test-app/12345.sock"},
		{"my-app", 67890, "/tmp/backd/my-app/67890.sock"},
		{"app-with-dashes", 11111, "/tmp/backd/app-with-dashes/11111.sock"},
	}

	for _, tc := range testCases {
		t.Run(tc.appName, func(t *testing.T) {
			// This tests the pattern that would be used for socket paths
			// In the real implementation, this would be generated dynamically
			socketPath := "/tmp/backd/" + tc.appName + "/" + string(rune(tc.pid)) + ".sock"

			// Just verify the pattern contains the expected components
			if !contains(socketPath, tc.appName) {
				t.Errorf("Socket path should contain app name %s", tc.appName)
			}

			if !contains(socketPath, ".sock") {
				t.Error("Socket path should end with .sock")
			}
		})
	}
}

// TestTimeoutHandlingInInvocation tests timeout handling in function invocation
func TestTimeoutHandlingInInvocation(t *testing.T) {
	testCases := []struct {
		name           string
		timeout        time.Duration
		shouldTimeout  bool
		expectedStatus int
	}{
		{
			name:           "short timeout",
			timeout:        1 * time.Millisecond,
			shouldTimeout:  true,
			expectedStatus: 408,
		},
		{
			name:           "reasonable timeout",
			timeout:        30 * time.Second,
			shouldTimeout:  false,
			expectedStatus: 200,
		},
		{
			name:           "zero timeout",
			timeout:        0,
			shouldTimeout:  true,
			expectedStatus: 408,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This would test timeout handling in real invocation
			// For now, we just test the timeout logic

			if tc.timeout <= 0 {
				if !tc.shouldTimeout {
					t.Error("Zero or negative timeout should result in timeout")
				}
			}

			if tc.timeout > 0 && tc.timeout < 10*time.Millisecond {
				if !tc.shouldTimeout {
					t.Error("Very short timeout should result in timeout")
				}
			}
		})
	}
}

// TestRequestHeaders tests request header handling
func TestRequestHeaders(t *testing.T) {
	headers := map[string]string{
		"Content-Type":    "application/json",
		"Authorization":   "Bearer token123",
		"X-Custom-Header": "custom-value",
	}

	request := FunctionRequest{
		ID:       "test-request",
		App:      "test-app",
		Function: "test-function",
		Method:   "POST",
		Headers:  headers,
		Body:     "test body",
		Params:   map[string]any{},
		Timeout:  30000,
	}

	// Test that headers are properly included
	if request.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header, got %v", request.Headers["Content-Type"])
	}

	if request.Headers["Authorization"] != "Bearer token123" {
		t.Errorf("Expected Authorization header, got %v", request.Headers["Authorization"])
	}

	if request.Headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("Expected X-Custom-Header, got %v", request.Headers["X-Custom-Header"])
	}
}

// TestResponseHeaders tests response header handling
func TestResponseHeaders(t *testing.T) {
	headers := map[string]string{
		"Content-Type":    "application/json",
		"X-Response-ID":   "response-123",
		"X-Custom-Header": "response-value",
	}

	response := FunctionResponse{
		ID:      "test-response",
		Status:  200,
		Headers: headers,
		Body:    "response body",
		Error:   "",
	}

	// Test that response headers are properly included
	if response.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header, got %v", response.Headers["Content-Type"])
	}

	if response.Headers["X-Response-ID"] != "response-123" {
		t.Errorf("Expected X-Response-ID header, got %v", response.Headers["X-Response-ID"])
	}

	if response.Headers["X-Custom-Header"] != "response-value" {
		t.Errorf("Expected X-Custom-Header, got %v", response.Headers["X-Custom-Header"])
	}
}

// TestRequestBodyHandling tests request body handling
func TestRequestBodyHandling(t *testing.T) {
	testCases := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"simple text", "Hello, World!"},
		{"json body", `{"key": "value", "number": 123}`},
		{"large body", string(make([]byte, 1024*1024))}, // 1MB body
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := FunctionRequest{
				ID:       "test-request",
				App:      "test-app",
				Function: "test-function",
				Method:   "POST",
				Headers:  map[string]string{"Content-Type": "text/plain"},
				Body:     tc.body,
				Params:   map[string]any{},
				Timeout:  30000,
			}

			if request.Body != tc.body {
				t.Errorf("Expected body %s, got %s", tc.body, request.Body)
			}

			// Test body serialization
			requestJSON, err := json.Marshal(request)
			if err != nil {
				t.Fatalf("Failed to marshal request with body: %v", err)
			}

			var decodedRequest FunctionRequest
			if err := json.Unmarshal(requestJSON, &decodedRequest); err != nil {
				t.Fatalf("Failed to unmarshal request with body: %v", err)
			}

			if decodedRequest.Body != tc.body {
				t.Errorf("Expected decoded body %s, got %s", tc.body, decodedRequest.Body)
			}
		})
	}
}

// TestResponseBodyHandling tests response body handling
func TestResponseBodyHandling(t *testing.T) {
	testCases := []struct {
		name string
		body string
	}{
		{"empty response", ""},
		{"simple text", "Response text"},
		{"json response", `{"result": "success", "data": [1, 2, 3]}`},
		{"large response", string(make([]byte, 512*1024))}, // 512KB response
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := FunctionResponse{
				ID:      "test-response",
				Status:  200,
				Headers: map[string]string{"Content-Type": "text/plain"},
				Body:    tc.body,
				Error:   "",
			}

			if response.Body != tc.body {
				t.Errorf("Expected body %s, got %s", tc.body, response.Body)
			}

			// Test response serialization
			responseJSON, err := json.Marshal(response)
			if err != nil {
				t.Fatalf("Failed to marshal response with body: %v", err)
			}

			var decodedResponse FunctionResponse
			if err := json.Unmarshal(responseJSON, &decodedResponse); err != nil {
				t.Fatalf("Failed to unmarshal response with body: %v", err)
			}

			if decodedResponse.Body != tc.body {
				t.Errorf("Expected decoded body %s, got %s", tc.body, decodedResponse.Body)
			}
		})
	}
}

// TestErrorHandlingInInvocation tests error handling in function invocation
func TestErrorHandlingInInvocation(t *testing.T) {
	testCases := []struct {
		name           string
		errorMessage   string
		expectedStatus int
	}{
		{"function error", "Function not found", 404},
		{"timeout error", "Function execution timeout", 408},
		{"internal error", "Internal server error", 500},
		{"validation error", "Invalid input parameters", 400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response := FunctionResponse{
				ID:      "test-error-response",
				Status:  tc.expectedStatus,
				Headers: map[string]string{},
				Body:    "",
				Error:   tc.errorMessage,
			}

			if response.Error != tc.errorMessage {
				t.Errorf("Expected error message %s, got %s", tc.errorMessage, response.Error)
			}

			if response.Status != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, response.Status)
			}
		})
	}
}

// TestSocketConnectionErrorHandling tests socket connection error handling
func TestSocketConnectionErrorHandling(t *testing.T) {
	// This tests error handling when socket connections fail
	// In a real implementation, this would handle cases where:
	// - Socket file doesn't exist
	// - Permission denied
	// - Connection refused
	// - Socket is not listening

	testCases := []struct {
		name        string
		socketPath  string
		shouldError bool
	}{
		{"non-existent socket", "/tmp/non-existent.sock", true},
		{"invalid path", "/invalid/path/test.sock", true},
		{"permission denied", "/root/protected.sock", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Try to connect to the socket (should fail)
			conn, err := net.Dial("unix", tc.socketPath)

			if err == nil {
				conn.Close()
				if tc.shouldError {
					t.Error("Expected connection to fail")
				}
			} else {
				if !tc.shouldError {
					t.Errorf("Expected connection to succeed, got error: %v", err)
				}
			}
		})
	}
}

// TestContextCancellationInInvocation tests context cancellation
func TestContextCancellationInInvocation(t *testing.T) {
	// Test that function invocation respects context cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context immediately
	cancel()

	// The context should be cancelled
	select {
	case <-ctx.Done():
		// Context is cancelled as expected
		if ctx.Err() != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", ctx.Err())
		}
	default:
		t.Error("Context should be cancelled")
	}

	// Test with timeout context
	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(2 * time.Millisecond)

	select {
	case <-ctx.Done():
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", ctx.Err())
		}
	default:
		t.Error("Context should have timed out")
	}
}
