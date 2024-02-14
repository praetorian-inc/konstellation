package konstellation

import (
	"encoding/json"
	"log"
	"os"

	"github.com/savaki/jq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// validate a directory path exists
// if no config parameter is passed, attempt the default
func GetDirectoryPath(cmd *cobra.Command, paramName string, defaultPath string) (string, error) {

	dir, _ := cmd.Flags().GetString(paramName)
	if dir == "" {
		dir = defaultPath
	}

	logrus.Infof("Directory path: %v", dir)
	_, err := os.Stat(dir)

	return dir, err
}

func FilterJson(data []byte, jsonPath string, stripQuotes ...bool) ([]byte, error) {
	parser, err := jq.Parse(jsonPath)
	if err != nil {
		log.Fatal("Failed to parse jsonPath: ", err)
		return nil, err
	}
	filtered, err := parser.Apply(data)

	if len(stripQuotes) > 0 && stripQuotes[0] {
		filtered = filtered[1 : len(filtered)-1]
	}

	return filtered, err
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
