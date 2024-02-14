package konstellation

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestFilterJson(t *testing.T) {
	data := []byte(`{"name": "John", "age": 30, "city": "New York"}`)
	jsonPath := ".name"

	filtered, err := FilterJson(data, jsonPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := []byte(`"John"`)
	if string(filtered) != string(expected) {
		t.Errorf("Expected filtered data %q, got %q", expected, filtered)
	}

	data = []byte(`{"a": "foo", "b": {"c": "bar"}}`)
	jsonPath = ".b.c"

	filtered, err = FilterJson(data, jsonPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected = []byte(`"bar"`)
	if string(filtered) != string(expected) {
		t.Errorf("Expected filtered data %q, got %q", expected, filtered)
	}

	data = []byte(`{"a": "foo", "b": {"c": "bar"}}`)
	jsonPath = ".b.c"

	filtered, err = FilterJson(data, jsonPath, true)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected = []byte(`bar`)
	if string(filtered) != string(expected) {
		t.Errorf("Expected filtered data %q, got %q", expected, filtered)
	}
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
