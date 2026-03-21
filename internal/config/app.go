package config

import "time"

// AppConfig represents a complete application configuration
type AppConfig struct {
	Name        string                    `yaml:"name"`
	Description string                    `yaml:"description"`
	Database    DatabaseConfig            `yaml:"database"`
	Auth        AuthConfig                `yaml:"auth"`
	Keys        KeysConfig                `yaml:"keys"`
	Secrets     map[string]string         `yaml:"secrets"`
	Storage     *StorageConfig            `yaml:"storage"` // nil = disabled
	Jobs        JobsConfig                `yaml:"jobs"`
	Cron        []CronEntry               `yaml:"cron"`
	Policies    map[string]TablePolicies  `yaml:"policies"`
}

// DatabaseConfig defines database connection settings
type DatabaseConfig struct {
	DSN             string        `yaml:"dsn"`
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	SSLMode         string        `yaml:"ssl_mode"`
	MaxConnections  int           `yaml:"max_connections"`
	MinConnections  int           `yaml:"min_connections"`
	ConnectTimeout  time.Duration `yaml:"connect_timeout"`
}

// AuthConfig defines authentication settings
type AuthConfig struct {
	Domain            string        `yaml:"domain"`
	SessionExpiry     time.Duration `yaml:"session_expiry"`
	AllowRegistration bool          `yaml:"allow_registration"`
}

// KeysConfig defines API key settings
type KeysConfig struct {
	PublishableKey string `yaml:"publishable_key"`
}

// StorageConfig defines S3-compatible storage settings
type StorageConfig struct {
	Endpoint        string        `yaml:"endpoint"`
	Bucket          string        `yaml:"bucket"`
	Region          string        `yaml:"region"`
	AccessKeyID     string        `yaml:"access_key_id"`
	SecretAccessKey string        `yaml:"secret_access_key"`
	PresignExpiry   time.Duration `yaml:"presign_expiry"`
}

// JobsConfig defines background job settings
type JobsConfig struct {
	MaxAttempts int                   `yaml:"max_attempts"`
	Timeout     time.Duration         `yaml:"timeout"`
	Custom      map[string]JobOverride `yaml:"custom"`
}

// JobOverride defines per-function job settings
type JobOverride struct {
	MaxAttempts int           `yaml:"max_attempts"`
	Timeout     time.Duration `yaml:"timeout"`
}

// CronEntry defines a scheduled job
type CronEntry struct {
	Schedule string         `yaml:"schedule"`
	Function string         `yaml:"function"`
	Payload  map[string]any `yaml:"payload"`
}

// TablePolicies defines RLS policies for a table
type TablePolicies map[string]PolicyEntry

// PolicyEntry defines a single RLS policy
type PolicyEntry struct {
	Expression string            `yaml:"expression"`
	Check      string            `yaml:"check"`
	Columns    []string          `yaml:"columns"`
	Defaults   map[string]string `yaml:"defaults"`
	Soft       string            `yaml:"soft"`
}

// ApplyDefaults sets default values for DatabaseConfig
func (c *DatabaseConfig) ApplyDefaults() {
	if c.Port == 0 {
		c.Port = 5432
	}
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}
	if c.MaxConnections == 0 {
		c.MaxConnections = 10
	}
	if c.MinConnections == 0 {
		c.MinConnections = 2
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 5 * time.Second
	}
}

// ApplyDefaults sets default values for AuthConfig
func (c *AuthConfig) ApplyDefaults() {
	if c.SessionExpiry == 0 {
		c.SessionExpiry = 24 * time.Hour
	}
}

// ApplyDefaults sets default values for StorageConfig
func (c *StorageConfig) ApplyDefaults() {
	if c.PresignExpiry == 0 {
		c.PresignExpiry = time.Hour
	}
}

// ApplyDefaults sets default values for JobsConfig
func (c *JobsConfig) ApplyDefaults() {
	if c.MaxAttempts == 0 {
		c.MaxAttempts = 3
	}
	if c.Timeout == 0 {
		c.Timeout = 15 * time.Second
	}
}

// ApplyDefaults sets default values for AppConfig
func (c *AppConfig) ApplyDefaults() {
	c.Database.ApplyDefaults()
	c.Auth.ApplyDefaults()
	if c.Storage != nil {
		c.Storage.ApplyDefaults()
	}
	c.Jobs.ApplyDefaults()
}
