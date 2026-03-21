package metrics

// Metrics interface defines the contract for metrics collection
// This will be fully implemented in milestone 11
type Metrics interface {
	// RecordRequest records an HTTP request with method, path, and status
	RecordRequest(method, path, status string)
	
	// RecordFunctionCall records a function invocation with app name, function name, and success status
	RecordFunctionCall(app, fn string, success bool)
	
	// RecordStorageOperation records a storage operation with app name, operation type, and success status
	RecordStorageOperation(app, operation string, success bool)
}
