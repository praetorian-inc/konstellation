package konstellation

import (
	_ "embed"
	"testing"

	"github.com/praetorian-inc/konstellation/resources"
)

func TestGetFileContents(t *testing.T) {
	// Call the function under test
	contents, err := GetFileContents("k8s/config.yml", resources.K8sConfigPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check if the contents match
	if string(contents) != string(resources.K8sConfigFile) {
		t.Errorf("Expected contents %q, got %q", resources.K8sConfigFile, contents)
	}
}

/*
func TestListFiles(t *testing.T) {
	err := os.Mkdir("test", 0755)
	if err != nil {
		fmt.Println("Error creating folder:", err)
		return
	}

	// Create and write to test1.txt
	err = os.WriteFile("test/test1.txt", []byte("test"), 0644)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}

	// Create and write to test2.txt
	err = os.WriteFile("test/test2.txt", []byte("test"), 0644)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}

	//go:embed test/*
	var testFS embed.FS

	// Call the function under test
	files, err := ListFiles(testFS)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check if the returned files match the expected files
	expectedFiles := []string{
		"config.go",
		"platform_test.go",
		"platform.go",
	}

	if len(files) != len(expectedFiles) {
		t.Fatalf("Expected %d files, got %d", len(expectedFiles), len(files))
	}

	for i, file := range expectedFiles {
		if files[i] != file {
			t.Errorf("Expected file %q, got %q", file, files[i])
		}
	}
}*/

func TestGetMappingValue(t *testing.T) {
	p := &Platform{
		Config: PlatformConfig{
			Mappings: map[string]Mapping{
				"default": {
					Template: "default.cypher",
				},
				"pods.json": {
					Template: "pods.cypher",
				},
			},
		},
	}

	field := "Template"
	fileName := "foo.json"
	expectedValue := "default.cypher"
	value, err := p.getMappingValue(fileName, field)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if value != expectedValue {
		t.Errorf("Expected value %q, got %q", expectedValue, value)
	}

	// Test case 2: Mapping does not exist for the given file name, fallback to default mapping
	fileName = "pods.json"
	expectedValue = "pods.cypher"
	value, err = p.getMappingValue(fileName, field)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if value != expectedValue {
		t.Errorf("Expected value %q, got %q", expectedValue, value)
	}

	// Test case 3: Field does not exist in the mapping
	expectedError := "Field Field3 does not exist"
	_, err = p.getMappingValue(fileName, "Field3")
	if err == nil {
		t.Error("Expected an error, but got nil")
	} else if err.Error() != expectedError {
		t.Errorf("Expected error %q, got %q", expectedError, err.Error())
	}
}
