package main

import (
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "api",
		Short: "Scripts for the Nexus API repo",
	}

	rootCmd.AddCommand(lintCmd())
	rootCmd.AddCommand(testCmd())
	rootCmd.AddCommand(installDepsCmd())

	if err := rootCmd.Execute(); err != nil {
		BackupLogger.Fatalf("Failed to run %s", err)
	}
}
