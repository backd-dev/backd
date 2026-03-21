package main

import (
	"os"

	"github.com/backd-dev/backd/cmd/backd/commands"
	"github.com/fernandezvara/commandkit"
)

func main() {
	cfg := commandkit.New()

	cfg.Command("bootstrap").
		Func(commands.BootstrapFunc).
		ShortHelp("Scaffold a new backd application or domain").
		LongHelp("Create a new backd application directory structure with configuration files, or create a domain configuration.").
		Config(func(cc *commandkit.CommandConfig) {
			cc.Define("name").
				String().
				Required().
				Flag("name").
				Description("Name of the app/domain to bootstrap")
			cc.Define("domain").
				Bool().
				Default(false).
				Flag("domain").
				Description("Create a domain instead of an app")
			cc.Define("force").
				Bool().
				Default(false).
				Flag("force").
				Description("Overwrite existing files")
		})

	cfg.Command("validate").
		Func(commands.ValidateFunc).
		ShortHelp("Validate configuration files").
		LongHelp("Static analysis of configuration files without runtime requirements. Validates all apps and domains in the config root.").
		Config(func(cc *commandkit.CommandConfig) {
			cc.Define("config-dir").
				String().
				Default(".").
				Flag("config-dir").
				Description("Configuration root directory")
			cc.Define("json").
				Bool().
				Default(false).
				Flag("json").
				Description("Output results as JSON")
			cc.Define("check-env").
				Bool().
				Default(false).
				Flag("check-env").
				Description("Check if environment variables exist")
		})

	if err := cfg.Execute(os.Args); err != nil {
		os.Exit(1)
	}
}
