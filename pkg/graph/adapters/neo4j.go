package adapters

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	neo4jConfig "github.com/neo4j/neo4j-go-driver/v5/neo4j/config"
	"github.com/praetorian-inc/konstellation/pkg/graph"
)

const (
	// DefaultBatchSize is the default number of nodes/relationships to process in a single transaction
	DefaultBatchSize = 1000
)

type Neo4jDatabase struct {
	driver    neo4j.DriverWithContext
	batchSize int
}

func NewNeo4jDatabase(config *graph.Config) (*Neo4jDatabase, error) {
	driver, err := neo4j.NewDriverWithContext(config.URI,
		neo4j.BasicAuth(config.Username, config.Password, ""),
		func(c *neo4jConfig.Config) {
			if v, ok := config.Options["maxConnectionPoolSize"]; ok {
				if maxPoolSize, err := strconv.Atoi(v); err == nil {
					c.MaxConnectionPoolSize = maxPoolSize
				}
			}
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	batchSize := DefaultBatchSize
	if size, ok := config.Options["batchSize"]; ok {
		if batchSizeInt, err := strconv.Atoi(size); err == nil {
			batchSize = batchSizeInt
		}
	}

	db := &Neo4jDatabase{driver: driver, batchSize: batchSize}
	db.initializeConstraints(context.Background())

	return db, nil
}

// pkg/graph/adapters/neo4j/neo4j.go

func (db *Neo4jDatabase) CreateNodes(ctx context.Context, nodes []*graph.Node) (*graph.BatchResult, error) {
	if len(nodes) == 0 {
		return &graph.BatchResult{}, nil
	}

	// Validate nodes
	for _, node := range nodes {
		if len(node.UniqueKey) == 0 {
			return nil, fmt.Errorf("node must have at least one unique key property")
		}
		if len(node.Labels) == 0 {
			return nil, fmt.Errorf("node must have at least one label")
		}
	}

	// Group nodes by their labels and unique keys
	groups := make(map[string][]*graph.Node)
	for _, node := range nodes {
		key := getNodeGroupKey(node.Labels, node.UniqueKey)
		groups[key] = append(groups[key], node)
	}

	result := &graph.BatchResult{}

	// Process each group in batches
	for groupKey, groupedNodes := range groups {
		// Parse the group key
		parts := strings.Split(groupKey, "||")
		labels := strings.Split(parts[0], ":")
		uniqueKeys := strings.Split(parts[1], ":")

		// Process in batches
		for i := 0; i < len(groupedNodes); i += db.batchSize {
			end := i + db.batchSize
			if end > len(groupedNodes) {
				end = len(groupedNodes)
			}
			batch := groupedNodes[i:end]

			session := db.driver.NewSession(ctx, neo4j.SessionConfig{})

			batchResult, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
				return db.processBatch(ctx, tx, batch, labels, uniqueKeys)
			})

			session.Close(ctx)

			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("batch processing error: %w", err))
				continue
			}

			if br, ok := batchResult.(*graph.BatchResult); ok {
				result.NodesCreated += br.NodesCreated
				result.NodesUpdated += br.NodesUpdated
			}
		}
	}

	return result, nil
}

