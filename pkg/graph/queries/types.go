package queries

// QueryMetadata remains useful for grouping metadata within the Query struct.
type QueryMetadata struct {
	Name             string   `yaml:"name"`             // User-friendly name of the query
	Description      string   `yaml:"description"`      // Detailed description of what the query does
	ImpactedServices []string `yaml:"impactedServices"` // List of cloud services the query relates to
	Severity         string   `yaml:"severity"`         // e.g., Critical, High, Medium, Low, Informational
	Order            int      `yaml:"order"`            // Execution order - lower numbers run first (default 0)
}

// Query represents a single loaded query.
type Query struct {
	// Fields loaded from YAML
	QueryMetadata `yaml:",inline"` // Embeds QueryMetadata fields at the top level of YAML
	Cypher        string           `yaml:"cypher"` // The Cypher query to execute

	// Fields populated programmatically, not from YAML
	ID       string // Unique identifier, e.g., "aws/analysis/privesc/ec2_RunInstances"
	Platform string // e.g., "aws"
	Type     string // e.g., "enrich", "analysis"
	Category string // e.g., "privesc", "iam"
	FileName string // Original filename, e.g., "ec2_RunInstances.yaml"
}
