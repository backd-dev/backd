package api

import (
	"github.com/backd-dev/backd/internal/auth"
	"github.com/backd-dev/backd/internal/config"
	"github.com/backd-dev/backd/internal/db"
	"github.com/backd-dev/backd/internal/functions"
	"github.com/backd-dev/backd/internal/metrics"
	"github.com/backd-dev/backd/internal/secrets"
	"github.com/backd-dev/backd/internal/storage"
)

// Deps holds all dependencies required by API handlers
// This is constructed once per app in cmd/api/commands/start.go
type Deps struct {
	DB              db.DB
	Auth            auth.Auth
	Secrets         secrets.Secrets
	Storage         storage.Storage      // nil if no storage for this app
	Metrics         metrics.Metrics
	Config          *config.ConfigSet
	FunctionsClient functions.Client
}
