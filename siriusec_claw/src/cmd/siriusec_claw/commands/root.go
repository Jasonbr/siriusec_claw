package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/siriusec/siriusec_claw/pkg/version"
)

var rootCmd = &cobra.Command{
	Use:     "siriusec_claw",
	Short:   "SiriuSec Claw - AI-powered operations platform",
	Long:    "SiriuSec Claw is an enterprise-grade AI agent platform for intelligent operations.",
	Version: version.Version,
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("SiriuSec Claw %s\n", version.Version))
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
