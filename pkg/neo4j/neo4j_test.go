package konstellation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/neo4j"
)

func TestNeo4j(t *testing.T) {
	ctx := context.Background()
	uri := "bolt://localhost:17687"
	user := "neo4j"
	password := "letmein!"

	neo4jContainer, err := neo4j.RunContainer(ctx,
		testcontainers.WithImage("docker.io/neo4j:4.4"),
		neo4j.WithAdminPassword(password),
		neo4j.WithNeo4jSetting("dbms.connectors.default_listen_address", "127.0.0.1"),
		neo4j.WithNeo4jSetting("dbms.connector.bolt.listen_address", "17687"),
		neo4j.WithNeo4jSetting("dbms.connector.http.listen_address", "17474"),
	)

	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}

	// Create a new instance of the neo4j package and verify connectivity
	New(
		WithContext(ctx),
		WithHost(uri),
		WithUser(user),
		WithPassword(password),
	)

	// Clean up the container
	defer func() {
		if err := neo4jContainer.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate container: %s", err)
		}
	}()

}

func TestFlattenMap(t *testing.T) {
	jsonData := []byte(`
	{
		"metadata": {
			"name": "node-proxy",
			"annotations": {
				"kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"rbac.authorization.k8s.io/v1\",\"kind\":\"ClusterRole\",\"metadata\":{\"annotations\":{},\"name\":\"node-proxy\"},\"rules\":[{\"apiGroups\":[\"\"],\"resources\":[\"nodes/proxy\"],\"verbs\":[\"get\",\"create\"]}]}\n"
			}
		},
		"rules": [
			{
				"verbs": [
					"get",
					"create"
				],
				"apiGroups": [
					""
				],
				"resources": [
					"nodes/proxy"
				]
			}
		]
	}
	`)
	//fmt.Println(jsonData)

	var data map[string]interface{}
	err := json.Unmarshal(jsonData, &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	//fmt.Println(data)

	result := FlattenMap(data)

	expectedName := "node-proxy"
	if result["metadata.name"] != expectedName {
		t.Errorf("Expected name %q, got %q", expectedName, result["metadata.name"])
	}

	expectedAnnotations := "{\"apiVersion\":\"rbac.authorization.k8s.io/v1\",\"kind\":\"ClusterRole\",\"metadata\":{\"annotations\":{},\"name\":\"node-proxy\"},\"rules\":[{\"apiGroups\":[\"\"],\"resources\":[\"nodes/proxy\"],\"verbs\":[\"get\",\"create\"]}]}\n"
	if result["metadata.annotations.kubectl.kubernetes.io/last-applied-configuration"] != expectedAnnotations {
		t.Errorf("Expected annotations %q, got %q", expectedAnnotations, result["metadata.annotations.kubectl.kubernetes.io/last-applied-configuration"])
	}

	expectedRules := "[{\"apiGroups\":[\"\"],\"resources\":[\"nodes/proxy\"],\"verbs\":[\"get\",\"create\"]}]"
	if result["rules"] != expectedRules {
		t.Errorf("Expected rules %q, got %q", expectedRules, result["rules"])
	}

	fmt.Printf("%v\n", result)
	fmt.Println(result["rules"])

}
