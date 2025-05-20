package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
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
	Records []Record
	Error   error
}

type Record map[string]any

// String formats a record as a string based on its content type.
// For paths, it formats them as "arn1 - relType -> arn2 - relType -> arn3 ..."
// For policy records, it returns a string version of the policy JSON
func (r Record) String() string {
	// Check if Record is empty
	if len(r) == 0 {
		return "Empty record"
	}

	// Use a switch statement to handle different record types
	switch {
	case r["path"] != nil:
		// Path record handling
		path, ok := r["path"].(dbtype.Path)
		if !ok {
			return fmt.Sprintf("Invalid path format")
		}

		nodes := path.Nodes
		rels := path.Relationships

		if len(nodes) == 0 {
			return "Path with no nodes"
		}

		if len(rels) == 0 {
			// Path with only one node
			startNode := nodes[0]
			if arn, found := startNode.Props["arn"]; found {
				return fmt.Sprintf("%v", arn)
			}
			return "Path with one node (no ARN)"
		}

		var formattedPath strings.Builder

		// Add the start node ARN
		startNode := nodes[0]
		startArn, found := startNode.Props["arn"]
		if !found {
			startArn = "unknown"
		}
		formattedPath.WriteString(fmt.Sprintf("(%v", startArn))

		// Add each relationship and target node
		for i := 0; i < len(rels); i++ {
			// Add relationship
			rel := rels[i]
			relType := rel.Type

			// Check relationship direction by comparing with the current node's ID
			var directionFormat string
			if i < len(nodes)-1 {
				if rel.StartElementId == nodes[i].ElementId && rel.EndElementId == nodes[i+1].ElementId {
					// Relationship goes from current node to next node
					directionFormat = ")-[%v]->("
				} else if rel.EndElementId == nodes[i].ElementId && rel.StartElementId == nodes[i+1].ElementId {
					// Relationship goes from next node to current node
					directionFormat = ")<-[%v]-("
				} else {
					// Fall back to default direction if IDs don't match
					directionFormat = ")-[%v]->("
				}
			} else {
				// Default direction if we can't determine
				directionFormat = ")-[%v]->("
			}

			formattedPath.WriteString(fmt.Sprintf(directionFormat, relType))

			// Add target node
			if i+1 < len(nodes) {
				targetNode := nodes[i+1]
				targetArn, found := targetNode.Props["arn"]
				if !found {
					targetArn = "unknown"
				}
				formattedPath.WriteString(fmt.Sprintf("%v)", targetArn))
			}
		}

		return formattedPath.String()

	case r["policy"] != nil:
		// Policy record handling
		return fmt.Sprintf("%v", r["policy"])

	case r["vulnerable"] != nil:
		// Vulnerability record with policy
		if r["policy"] != nil {
			return fmt.Sprintf("Vulnerable: %v\nPolicy: %v", r["vulnerable"], r["policy"])
		}
		return fmt.Sprintf("Vulnerable: %v", r["vulnerable"])

	default:
		// Default case for other record types
		return fmt.Sprintf("Record with %d entries", len(r))
	}
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

	// Verify connectivity to the database
	VerifyConnectivity(ctx context.Context) error
}

// Config holds database configuration
type Config struct {
	URI      string            `json:"uri"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Options  map[string]string `json:"options"`
}
