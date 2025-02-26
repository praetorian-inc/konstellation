package adapters

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/neo4j"

	"github.com/praetorian-inc/konstellation/pkg/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestContainer(t *testing.T) (string, string, string, func()) {
	ctx := context.Background()

	cfg := graph.Config{
		URI:      "127.0.0.1",
		Username: "neo4j",
		Password: "letmein!",
	}

	container, err := neo4j.RunContainer(ctx,
		testcontainers.WithImage("neo4j:latest"),
		neo4j.WithAdminPassword(cfg.Password),
		neo4j.WithNeo4jSetting("dbms.connectors.default_listen_address", cfg.URI),
		neo4j.WithNeo4jSetting("dbms.connector.bolt.listen_address", "17687"),
		neo4j.WithNeo4jSetting("dbms.connector.http.listen_address", "17474"),
	)
	require.NoError(t, err)

	// Get connection details
	endpoint, err := container.BoltUrl(ctx)
	require.NoError(t, err)

	// Return cleanup function
	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	return endpoint, cfg.Username, cfg.Password, cleanup
}

func setupTestDB(t *testing.T) *Neo4jDatabase {
	uri, username, password, cleanup := setupTestContainer(t)
	t.Cleanup(cleanup)

	db, err := NewNeo4jDatabase(&graph.Config{
		URI:      uri,
		Username: username,
		Password: password,
		Options: map[string]string{
			"batchSize": "100",
		},
	})
	require.NoError(t, err)
	return db
}

func TestCreateSingleNode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	node := &graph.Node{
		Labels:    []string{"TestNode"},
		UniqueKey: []string{"name"},
		Properties: map[string]interface{}{
			"name":  "test1",
			"value": "something",
		},
	}

	result, err := db.CreateNodes(ctx, []*graph.Node{node})
	require.NoError(t, err)
	assert.Equal(t, 1, result.NodesCreated)
	assert.Equal(t, 0, result.NodesUpdated)

	// Verify node was created
	queryResult, err := db.Query(ctx, `
        MATCH (n:TestNode {name: $name})
        RETURN n.name, n.value
    `, map[string]interface{}{
		"name": "test1",
	})
	require.NoError(t, err)
	require.Len(t, queryResult.Records, 1)
	assert.Equal(t, "test1", queryResult.Records[0]["n.name"])
	assert.Equal(t, "something", queryResult.Records[0]["n.value"])
}

func TestCreateDuplicateNode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	node := &graph.Node{
		Labels:    []string{"TestNode"},
		UniqueKey: []string{"name"},
		Properties: map[string]interface{}{
			"name":  "test1",
			"value": "original",
		},
	}

	// Create initial node
	result, err := db.CreateNodes(ctx, []*graph.Node{node})
	require.NoError(t, err)
	assert.Equal(t, 1, result.NodesCreated)

	// Try to create same node with different value
	node.Properties["value"] = "updated"
	result, err = db.CreateNodes(ctx, []*graph.Node{node})
	require.NoError(t, err)
	assert.Equal(t, 0, result.NodesCreated)
	assert.Equal(t, 1, result.NodesUpdated)

	// Verify node was updated
	queryResult, err := db.Query(ctx, `
        MATCH (n:TestNode {name: $name})
        RETURN n.name, n.value
    `, map[string]interface{}{
		"name": "test1",
	})
	require.NoError(t, err)
	require.Len(t, queryResult.Records, 1)
	assert.Equal(t, "updated", queryResult.Records[0]["n.value"])
}

func TestBulkCreateNodes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a large batch of nodes
	nodes := make([]*graph.Node, 0, 1000)
	for i := 0; i < 1000; i++ {
		nodes = append(nodes, &graph.Node{
			Labels:    []string{"TestNode"},
			UniqueKey: []string{"id"},
			Properties: map[string]interface{}{
				"id":    i,
				"value": fmt.Sprintf("value-%d", i),
			},
		})
	}

	result, err := db.CreateNodes(ctx, nodes)
	require.NoError(t, err)
	assert.Equal(t, 1000, result.NodesCreated)

	// Verify nodes were created
	queryResult, err := db.Query(ctx, `
        MATCH (n:TestNode)
        RETURN count(n) as count
    `, nil)
	require.NoError(t, err)
	require.Len(t, queryResult.Records, 1)
	assert.Equal(t, int64(1000), queryResult.Records[0]["count"])
}

