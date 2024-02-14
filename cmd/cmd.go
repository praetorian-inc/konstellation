package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func Execute() error {
	rootCmd := &cobra.Command{
		Use:   "konstellation",
		Short: "Konstellation",
		Long:  "Konstellation Konstellation is a configuration-driven CLI tool to enumerate cloud resources and store the data into neo4j.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 || args[0] != "k8s" {
				cmd.Help()
				os.Exit(1)
				return
			}
		},
	}

	banner := "Konstellation"
	rootCmd.SetVersionTemplate(fmt.Sprintf("%s\n\n", banner))

	// Add subcommands and flags here
	rootCmd.AddCommand(k8sCmd())

	return rootCmd.Execute()
}
