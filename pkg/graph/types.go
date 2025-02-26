package graph

import (
	"context"
)

// Node represents a graph node with built-in identity management
type Node struct {
	// Labels for the node
	Labels []string

	// Properties of the node
	Properties map[string]interface{}

	// UniqueKey specifies which properties form the unique identity of this node
	// Multiple property names mean a composite key
	UniqueKey []string
}

// GetIdentity returns the identity map for this node based on its UniqueKey
func (n *Node) GetIdentity() map[string]interface{} {
	if len(n.UniqueKey) == 0 {
		return nil
	}

	identity := make(map[string]interface{})
	for _, key := range n.UniqueKey {
		if val, exists := n.Properties[key]; exists {
			identity[key] = val
		}
	}
	return identity
}

// Equals checks if two nodes have the same identity
func (n *Node) Equals(other *Node) bool {
	if n == nil || other == nil {
		return false
	}

	nid := n.GetIdentity()
	oid := other.GetIdentity()

	if len(nid) != len(oid) {
		return false
	}

	for k, v := range nid {
		if ov, exists := oid[k]; !exists || v != ov {
			return false
		}
	}
	return true
}

// Relationship represents a graph relationship
type Relationship struct {
	// Type of relationship
	Type string

	// Properties of the relationship
	Properties map[string]interface{}

	// Start and end nodes - these must have UniqueKey defined
	StartNode *Node
	EndNode   *Node
}

// BatchResult contains results from a bulk operation
type BatchResult struct {
	// Number of nodes created
	NodesCreated int
	// Number of nodes updated
	NodesUpdated int
	// Number of relationships created
	RelationshipsCreated int
	// Number of relationships updated
	RelationshipsUpdated int
	// Any errors encountered during the batch operation
	Errors []error
}

func (b *BatchResult) PrintSummary() {
	println("Nodes created:", b.NodesCreated)
	println("Nodes updated:", b.NodesUpdated)
	println("Relationships created:", b.RelationshipsCreated)
	println("Relationships updated:", b.RelationshipsUpdated)

	if len(b.Errors) > 0 {
		println("Errors:")
		for _, err := range b.Errors {
			println(err.Error())
		}
	}
}

// QueryResult represents the result of a graph query
type QueryResult struct {
	Records []map[string]interface{}
	Error   error
}

// GraphDatabase defines the core interface for graph operations
type GraphDatabase interface {
	// Bulk node operations - will update existing nodes if they match on UniqueKey
	CreateNodes(ctx context.Context, nodes []*Node) (*BatchResult, error)

	// Bulk relationship operations - will create/update nodes as needed based on UniqueKey
	CreateRelationships(ctx context.Context, rels []*Relationship) (*BatchResult, error)

	// Query operations
	Query(ctx context.Context, query string, params map[string]interface{}) (*QueryResult, error)

	// Lifecycle
	Close() error
}

// Config holds database configuration
type Config struct {
	URI      string            `json:"uri"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Options  map[string]string `json:"options"`
}