func TestCreateRelationships(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create two nodes
	startNode := &graph.Node{
		Labels:    []string{"User"},
		UniqueKey: []string{"name"},
		Properties: map[string]interface{}{
			"name": "user1",
		},
	}

	endNode := &graph.Node{
		Labels:    []string{"Resource"},
		UniqueKey: []string{"arn"},
		Properties: map[string]interface{}{
			"arn": "arn:aws:s3:::test-bucket",
		},
	}

	// Create relationship
	rel := &graph.Relationship{
		Type: "HAS_ACCESS",
		Properties: map[string]interface{}{
			"permissions": []string{"read", "write"},
			"created":     time.Now().Unix(),
		},
		StartNode: startNode,
		EndNode:   endNode,
	}

	result, err := db.CreateRelationships(ctx, []*graph.Relationship{rel})
	require.NoError(t, err)
	assert.Equal(t, 2, result.NodesCreated) // Both nodes should be created
	assert.Equal(t, 1, result.RelationshipsCreated)

	// Verify relationship was created
	queryResult, err := db.Query(ctx, `
        MATCH (u:User)-[r:HAS_ACCESS]->(s:Resource)
        RETURN u.name, s.arn, r.permissions
    `, nil)
	require.NoError(t, err)
	require.Len(t, queryResult.Records, 1)
	assert.Equal(t, "user1", queryResult.Records[0]["u.name"])
	assert.Equal(t, "arn:aws:s3:::test-bucket", queryResult.Records[0]["s.arn"])
	assert.Equal(t, []interface{}{"read", "write"}, queryResult.Records[0]["r.permissions"])
}

func TestBulkCreateRelationships(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create multiple relationships
	rels := make([]*graph.Relationship, 0, 100)
	for i := 0; i < 100; i++ {
		startNode := &graph.Node{
			Labels:    []string{"User"},
			UniqueKey: []string{"name"},
			Properties: map[string]interface{}{
				"name": fmt.Sprintf("user%d", i),
			},
		}

		endNode := &graph.Node{
			Labels:    []string{"Resource"},
			UniqueKey: []string{"arn"},
			Properties: map[string]interface{}{
				"arn": fmt.Sprintf("arn:aws:s3:::bucket-%d", i),
			},
		}

		rels = append(rels, &graph.Relationship{
			Type: "HAS_ACCESS",
			Properties: map[string]interface{}{
				"permissions": []string{"read"},
				"created":     time.Now().Unix(),
			},
			StartNode: startNode,
			EndNode:   endNode,
		})
	}

	result, err := db.CreateRelationships(ctx, rels)
	require.NoError(t, err)
	assert.Equal(t, 200, result.NodesCreated) // 100 users + 100 buckets
	assert.Equal(t, 100, result.RelationshipsCreated)

	// Verify relationships were created
	queryResult, err := db.Query(ctx, `
        MATCH (u:User)-[r:HAS_ACCESS]->(s:Resource)
        RETURN count(r) as count
    `, nil)
	require.NoError(t, err)
	require.Len(t, queryResult.Records, 1)
	assert.Equal(t, int64(100), queryResult.Records[0]["count"])
}

func TestInvalidNodes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	tests := []struct {
		name      string
		nodes     []*graph.Node
		expectErr string
	}{
		{
			name:      "nil nodes",
			nodes:     nil,
			expectErr: "nodes slice cannot be nil",
		},
		{
			name:      "nil node in slice",
			nodes:     []*graph.Node{nil},
			expectErr: "node at index 0 is nil",
		},
		{
			name: "missing unique key",
			nodes: []*graph.Node{{
				Labels:     []string{"Test"},
				Properties: map[string]interface{}{"name": "test"},
			}},
			expectErr: "must have at least one unique key property",
		},
		{
			name: "missing labels",
			nodes: []*graph.Node{{
				UniqueKey:  []string{"name"},
				Properties: map[string]interface{}{"name": "test"},
			}},
			expectErr: "must have at least one label",
		},
		{
			name: "missing unique key property",
			nodes: []*graph.Node{{
				Labels:     []string{"Test"},
				UniqueKey:  []string{"id"},
				Properties: map[string]interface{}{"name": "test"},
			}},
			expectErr: "missing unique key property id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.CreateNodes(ctx, tt.nodes)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectErr)
		})
	}
}
