package queries

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/praetorian-inc/konstellation/pkg/graph"
	"gopkg.in/yaml.v3"
)

//go:embed all:enrich/aws
var awsEnrichFS embed.FS

//go:embed all:analysis/aws
var awsQueriesFS embed.FS

// LoadedQueries will store all parsed queries, keyed by their unique ID.
var LoadedQueries map[string]Query

func init() {
	LoadedQueries = make(map[string]Query)
	var loadErrors []string

	slog.Info("Loading AWS enrichment queries...")
	enrichQueries, err := loadQueriesFromFS(awsEnrichFS, "aws", "enrich", "enrich/aws")
	if err != nil {
		loadErrors = append(loadErrors, fmt.Sprintf("error loading AWS enrichment queries: %v", err))
	}
	for id, q := range enrichQueries {
		LoadedQueries[id] = q
		slog.Debug("Loaded enrichment query", "id", id, "name", q.Name)
	}

	slog.Info("Loading AWS analysis queries...")
	analysisQueries, err := loadQueriesFromFS(awsQueriesFS, "aws", "analysis", "analysis/aws")
	if err != nil {
		loadErrors = append(loadErrors, fmt.Sprintf("error loading AWS analysis queries: %v", err))
	}
	for id, q := range analysisQueries {
		LoadedQueries[id] = q
		slog.Debug("Loaded analysis query", "id", id, "name", q.Name)
	}

	if len(loadErrors) > 0 {
		slog.Error("Failed to load some queries", "errors", strings.Join(loadErrors, "; "))
		// Depending on requirements, you might want to panic here or handle it differently
	}
	slog.Info("Query loading complete", "totalQueries", len(LoadedQueries))
}

func loadQueriesFromFS(targetFS embed.FS, platform, queryType, embedBasePath string) (map[string]Query, error) {
	queries := make(map[string]Query)
	err := fs.WalkDir(targetFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".yaml") {
			return nil
		}

		// Get the relative path from the embedBasePath
		relPath := strings.TrimPrefix(path, embedBasePath+"/")
		dir, fileNameWithExt := filepath.Split(relPath)
		category := strings.Trim(filepath.ToSlash(dir), "/")
		queryName := strings.TrimSuffix(fileNameWithExt, filepath.Ext(fileNameWithExt))

		// Construct queryID without duplicating the path components
		queryID := fmt.Sprintf("%s/%s", platform, queryType)
		if category != "" {
			queryID = fmt.Sprintf("%s/%s", queryID, category)
		}
		queryID = fmt.Sprintf("%s/%s", queryID, queryName)

		fileContentBytes, err := fs.ReadFile(targetFS, path)
		if err != nil {
			slog.Error("Failed to read YAML query file", "path", path, "error", err)
			return nil // Continue with other files
		}

		var loadedQuery Query
		if unmarshalErr := yaml.Unmarshal(fileContentBytes, &loadedQuery); unmarshalErr != nil {
			slog.Warn("Failed to parse YAML query file, skipping.", "path", path, "error", unmarshalErr)
			return nil // Continue with other files
		}

		// Populate programmatically-set fields
		loadedQuery.ID = queryID
		loadedQuery.Platform = platform
		loadedQuery.Type = queryType
		loadedQuery.Category = category
		loadedQuery.FileName = fileNameWithExt // Store the full .yaml filename

		// If metadata.Name is empty from YAML, use a derived name
		if loadedQuery.Name == "" {
			nameParts := strings.Split(strings.ReplaceAll(queryName, "_", " "), " ")
			for i, part := range nameParts {
				if len(part) > 0 {
					nameParts[i] = strings.ToUpper(string(part[0])) + part[1:]
				}
			}
			loadedQuery.Name = strings.Join(nameParts, " ")
			if loadedQuery.Name == "" {
				loadedQuery.Name = "Untitled Query"
			}
		}

		if loadedQuery.Cypher == "" {
			slog.Warn("Query file has no cypher content, skipping.", "path", path, "id", queryID)
			return nil
		}

		queries[queryID] = loadedQuery
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", embedBasePath, err)
	}
	return queries, nil
}

// GetPlatformQueries now returns a slice of Query objects matching the platform and type.
// It can be extended to filter by category as well.
func GetPlatformQueries(platform, qType string, categories ...string) ([]Query, error) {
	var result []Query
	for _, query := range LoadedQueries {
		if query.Platform == platform && query.Type == qType {
			if len(categories) == 0 {
				result = append(result, query)
			} else {
				for _, cat := range categories {
					if query.Category == cat {
						result = append(result, query)
						break
					}
				}
			}
		}
	}
	if len(result) == 0 {
		// It's possible no queries match, decide if this is an error or not.
		// For now, returning an empty slice and no error.
		slog.Debug("No queries found for", "platform", platform, "type", qType, "categories", categories)
	}
	return result, nil
}

func EnrichAWS(db graph.GraphDatabase) ([]*graph.QueryResult, error) {
	awsEnrichmentQueries, err := GetPlatformQueries("aws", "enrich")
	if err != nil {
		return []*graph.QueryResult{}, err
	}

	slog.Debug("Enriching AWS", "queryCount", len(awsEnrichmentQueries))

	results := make([]*graph.QueryResult, 0)
	for _, query := range awsEnrichmentQueries {
		slog.Info("Running enrichment query", "id", query.ID, "name", query.Name)
		params := make(map[string]interface{}) // Params might come from query metadata or elsewhere in future
		qr, err := db.Query(context.Background(), query.Cypher, params)
		if err != nil {
			// Consider how to handle individual query errors: log and continue, or return immediately?
			slog.Error("Error running enrichment query", "id", query.ID, "name", query.Name, "error", err)
			return results, fmt.Errorf("error running query %s (%s): %w", query.ID, query.Name, err)
		}
		results = append(results, qr)
	}

	return results, nil
}

// RunPlatformQuery now takes a queryID (e.g., "aws/analysis/privesc/ec2_RunInstances")
func RunPlatformQuery(db graph.GraphDatabase, queryID string, params map[string]any) (*graph.QueryResult, error) {
	query, found := LoadedQueries[queryID]
	if !found {
		return nil, fmt.Errorf("query with ID '%s' not found", queryID)
	}

	slog.Info("Running platform query", "id", query.ID, "name", query.Name)

	if params == nil {
		params = make(map[string]any)
	}

	return db.Query(context.Background(), query.Cypher, params)
}