func (db *Neo4jDatabase) CreateRelationships(ctx context.Context, rels []*graph.Relationship) (*graph.BatchResult, error) {
	if len(rels) == 0 {
		return &graph.BatchResult{}, nil
	}

	result := &graph.BatchResult{}

	// Group relationships
	groups := make(map[string][]*graph.Relationship)
	for _, rel := range rels {
		// log.Default().Println("rel.SartNode", rel.StartNode)
		// log.Default().Println("rel.EndNode", rel.EndNode)
		// log.Default().Println("rel.Type", rel.Type)
		// log.Default().Println("rel.Properties", rel.Properties)
		// log.Default().Println("rel.StartNode.UniqueKey", rel.StartNode.UniqueKey)
		// log.Default().Println("rel.EndNode.UniqueKey", rel.EndNode.UniqueKey)
		if len(rel.StartNode.UniqueKey) == 0 || len(rel.EndNode.UniqueKey) == 0 {
			return nil, fmt.Errorf("both start and end nodes must have unique keys")
		}
		key := fmt.Sprintf("%s||%s||%s||%s||%s",
			rel.Type,
			strings.Join(rel.StartNode.Labels, ":"),
			strings.Join(rel.StartNode.UniqueKey, ":"),
			strings.Join(rel.EndNode.Labels, ":"),
			strings.Join(rel.EndNode.UniqueKey, ":"))
		groups[key] = append(groups[key], rel)
	}

	// Process each group in batches
	for _, groupedRels := range groups {
		for i := 0; i < len(groupedRels); i += db.batchSize {
			end := i + db.batchSize
			if end > len(groupedRels) {
				end = len(groupedRels)
			}
			batch := groupedRels[i:end]

			session := db.driver.NewSession(ctx, neo4j.SessionConfig{})

			batchResult, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
				if len(batch) == 0 {
					return &graph.BatchResult{}, nil
				}

				// Use first relationship in batch as template for query
				exemplar := batch[0]
				query := buildRelationshipMergeQuery(
					exemplar.Type,
					exemplar.StartNode,
					exemplar.EndNode)
				slog.Debug("query", query)

				// Build parameters for all relationships in batch
				params := make([]map[string]interface{}, len(batch))
				for i, rel := range batch {
					params[i] = map[string]interface{}{
						"startProperties": rel.StartNode.Properties,
						"endProperties":   rel.EndNode.Properties,
						"properties":      rel.Properties,
					}
				}
				//slog.Debug("params", params)

				res, err := tx.Run(ctx, query, map[string]interface{}{
					"rels": params,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to merge relationships: %w", err)
				}

				summary, err := res.Consume(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to get query stats: %w", err)
				}

				return &graph.BatchResult{
					NodesCreated:         summary.Counters().NodesCreated(),
					NodesUpdated:         summary.Counters().PropertiesSet(),
					RelationshipsCreated: summary.Counters().RelationshipsCreated(),
					// RelationshipsUpdated: summary.Counters().RelationshipsUpdated(),
				}, nil
			})

			session.Close(ctx)

			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("batch processing error: %w", err))
				continue
			}

			if br, ok := batchResult.(*graph.BatchResult); ok {
				result.NodesCreated += br.NodesCreated
				result.NodesUpdated += br.NodesUpdated
				result.RelationshipsCreated += br.RelationshipsCreated
				result.RelationshipsUpdated += br.RelationshipsUpdated
			}
		}
	}

	return result, nil
}

func (db *Neo4jDatabase) Query(ctx context.Context, query string, params map[string]interface{}) (*graph.QueryResult, error) {
	session := db.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	records := make([]map[string]interface{}, 0)
	for result.Next(ctx) {
		record := result.Record()
		recordMap := make(map[string]interface{})
		for i, key := range record.Keys {
			recordMap[key] = record.Values[i]
		}
		records = append(records, recordMap)
	}

	if err = result.Err(); err != nil {
		return nil, fmt.Errorf("error during query iteration: %w", err)
	}

	return &graph.QueryResult{
		Records: records,
	}, nil
}

func (db *Neo4jDatabase) Close() error {
	if db.driver != nil {
		return db.driver.Close(context.Background())
	}
	return nil
}

// propertyMapToString converts a map of properties to a Neo4j property string format
// e.g., {arn: "...", name: "..."} becomes {arn: $arn, name: $name}
// func propertyMapToString(props map[string]interface{}) string {
// 	if len(props) == 0 {
// 		return "{}"
// 	}

// 	parts := make([]string, 0, len(props))
// 	for key := range props {
// 		parts = append(parts, fmt.Sprintf("%s: $props.%s", key, key))
// 	}
// 	return "{" + strings.Join(parts, ", ") + "}"
// }

// Instead of using a struct as map key
// type nodeGroupKey struct {
// 	labels    []string
// 	uniqueKey []string
// }

// Convert to using a string key that encodes the same information
func getNodeGroupKey(labels []string, uniqueKey []string) string {
	return fmt.Sprintf("%s||%s",
		strings.Join(labels, ":"),
		strings.Join(uniqueKey, ":"))
}

// func groupNodes(nodes []*graph.Node) map[string][]*graph.Node {
// 	groups := make(map[string][]*graph.Node)

// 	for _, node := range nodes {
// 		key := getNodeGroupKey(node.Labels, node.UniqueKey)
// 		groups[key] = append(groups[key], node)
// 	}

// 	return groups
// }

