package main

import (
	"os"

	"github.com/backd-dev/backd/cmd/api/commands"
	"github.com/fernandezvara/commandkit"
)

func main() {
	cfg := commandkit.New()

	cfg.Command("start").
		Func(commands.StartFunc).
		ShortHelp("Start the backd API server").
		LongHelp("Start the full backd API service with REST endpoints, authentication, and metrics. Follows the 7-phase startup sequence.").
		Config(func(cc *commandkit.CommandConfig) {
			cc.Define("config-dir").
				String().
				Required().
				Flag("config-dir").
				Description("Configuration root directory (required)")
			cc.Define("mode").
				String().
				Default("both").
				Flag("mode").
				Description("Service mode: api, auth, or both")
		})

	cfg.Command("functions").
		Func(commands.FunctionsFunc).
		ShortHelp("Start the backd functions service").
		LongHelp("Start the Deno functions service with HTTP routes and/or job worker depending on mode.").
		Config(func(cc *commandkit.CommandConfig) {
			cc.Define("config-dir").
				String().
				Required().
				Flag("config-dir").
				Description("Configuration root directory (required)")
			cc.Define("mode").
				String().
				Default("both").
				Flag("mode").
				Description("Functions mode: functions, worker, or both")
		})

	cfg.Command("migrate").
		Func(commands.MigrateFunc).
		ShortHelp("Run database migrations").
		LongHelp("Run pending migrations for all apps or a specific app. Each migration runs in its own transaction.").
		Config(func(cc *commandkit.CommandConfig) {
			cc.Define("config-dir").
				String().
				Required().
				Flag("config-dir").
				Description("Configuration root directory (required)")
			cc.Define("app").
				String().
				Flag("app").
				Description("Run migrations for specific app only")
		})

	cfg.Command("secrets").
		Func(commands.SecretsFunc).
		ShortHelp("Apply encrypted secrets").
		LongHelp("Encrypt and apply secrets to the database. Two-phase process: validate then apply.").
		Config(func(cc *commandkit.CommandConfig) {
			cc.Define("config-dir").
				String().
				Required().
				Flag("config-dir").
				Description("Configuration root directory (required)")
			cc.Define("app").
				String().
				Flag("app").
				Description("Apply secrets for specific app only")
			cc.Define("dry-run").
				Bool().
				Default(false).
				Flag("dry-run").
				Description("Validate only, do not apply secrets")
		})

	if err := cfg.Execute(os.Args); err != nil {
		os.Exit(1)
	}
}
