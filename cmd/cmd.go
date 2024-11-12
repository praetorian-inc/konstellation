package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "konstellation",
	Short: "Konstellation",
	Long:  "Konstellation Konstellation is a configuration-driven CLI tool to enumerate cloud resources and store the data into neo4j.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		//if len(args) != 1 || args[0] != "k8s" {
		if len(args) != 1 || args[0] != "k8s" {
			cmd.Help()
			os.Exit(1)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(k8sCmd())
	rootCmd.AddCommand(awsCmd)

	banner := "Konstellation"
	rootCmd.SetVersionTemplate(fmt.Sprintf("%s\n\n", banner))
}

func AddPushFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&enumOutput, "enum", "", "Enum output.")
	cmd.MarkFlagDirname("enum")
	cmd.PersistentFlags().Bool("relationships", false, "Only run relationship mappings")
	cmd.PersistentFlags().StringVar(&relationshipName, "relationship-name", "", "Run the specififed relationship mapping")
}

func AddQueryFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&queryName, "name", "", "Name of query to run")
	cmd.PersistentFlags().Bool("print", false, "Print query results to stdout")
	cmd.PersistentFlags().StringVar(&resultsDir, "results", "results", "Query results output directory")
	cmd.MarkFlagDirname("results")
}
