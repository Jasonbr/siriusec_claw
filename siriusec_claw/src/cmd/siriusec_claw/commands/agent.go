package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run agent commands",
	Long:  "Execute AI agent operations directly from the command line.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Agent command not yet implemented. Use 'gateway run' to start the server.")
		return nil
	},
}

func init() {
	agentCmd.Flags().StringP("message", "m", "", "Message to send to the agent")
	rootCmd.AddCommand(agentCmd)
}