func buildBatchMergeQuery(labels []string, uniqueKey []string) string {
	labelStr := strings.Join(labels, ":")

	// Check if this is a Service Principal
	//isServicePrincipal := contains(labels, "Service") && contains(labels, "Principal")

	// Build the merge criteria based on the unique key
	mergeParts := make([]string, len(uniqueKey))
	for i, key := range uniqueKey {
		mergeParts[i] = fmt.Sprintf("%s: node.%s", key, key)
	}
	mergeProps := strings.Join(mergeParts, ", ")

	// Construct the query with null-safety
	return fmt.Sprintf(`
        UNWIND $nodes as node
        MERGE (n:%s {%s})
        ON CREATE SET n = node.properties, n._created = timestamp()
        ON MATCH SET n = node.properties, n._updated = timestamp()
    `, labelStr, mergeProps)
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func nodeListToParams(nodes []*graph.Node) []map[string]interface{} {
	params := make([]map[string]interface{}, len(nodes))

	for i, node := range nodes {
		nodeMap := map[string]interface{}{
			"properties": node.Properties,
		}
		// Add unique keys at top level for MERGE
		for _, key := range node.UniqueKey {
			if val, ok := node.Properties[key]; ok {
				nodeMap[key] = val
			}
		}
		params[i] = nodeMap
	}

	return params
}

// processBatch handles a single batch of nodes with the same structure
func (db *Neo4jDatabase) processBatch(
	ctx context.Context,
	tx neo4j.ManagedTransaction,
	nodes []*graph.Node,
	labels []string,
	uniqueKey []string,
) (*graph.BatchResult, error) {
	result := &graph.BatchResult{}

	// Build query for this batch
	query := buildBatchMergeQuery(labels, uniqueKey)
	params := map[string]interface{}{
		"nodes": nodeListToParams(nodes),
	}

	// Execute merge
	res, err := tx.Run(ctx, query, params)
	if err != nil {
		return result, fmt.Errorf("failed to merge nodes: %w", err)
	}

	summary, err := res.Consume(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to get query stats: %w", err)
	}

	result.NodesCreated = summary.Counters().NodesCreated()
	result.NodesUpdated = summary.Counters().PropertiesSet()

	return result, nil
}

// Add this to your Neo4jDatabase struct
func (db *Neo4jDatabase) initializeConstraints(ctx context.Context) error {
	session := db.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Create constraint for ARN uniqueness across all nodes that have an ARN
	_, err := session.Run(ctx,
		"CREATE CONSTRAINT unique_arn IF NOT EXISTS FOR (n) REQUIRE n.arn IS UNIQUE",
		nil)
	if err != nil {
		return fmt.Errorf("failed to create ARN constraint: %w", err)
	}

	// Create specific constraint for Service Principals
	_, err = session.Run(ctx,
		"CREATE CONSTRAINT unique_service_principal IF NOT EXISTS FOR (n:Service:Principal) REQUIRE n.name IS UNIQUE",
		nil)
	if err != nil {
		return fmt.Errorf("failed to create Service Principal constraint: %w", err)
	}

	return nil
}

func buildRelationshipMergeQuery(relType string, startNode, endNode *graph.Node) string {
	// Wrap each label in backticks and join them with ":"
	startLabels := make([]string, len(startNode.Labels))
	for i, label := range startNode.Labels {
		startLabels[i] = "`" + label + "`"
	}

	endLabels := make([]string, len(endNode.Labels))
	for i, label := range endNode.Labels {
		endLabels[i] = "`" + label + "`"
	}

	// Determine the unique property for start node
	startUnique := buildPropsString(startNode.UniqueKey, "rel.startProperties")

	// Determine the unique property for end node
	endUnique := buildPropsString(endNode.UniqueKey, "rel.endProperties")

	return fmt.Sprintf(`
        UNWIND $rels as rel
        MERGE (start:%s {%s})
        ON CREATE SET start = rel.startProperties
        ON MATCH SET start += rel.startProperties
        MERGE (end:%s {%s})
        ON CREATE SET end = rel.endProperties
        ON MATCH SET end += rel.endProperties
        MERGE (start)-[r:`+"`%s`"+`]->(end)
        ON CREATE SET r = rel.properties, r._created = timestamp()
        ON MATCH SET r = rel.properties, r._updated = timestamp()
        RETURN count(r) as total
    `,
		strings.Join(startLabels, ":"),
		startUnique,
		strings.Join(endLabels, ":"),
		endUnique,
		relType)
}

func buildPropsString(uniqueKey []string, prefix string) string {
	parts := make([]string, len(uniqueKey))
	for i, key := range uniqueKey {
		parts[i] = fmt.Sprintf("%s: %s.%s", key, prefix, key)
	}
	return strings.Join(parts, ", ")
}
