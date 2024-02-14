package cmd

import (
	"github.com/spf13/cobra"
)

var (
	neo4jURI  string
	neo4jUser string
	neo4jPass string
)

func AddNeo4jFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&neo4jURI, "neo4j-uri", "bolt://localhost:7687", "Neo4j URI")
	cmd.PersistentFlags().StringVar(&neo4jUser, "neo4j-user", "neo4j", "Neo4j user")
	cmd.PersistentFlags().StringVar(&neo4jPass, "neo4j-pass", "neo4j", "Neo4j password")
}
