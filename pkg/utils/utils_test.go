package konstellation

import (
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
