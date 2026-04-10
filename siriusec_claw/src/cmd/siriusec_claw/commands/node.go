package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage remote nodes",
	Long:  "Install, start, and stop remote execution nodes.",
}

var nodeInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install node as a system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Node install not yet implemented.")
		return nil
	},
}

var nodeStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the node service",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Node start not yet implemented.")
		return nil
	},
}

var nodeStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the node service",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Node stop not yet implemented.")
		return nil
	},
}

func init() {
	nodeCmd.AddCommand(nodeInstallCmd, nodeStartCmd, nodeStopCmd)
	rootCmd.AddCommand(nodeCmd)
}
