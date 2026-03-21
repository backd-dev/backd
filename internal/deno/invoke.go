package deno

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/backd-dev/backd/internal/db"
)

// FunctionRequest represents a function invocation request
type FunctionRequest struct {
	ID       string            `json:"id"`
	App      string            `json:"app"`
	Function string            `json:"function"`
	Method   string            `json:"method"`
	Headers  map[string]string `json:"headers"`
	Body     string            `json:"body"`
	Params   map[string]any    `json:"params"`
	Timeout  int64             `json:"timeout"`
}

// FunctionResponse represents a function invocation response
type FunctionResponse struct {
	ID      string            `json:"id"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	Error   string            `json:"error,omitempty"`
}

// invokeViaSocket sends a request to a runner via Unix socket
func (d *denoImpl) invokeViaSocket(ctx context.Context, runner *Runner, appName, fnName string, input []byte) ([]byte, error) {
	// Create Unix socket connection with timeout
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "unix", runner.SocketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to runner socket: %w", err)
	}
	defer conn.Close()

	// Set read/write deadlines
	if err := conn.SetDeadline(time.Now().Add(d.config.WorkerTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set socket deadline: %w", err)
	}

	// Create request JSON
	request := FunctionRequest{
		ID:       db.NewXID(),
		App:      appName,
		Function: fnName,
		Method:   "POST",
		Headers:  make(map[string]string),
		Body:     string(input),
		Params:   make(map[string]any),
		Timeout:  d.config.WorkerTimeout.Milliseconds(),
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request with newline delimiter
	if _, err := conn.Write(append(requestJSON, '\n')); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	response, err := readFunctionResponse(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error in response
	if response.Error != "" {
		return nil, fmt.Errorf("function execution failed: %s", response.Error)
	}

	// Check status code
	if response.Status >= 400 {
		return nil, fmt.Errorf("function returned error status: %d", response.Status)
	}

	return []byte(response.Body), nil
}

// readFunctionResponse reads a function response from a connection
func readFunctionResponse(conn net.Conn) (*FunctionResponse, error) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response line: %w", err)
	}

	var response FunctionResponse
	if err := json.Unmarshal([]byte(line[:len(line)-1]), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// sendResponse sends a response back to the caller
func sendResponse(conn net.Conn, response *FunctionResponse) error {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if _, err := conn.Write(append(responseJSON, '\n')); err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}

	return nil
}

// handleRequest handles an incoming request from the Unix socket
func handleRequest(conn net.Conn, d *denoImpl) error {
	// Read request
	request, err := readFunctionRequest(conn)
	if err != nil {
		return fmt.Errorf("failed to read request: %w", err)
	}

	// Create response
	response := &FunctionResponse{
		ID:      request.ID,
		Status:  200,
		Headers: make(map[string]string),
		Body:    "",
	}

	// Execute the function
	output, err := d.InvokeFunction(context.Background(), request.App, request.Function, []byte(request.Body))
	if err != nil {
		response.Status = 500
		response.Error = err.Error()
	} else {
		response.Body = string(output)
	}

	// Send response
	return sendResponse(conn, response)
}

// readFunctionRequest reads a function request from a connection
func readFunctionRequest(conn net.Conn) (*FunctionRequest, error) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read request line: %w", err)
	}

	var request FunctionRequest
	if err := json.Unmarshal([]byte(line[:len(line)-1]), &request); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	return &request, nil
}
