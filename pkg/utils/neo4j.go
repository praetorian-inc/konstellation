package konstellation

import (
	"context"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func Neo4jSetup(ctx context.Context, cmd *cobra.Command) neo4j.DriverWithContext {

	uri, _ := cmd.Flags().GetString("neo4j-uri")
	user, _ := cmd.Flags().GetString("neo4j-user")
	password, _ := cmd.Flags().GetString("neo4j-pass")

	//fmt.Printf("uri: %s, user: %s, password: %s\n", uri, user, password)
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))

	if err != nil {
		logrus.Errorf("Exception: %v", err)
		os.Exit(1)
	}

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		logrus.Errorf("Exception: %v", err)
		os.Exit(1)
	}

	return driver
}
