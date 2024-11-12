package konstellation

import (
	"context"
	"embed"

	graphdb "github.com/praetorian-inc/konstellation/pkg/neo4j"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type AWS struct {
	BasePlatform
}

func NewAWS(configPath embed.FS, cmd *cobra.Command, graphdb graphdb.GraphDB) *AWS {

	aws := &AWS{
		BasePlatform: BasePlatform{
			Name:    "aws",
			graphdb: graphdb,
		},
	}

	aws.BasePlatform.cmd = cmd
	aws.loadConfig("aws/config.yml", configPath)

	return aws
}

func (a *AWS) insertNode(ctx context.Context, node *graphdb.Node) (chan any, error) {
	anyChannel := make(chan any, 1)

	err := a.graphdb.Insert(node, anyChannel)

	if err != nil {
		logrus.Errorf("Failed to insert node: %v", err)
		return nil, err
	}

	return anyChannel, nil
}
