package main

import (
	"os"

	"github.com/siriusec/siriusec_claw/cmd/siriusec_claw/commands"
	"github.com/siriusec/siriusec_claw/pkg/config"
)

func main() {
	// Best-effort: load .env from cwd
	_ = config.LoadEnvFromCurrentDir()

	// Ensure default config exists
	_ = config.EnsureDefaultConfig(config.DefaultEnv)

	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
