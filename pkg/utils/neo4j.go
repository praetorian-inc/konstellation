package konstellation

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	k "github.com/praetorian-inc/konstellation/pkg/neo4j"
	"github.com/spf13/cobra"
)

func Neo4jSetup(ctx context.Context, cmd *cobra.Command) neo4j.DriverWithContext {

	uri, _ := cmd.Flags().GetString("neo4j-uri")
	user, _ := cmd.Flags().GetString("neo4j-user")
	password, _ := cmd.Flags().GetString("neo4j-pass")

	//fmt.Printf("uri: %s, user: %s, password: %s\n", uri, user, password)

	driver := k.New(
		k.WithContext(ctx),
		k.WithHost(uri),
		k.WithUser(user),
		k.WithPassword(password),
	)

	return driver.Driver
}
