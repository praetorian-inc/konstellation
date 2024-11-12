package konstellation

import (
	"context"
	"encoding/json"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type GraphDB interface {
	Query(query string, resultsChannel chan any) error
	Insert(node *Node, resultsChannel chan any) error
	Update(node *Node, resultsChannel chan any) error
	Delete(node *Node) error
}

type Node struct {
	neo4j.Node
}

type Neo4j struct {
	ctx      context.Context
	uri      string
	port     int
	user     string
	password string
	driver   neo4j.DriverWithContext
	session  neo4j.SessionWithContext
}

func (n *Neo4j) connect() {
	var err error
	n.driver, err = neo4j.NewDriverWithContext(n.uri, neo4j.BasicAuth(n.user, n.password, ""))

	if err != nil {
		logrus.Errorf("Exception: %v", err)
		os.Exit(1)
	}

	err = n.driver.VerifyConnectivity(n.ctx)
	if err != nil {
		logrus.Errorf("Exception: %v", err)
		os.Exit(1)
	}

	n.session = n.driver.NewSession(n.ctx, neo4j.SessionConfig{})
	defer n.session.Close(n.ctx)
}

func New(options ...func(*Neo4j)) *Neo4j {
	svr := &Neo4j{}
	for _, o := range options {
		o(svr)
	}

	// verify connectivity
	svr.connect()

	return svr
}

func (n *Neo4j) Query(query string, resultsChan chan any) error {

	_, err := n.session.ExecuteRead(n.ctx, func(transaction neo4j.ManagedTransaction) (any, error) {
		result, err := transaction.Run(n.ctx,
			query,
			nil)
		if err != nil {
			return nil, err
		}

		if result.Next(n.ctx) {
			resultsChan <- result
			return resultsChan, nil
		}

		return nil, result.Err()
	})

	return err
}

func (n *Neo4j) Insert(node *Node, resultsChan chan any) error {
	val, err := n.session.ExecuteWrite(n.ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		val, err := tx.Run(n.ctx, "CREATE (n:"+node.Labels[0]+") SET n = $nodeProps", map[string]interface{}{
			"labels":    node.Labels,
			"nodeProps": node.Props,
		})
		if err != nil {
			return nil, err
		}
		return val, nil
	})

	resultsChan <- val

	return err
}

func (n *Neo4j) Update(node *Node, resultsChan chan any) error {

	return nil
}

func (n *Neo4j) Delete(node *Node) error {

	return nil
}

func WithContext(ctx context.Context) func(*Neo4j) {
	return func(s *Neo4j) {
		s.ctx = ctx
	}
}

func WithHost(uri string) func(*Neo4j) {
	return func(s *Neo4j) {
		s.uri = uri
	}
}

func WithPort(port int) func(*Neo4j) {
	return func(s *Neo4j) {
		s.port = port
	}
}

func WithUser(user string) func(*Neo4j) {
	return func(s *Neo4j) {
		s.user = user
	}
}

func WithPassword(password string) func(*Neo4j) {
	return func(s *Neo4j) {
		s.password = password
	}
}

func Item2Node(item map[string]interface{}, label string) Node {
	flat := FlattenMap(item)
	node := neo4j.Node{Labels: []string{label}, Props: flat}
	n := Node{node}

	logrus.Trace(node)
	return n
}

func FlattenMap(m map[string]interface{}) map[string]interface{} {
	flattened := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			flattenedMap := FlattenMap(val)
			for fk, fv := range flattenedMap {
				flattened[k+"."+fk] = fv
			}
		case string:
			flattened[k] = val
		default:
			jsonString, _ := json.Marshal(val)
			flattened[k] = string(jsonString)
		}
	}
	logrus.Tracef("Flattened map: %v", flattened)
	return flattened
}

func Neo4jFromCLI(ctx context.Context, cmd *cobra.Command) *Neo4j {

	uri, _ := cmd.Flags().GetString("neo4j-uri")
	user, _ := cmd.Flags().GetString("neo4j-user")
	password, _ := cmd.Flags().GetString("neo4j-pass")

	//fmt.Printf("uri: %s, user: %s, password: %s\n", uri, user, password)
	n := NewNeo4j(ctx, uri, user, password)

	return n
}

func NewNeo4j(ctx context.Context, uri, user, password string) *Neo4j {

	n := New(
		WithContext(ctx),
		WithHost(uri),
		WithUser(user),
		WithPassword(password),
	)

	return n
}
