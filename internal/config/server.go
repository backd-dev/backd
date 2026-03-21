package config

import (
	"strconv"
	"time"
)

// ServerConfig represents all runtime environment variables
type ServerConfig struct {
	EncryptionKey     string        `json:"encryption_key"`
	DatabaseURL       string        `json:"database_url"`
	APIPort           int           `json:"api_port"`
	FunctionsPort     int           `json:"functions_port"`
	MetricsPort       int           `json:"metrics_port"`
	InternalPort      int           `json:"internal_port"`
	FunctionsURL      string        `json:"functions_url"`
	LogLevel          string        `json:"log_level"`
	LogFormat         string        `json:"log_format"`
	DenoMinWorkers    int           `json:"deno_min_workers"`
	DenoMaxWorkers    int           `json:"deno_max_workers"`
	DenoIdleTimeout   time.Duration `json:"deno_idle_timeout"`
	JobPollInterval   time.Duration `json:"job_poll_interval"`
}

// LoadServerConfig creates ServerConfig from environment variables provided by commandkit
func LoadServerConfig(env map[string]string) *ServerConfig {
	config := &ServerConfig{
		EncryptionKey:   env["BACKD_ENCRYPTION_KEY"],
		DatabaseURL:     env["DATABASE_URL"],
		APIPort:         getPort(env["BACKD_API_PORT"], 8080),
		FunctionsPort:   getPort(env["BACKD_FUNCTIONS_PORT"], 8081),
		MetricsPort:     getPort(env["BACKD_METRICS_PORT"], 9090),
		InternalPort:    getPort(env["BACKD_INTERNAL_PORT"], 9191),
		FunctionsURL:    getEnvWithDefault(env, "BACKD_FUNCTIONS_URL", "http://functions:8081"),
		LogLevel:        getEnvWithDefault(env, "BACKD_LOG_LEVEL", "info"),
		LogFormat:       getEnvWithDefault(env, "BACKD_LOG_FORMAT", "json"),
		DenoMinWorkers:  getIntWithDefault(env, "BACKD_DENO_MIN_WORKERS", 2),
		DenoMaxWorkers:  getIntWithDefault(env, "BACKD_DENO_MAX_WORKERS", 10),
		DenoIdleTimeout: getDuration(env, "BACKD_DENO_IDLE_TIMEOUT", 5*time.Minute),
		JobPollInterval: getDuration(env, "BACKD_JOB_POLL_INTERVAL", time.Second),
	}
	return config
}

// getPort parses a port number from string with default
func getPort(value string, defaultPort int) int {
	if value == "" {
		return defaultPort
	}
	if port, err := strconv.Atoi(value); err == nil && port > 0 && port <= 65535 {
		return port
	}
	return defaultPort
}

// getEnvWithDefault returns environment value or default
func getEnvWithDefault(env map[string]string, key, defaultValue string) string {
	if value, ok := env[key]; ok && value != "" {
		return value
	}
	return defaultValue
}

// getIntWithDefault parses integer from environment with default
func getIntWithDefault(env map[string]string, key string, defaultValue int) int {
	value := env[key]
	if value == "" {
		return defaultValue
	}
	if result, err := strconv.Atoi(value); err == nil && result >= 0 {
		return result
	}
	return defaultValue
}

// getDuration parses duration from environment with default
func getDuration(env map[string]string, key string, defaultValue time.Duration) time.Duration {
	value := env[key]
	if value == "" {
		return defaultValue
	}
	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}
	return defaultValue
}
