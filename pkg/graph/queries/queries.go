package queries

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"log/slog"
	"strings"

	"github.com/praetorian-inc/konstellation/pkg/graph"
)

//go:embed enrich/aws/*.cypher
var awsEnrichFS embed.FS

//go:embed analysis/aws/*.cypher
var awsQueriesFS embed.FS

func GetPlatformQueries(platform, qType string) ([]string, error) {
	switch platform {
	case "aws":
		switch qType {
		case "enrich":
			return fs.Glob(awsEnrichFS, "enrich/aws/*.cypher")
		case "analysis":
			return fs.Glob(awsQueriesFS, "analysis/aws/*.cypher")
		default:
			return []string{}, nil
		}
	default:
		return []string{}, nil
	}
}

func EnrichAWS(db graph.GraphDatabase) ([]*graph.QueryResult, error) {
	awsQueries, err := GetPlatformQueries("aws", "enrich")
	if err != nil {
		return []*graph.QueryResult{}, err
	}

	slog.Debug("Enriching AWS", slog.String("awsQueries", strings.Join(awsQueries, ", ")))

	params := make(map[string]interface{})
	results := make([]*graph.QueryResult, 0)
	for _, qpath := range awsQueries {
		cypher, err := awsEnrichFS.ReadFile(qpath)
		if err != nil {
			return []*graph.QueryResult{}, err
		}

		log.Default().Println("Running query", "query", cypher)
		qr, err := db.Query(context.Background(), string(cypher), params)
		if err != nil {
			return []*graph.QueryResult{}, err
		}

		results = append(results, qr)

	}

	return results, nil
}
