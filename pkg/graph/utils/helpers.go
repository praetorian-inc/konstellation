package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// // FlattenJSON flattens a nested JSON structure
// func FlattenJSON(data map[string]interface{}) map[string]interface{} {
// 	result := make(map[string]interface{})
// 	for key, value := range data {
// 		switch value := value.(type) {
// 		case map[string]interface{}:
// 			flatMap := FlattenJSON(value)
// 			for k, v := range flatMap {
// 				result[fmt.Sprintf("%s.%s", key, k)] = fmt.Sprintf("%v", v)
// 			}
// 		default:
// 			result[key] = fmt.Sprintf("%s", value)
// 		}
// 	}
// 	return result
// }

// // ConvertAndFlatten converts an interface to JSON, flattens it, and returns a map
// func ConvertAndFlatten(input interface{}) (map[string]interface{}, error) {

// 	jsonData, err := json.Marshal(input)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var data map[string]interface{}
// 	if err := json.Unmarshal(jsonData, &data); err != nil {
// 		return nil, err
// 	}

// 	return FlattenJSON(data), nil
// }

// FlattenJSON flattens a nested JSON structure
func FlattenJSON(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			flatMap := FlattenJSON(v)
			for k, v := range flatMap {
				result[fmt.Sprintf("%s.%s", key, k)] = v
			}
		case []interface{}:
			// Handle arrays by including index in the key
			for i, item := range v {
				if subMap, ok := item.(map[string]interface{}); ok {
					flatMap := FlattenJSON(subMap)
					for k, v := range flatMap {
						result[fmt.Sprintf("%s.%d.%s", key, i, k)] = v
					}
				} else {
					result[fmt.Sprintf("%s.%d", key, i)] = item
				}
			}
		default:
			// Handle primitive values
			result[key] = value
		}
	}
	return result
}

// UnescapeJSONString properly handles escaped JSON strings commonly returned by AWS
func UnescapeJSONString(s string) (string, error) {
	// Check if the string looks like an escaped JSON (contains escaped quotes)
	if strings.Contains(s, "\\\"") {
		// Add outer quotes if not already present
		if !strings.HasPrefix(s, "\"") {
			s = "\"" + s + "\""
		}

		// Use strconv.Unquote to handle the escaping properly
		unescaped, err := strconv.Unquote(s)
		if err != nil {
			// Try an alternative approach for strings that might have been double-escaped
			inner := strings.ReplaceAll(s[1:len(s)-1], `\"`, `"`)
			return inner, nil
		}
		return unescaped, nil
	}
	return s, nil
}

// ConvertAndFlatten converts an interface to JSON, flattens it, and returns a map
// It handles special cases where Properties might be a string, array, or other types
func ConvertAndFlatten(input interface{}) (map[string]interface{}, error) {
	// For nil input, return empty map
	if input == nil {
		return make(map[string]interface{}), nil
	}

	// Handle the case where input is already a map
	if m, ok := input.(map[string]interface{}); ok {
		return FlattenJSON(m), nil
	}

	// Handle case where input is a JSON string
	if s, ok := input.(string); ok {
		if s == "" {
			return make(map[string]interface{}), nil
		}

		// Handle escaped JSON strings (with backslashes before quotes)
		if strings.Contains(s, "\\\"") {
			// Try to unescape the string
			unescaped, err := UnescapeJSONString(s)
			if err == nil {
				s = unescaped
			}
		}

		// Try to unmarshal the string as JSON
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(s), &data); err == nil {
			return FlattenJSON(data), nil
		}

		// If not valid JSON, treat as a simple string property
		return map[string]interface{}{"value": s}, nil
	}

	// Try standard JSON marshalling
	jsonData, err := json.Marshal(input)
	if err != nil {
		// If marshalling fails, try to extract fields directly
		return extractFields(input)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		// If the JSON is not an object, try unmarshaling as an array or primitive
		var anyData interface{}
		if err := json.Unmarshal(jsonData, &anyData); err != nil {
			return nil, err
		}

		// Handle based on type
		switch v := anyData.(type) {
		case []interface{}:
			result := make(map[string]interface{})
			for i, item := range v {
				result[fmt.Sprintf("item_%d", i)] = item
			}
			return result, nil
		default:
			// For primitives or other types
			return map[string]interface{}{"value": v}, nil
		}
	}

	return FlattenJSON(data), nil
}

// extractFields is a fallback method that uses reflection to extract fields from a struct
func extractFields(input interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	v := reflect.ValueOf(input)

	// Handle pointer types
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return result, nil
		}
		v = v.Elem()
	}

	// Only handle struct types
	if v.Kind() != reflect.Struct {
		return result, fmt.Errorf("cannot extract fields from non-struct type: %T", input)
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}

		// Get JSON field name from tag or use struct field name
		fieldName := fieldType.Name
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "-" {
				fieldName = parts[0]
			}
		}

		// Add field value to result
		result[fieldName] = field.Interface()
	}

	return FlattenJSON(result), nil
}
