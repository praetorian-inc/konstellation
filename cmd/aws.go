package cmd

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	neo4j "github.com/praetorian-inc/konstellation/pkg/neo4j"
	p "github.com/praetorian-inc/konstellation/pkg/platform"
	utils "github.com/praetorian-inc/konstellation/pkg/utils"
	"github.com/praetorian-inc/konstellation/resources"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var awsCmd = &cobra.Command{
	Use:     "aws",
	Short:   "aws",
	Example: "konstellation aws [command]",
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

var awsPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push AWS resources",
	Long:  "Push AWS resources to a cluster.",
	Run:   awsPush,
}

func init() {
	AddNeo4jFlags(awsPushCmd)
	AddPushFlags(awsPushCmd)

	AddNeo4jFlags(awsQueryCmd)
	AddQueryFlags(awsQueryCmd)
	awsCmd.AddCommand(awsPushCmd)
	awsCmd.AddCommand(awsQueryCmd)
}

var awsQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query AWS resources",
	Long:  "Query AWS resources and retrieve information.",
	Run:   awsQuery,
}

func awsPush(cmd *cobra.Command, args []string) {
	//validateDefaultFlags(cmd)
	logrus.Warn("push.doPush()")

	ctx := context.Background()
	n := neo4j.Neo4jFromCLI(ctx, cmd)
	platform := p.NewAWS(resources.AwsConfigPath, cmd, n)

	defaultEnumOutput = platform.Name + "-enum"

	// validate enum directory
	pushEnumOutput, err := utils.GetDirectoryPath(cmd, "enum", defaultEnumOutput)
	if err != nil {

		logrus.Errorf("enum directory '%v' doesn't exist", pushEnumOutput)
		os.Exit(1)
	}
	logrus.Infof("Using enum directory %v", pushEnumOutput)

	platform.Push(ctx)
}

func awsQuery(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	fmt.Println("doQuery")
	n := neo4j.Neo4jFromCLI(ctx, cmd)
	platform := p.NewK8s(resources.AwsConfigPath, cmd, n)
	defaultEnumOutput = platform.Name + "-enum"
	platform.Query(ctx)

}
