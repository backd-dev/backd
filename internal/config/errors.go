package config

import (
	"fmt"
	"strings"
)

// Status represents validation status
type Status int

const (
	StatusOK Status = iota
	StatusWarn
	StatusError
)

// String returns string representation of Status
func (s Status) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusWarn:
		return "WARN"
	case StatusError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ValidationIssue represents a single validation issue
type ValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
	IsError bool   `json:"is_error"`
}

// ValidationResult contains validation results for a config
type ValidationResult struct {
	Status  Status            `json:"status"`
	Issues  []ValidationIssue `json:"issues"`
	AppName string            `json:"app_name,omitempty"`
}

// AddError adds an error issue to the result
func (r *ValidationResult) AddError(field, message, hint string) {
	r.Issues = append(r.Issues, ValidationIssue{
		Field:   field,
		Message: message,
		Hint:    hint,
		IsError: true,
	})
	if r.Status < StatusError {
		r.Status = StatusError
	}
}

// AddWarning adds a warning issue to the result
func (r *ValidationResult) AddWarning(field, message, hint string) {
	r.Issues = append(r.Issues, ValidationIssue{
		Field:   field,
		Message: message,
		Hint:    hint,
		IsError: false,
	})
	if r.Status < StatusWarn {
		r.Status = StatusWarn
	}
}

// HasErrors returns true if there are error-level issues
func (r *ValidationResult) HasErrors() bool {
	return r.Status == StatusError && len(r.Issues) > 0
}

// HasWarnings returns true if there are warning-level issues
func (r *ValidationResult) HasWarnings() bool {
	return r.Status == StatusWarn && len(r.Issues) > 0
}

// ErrorCount returns the number of error-level issues
func (r *ValidationResult) ErrorCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.IsError {
			count++
		}
	}
	return count
}

// WarningCount returns the number of warning-level issues
func (r *ValidationResult) WarningCount() int {
	count := 0
	for _, issue := range r.Issues {
		if !issue.IsError {
			count++
		}
	}
	return count
}

// FormatForCLI returns a formatted string for CLI output
func (r *ValidationResult) FormatForCLI() string {
	var sb strings.Builder

	if r.AppName != "" {
		sb.WriteString(fmt.Sprintf("%s: ", r.AppName))
	}

	sb.WriteString(r.Status.String())

	if len(r.Issues) > 0 {
		sb.WriteString("\n")
		for _, issue := range r.Issues {
			prefix := "  ⚠ "
			if r.Status == StatusError {
				prefix = "  ✗ "
			} else if r.Status == StatusOK {
				prefix = "  ✓ "
			}

			sb.WriteString(fmt.Sprintf("%s%s", prefix, issue.Message))
			if issue.Field != "" {
				sb.WriteString(fmt.Sprintf(" (field: %s)", issue.Field))
			}
			if issue.Hint != "" {
				sb.WriteString(fmt.Sprintf("\n    Hint: %s", issue.Hint))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// ConfigSet represents a set of loaded configurations
type ConfigSet struct {
	Apps    map[string]*AppConfig    `json:"apps"`
	Domains map[string]*DomainConfig `json:"domains"`
}

// NewConfigSet creates a new empty ConfigSet
func NewConfigSet() *ConfigSet {
	return &ConfigSet{
		Apps:    make(map[string]*AppConfig),
		Domains: make(map[string]*DomainConfig),
	}
}

// AddApp adds an app configuration
func (cs *ConfigSet) AddApp(name string, config *AppConfig) {
	cs.Apps[name] = config
}

// AddDomain adds a domain configuration
func (cs *ConfigSet) AddDomain(name string, config *DomainConfig) {
	cs.Domains[name] = config
}

// GetApp returns an app configuration by name
func (cs *ConfigSet) GetApp(name string) (*AppConfig, bool) {
	app, ok := cs.Apps[name]
	return app, ok
}

// GetDomain returns a domain configuration by name
func (cs *ConfigSet) GetDomain(name string) (*DomainConfig, bool) {
	domain, ok := cs.Domains[name]
	return domain, ok
}

// AppNames returns all app names
func (cs *ConfigSet) AppNames() []string {
	names := make([]string, 0, len(cs.Apps))
	for name := range cs.Apps {
		names = append(names, name)
	}
	return names
}

// DomainNames returns all domain names
func (cs *ConfigSet) DomainNames() []string {
	names := make([]string, 0, len(cs.Domains))
	for name := range cs.Domains {
		names = append(names, name)
	}
	return names
}
