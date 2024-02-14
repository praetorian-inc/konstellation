package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	p "github.com/praetorian-inc/konstellation/pkg/platform"
	utils "github.com/praetorian-inc/konstellation/pkg/utils"
	"github.com/praetorian-inc/konstellation/resources"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	config            string
	defaultEnumOutput string = "k8s-enum"
	enumOutput        string
	kubeconfig        string
	platform          p.Platform
	queryName         string
	relationshipName  string
	resultsDir        string
)

func k8sCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "k8s",
		Short:   "Manage Kubernetes",
		Long:    "Manage Kubernetes clusters and resources.",
		Example: "konstellation k8s [command]",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			validSubcommands := map[string]struct{}{
				"enum":  {},
				"push":  {},
				"query": {},
			}

			subcommand := cmd.CalledAs()

			if _, ok := validSubcommands[subcommand]; !ok {
				fmt.Println("Invalid subcommand. Please use one of: enum, push, query.")
				cmd.Help()
				os.Exit(1)
			}
		},
	}

	//cmd.PersistentFlags().StringVar(&enumOutput, "config", defaultEnumOutput, "Enum output.")
	cmd.AddCommand(k8sEnumCmd())
	cmd.AddCommand(k8sPushCmd())
	cmd.AddCommand(k8sQueryCmd())

	return cmd
}

func k8sEnumCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enum",
		Short: "Enumerate Kubernetes resources",
		Long:  "Enumerate Kubernetes resources and display them.",
		Run:   doEnum,
	}

	cmd.PersistentFlags().StringVar(&enumOutput, "enum", defaultEnumOutput, "Enum output.")
	cmd.MarkFlagDirname("enum")
	cmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig")
	cmd.MarkFlagFilename("kubeconfig")
	cmd.PersistentFlags().Bool("incluster", false, "Incluster kubernetes")

	return cmd
}

func k8sPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push Kubernetes resources",
		Long:  "Push Kubernetes resources to a cluster.",
		Run:   doPush,
	}

	// Add flags for push command here
	AddNeo4jFlags(cmd)
	cmd.PersistentFlags().StringVar(&enumOutput, "enum", "", "Enum output.")
	cmd.MarkFlagDirname("enum")
	cmd.PersistentFlags().Bool("relationships", false, "Only run relationship mappings")
	cmd.PersistentFlags().StringVar(&relationshipName, "relationship-name", "", "Run the specififed relationship mapping")

	return cmd
}

func k8sQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query Kubernetes resources",
		Long:  "Query Kubernetes resources and retrieve information.",
		Run:   doQuery,
	}

	AddNeo4jFlags(cmd)
	cmd.PersistentFlags().StringVar(&queryName, "name", "", "Name of query to run")
	cmd.PersistentFlags().Bool("print", false, "Print query results to stdout")
	cmd.PersistentFlags().StringVar(&resultsDir, "results", "results", "Query results output directory")
	cmd.MarkFlagDirname("results")

	return cmd
}

func doEnum(cmd *cobra.Command, args []string) {
	fmt.Println("doEnum")
	platform := p.NewPlatform("k8s", resources.K8sConfigPath)
	platform.SetCmd(cmd)
	ctx := context.Background()
	//platform.SetDriver(utils.Neo4jSetup(ctx, cmd))
	platform.Enum(ctx)
}

func doPush(cmd *cobra.Command, args []string) {
	//validateDefaultFlags(cmd)
	logrus.Warn("push.doPush()")

	platform := p.NewPlatform("k8s", resources.K8sConfigPath)
	platform.Name = "k8s"
	platform.SetCmd(cmd)

	defaultEnumOutput = platform.Name + "-enum"

	// validate enum directory
	pushEnumOutput, err := utils.GetDirectoryPath(cmd, "enum", defaultEnumOutput)
	if err != nil {

		logrus.Errorf("enum directory '%v' doesn't exist", pushEnumOutput)
		os.Exit(1)
	}
	logrus.Infof("Using enum directory %v", pushEnumOutput)

	ctx := context.Background()
	platform.SetDriver(utils.Neo4jSetup(ctx, cmd))
	platform.Push(ctx)
}

func doQuery(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	fmt.Println("doQuery")
	platform := p.NewPlatform("k8s", resources.K8sConfigPath)
	platform.Name = "k8s"
	platform.SetCmd(cmd)
	platform.SetDriver(utils.Neo4jSetup(ctx, cmd))
	defaultEnumOutput = platform.Name + "-enum"
	platform.Query(ctx)

}
