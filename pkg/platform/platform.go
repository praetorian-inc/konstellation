package konstellation

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	graphdb "github.com/praetorian-inc/konstellation/pkg/neo4j"
	utils "github.com/praetorian-inc/konstellation/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Platform interface {
	Enum()
	Push()
	Query()
}

type BasePlatform struct {
	Name    string
	Config  PlatformConfig
	graphdb graphdb.GraphDB
	cmd     *cobra.Command
}

func (p *BasePlatform) SetCmd(cmd *cobra.Command) {
	p.cmd = cmd
}

func (p *BasePlatform) Push(ctx context.Context) {
	logrus.Info(p.Name + " push")
	//enumOutput, _ := cmd.Flags().GetString("enum")
	relationships, _ := p.cmd.Flags().GetBool("relationships")
	//relationshipName, _ := cmd.Flags().GetString("relationship-name")

	if !relationships {
		fmt.Printf("default: %v", p.Config.Mappings["default"])
		/*if !p.Config.Mappings["default"] {
			configPath := filepath.Join(path, p.Name, "config.yml")
			logrus.Panicf("%v is missing a \"default\" template mapping entry")
			os.Exit(1)
		}*/
		p.pushNodes(ctx)
	}
}

func (p *BasePlatform) pushNodes(ctx context.Context) error {
	runtime.GOMAXPROCS(4)
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debug("Pushing nodes")

	files, err := p.getEnumFiles()
	if err != nil {
		return err
	}

	for _, filePath := range files {

		data, err := os.ReadFile(filePath)
		if err != nil {
			logrus.Errorf("Failed to read file: %v", err)
			continue
		}

		file := filepath.Base(filePath)
		// TODO implement template usage
		//template, _ := p.getMappingValue(file.Name(), "template")
		jsonPath, _ := p.getMappingValue(file, "JsonPath")
		labelField, _ := p.getMappingValue(file, "LabelField")
		label, _ := p.getMappingValue(file, "Label")
		nameField, _ := p.getMappingValue(file, "NameField")

		itemsSlice, err := getItems(data, jsonPath)
		if err != nil {
			logrus.Errorf("Failed to get items: %v", err)
			continue
		}
		logrus.Infof("Parsed JSON file: %v", file)

		var itemLabel string
		var nodeChannel = make(chan graphdb.Node, 1)
		//var errorChannel = make(error, 1)
		var resultsChannel = make(chan any, 1)
		wg := sync.WaitGroup{}

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for node := range nodeChannel {
					rc, err := p.insertNode(ctx, node)
					if err != nil {
						logrus.Errorf("Failed to insert node: %v", err)
						continue
					}
					resultsChannel <- rc
				}
			}()
		}

		// Iterate over items
		go func() {
			for _, item := range itemsSlice {
				//TODO make the channel size configurable

				//TODO we've got j and item in this loop, which is confusing
				j, err := json.Marshal(item)
				// convert item to map/number of items

				item := item.(map[string]interface{})

				if err != nil {
					logrus.Errorf("Failed to marshal item: %v", err)
					continue
				}

				// get the label
				if label == "" {
					l, err := utils.FilterJson(j, labelField, true)
					if err != nil {
						logrus.Errorf("Failed to get labelField: %s", err)
						continue
					}

					itemLabel = string(l)
				} else {
					itemLabel = label
				}
				logrus.Tracef("Item label: %s", itemLabel)

				name, err := utils.FilterJson(j, nameField, true)
				if err != nil {
					logrus.Errorf("Failed to get nameField: %s", err)
					continue
				}
				logrus.Tracef("Item name: %s", name)
				item["name"] = string(name)

				node := graphdb.Item2Node(item, itemLabel)
				logrus.Trace(node)

				nodeChannel <- node

			}

			close(nodeChannel)
			wg.Wait()
			close(resultsChannel)

		}()

		// consume the results
		for range resultsChannel {
		}
	}

	return nil
}

// getFilesEnum returns a list of file paths for JSON files in the "enum/k8s-enum" directory.
// It uses the provided Platform's cmd field to get the directory path.
// If there is an error while reading the directory or finding JSON files, it returns an error.
func (p *BasePlatform) getEnumFiles() ([]string, error) {

	enumPath, _ := utils.GetDirectoryPath(p.cmd, "enum", "k8s-enum")
	files, err := os.ReadDir(enumPath)
	if err != nil {
		return nil, err
	}

	var fileNames []string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(enumPath, file.Name())
			fileNames = append(fileNames, filePath)
		}
	}

	return fileNames, nil
}

// getItems parses the given JSON data using the provided jsonPath and returns a slice of items.
// It filters the JSON data based on the jsonPath, unmarshals the filtered data into an interface{},
// and asserts it to a slice of interfaces. It returns the items slice and any error encountered during the process.
func getItems(data []byte, jsonPath string) (itemsSlice []interface{}, err error) {

	// Parse JSON using jsonPath
	filtered, err := utils.FilterJson(data, jsonPath)
	if err != nil {
		logrus.Errorf("Failed to filter JSON: %v", err)
		return nil, err
	}

	//fmt.Println("jsonPath: ", filtered)
	var items interface{}
	err = json.Unmarshal(filtered, &items)
	if err != nil {
		logrus.Errorf("Failed to parse JSON: %v", err)
		return nil, err
	}

	itemsSlice, ok := items.([]interface{})

	if !ok {
		log.Fatal("Could not assert items to slice")
	}

	logrus.Debug("Number of items: ", len(itemsSlice))

	return itemsSlice, nil
}

func (p *BasePlatform) getMappingValue(fileName string, key string) (string, error) {

	mapping, ok := p.Config.Mappings[fileName]
	if !ok {
		mapping, ok = p.Config.Mappings["default"]
		if !ok {
			logrus.Panic("No default mapping found")
			return "", fmt.Errorf("No default mapping found")
		}
	}

	val := reflect.ValueOf(mapping)
	fieldVal := val.FieldByName(key)
	if !fieldVal.IsValid() {
		return "", fmt.Errorf("Field %v does not exist", key)
	}

	logrus.Tracef("file: %v, key: %v, value: %v", fileName, key, fieldVal)

	return fmt.Sprintf("%v", fieldVal), nil
}

func (p *BasePlatform) insertNode(ctx context.Context, node graphdb.Node) (chan any, error) {
	anyChannel := make(chan any, 1)

	err := p.graphdb.Insert(&node, anyChannel)

	if err != nil {
		logrus.Errorf("Failed to insert node: %v", err)
		return nil, err
	}

	return anyChannel, nil
}

func (p *BasePlatform) applyRelationships() {

	// TODO call generic apply relationships

}

func (p *BasePlatform) Query(ctx context.Context) {
	logrus.Info(p.Name + " query")

	resultChannel := make(chan any, 10)
	var wg sync.WaitGroup

	for _, query := range p.Config.Queries {
		logrus.Info("Running query: " + query.Name)
		logrus.Debug("Query: " + query.Query)

		//TODO write data to file
		// res, err := session...
		err := p.graphdb.Query(query.Query, resultChannel)
		if err != nil {
			logrus.Errorf("Failed to run query: %v", err)
			continue
		}

		wg.Add(1)
		go p.processQueryResults(query, resultChannel, &wg)

		if err != nil {
			return
		}
	}
	close(resultChannel)
	//defer session.Close(ctx)
}

func (p *BasePlatform) loadConfig(fileName string, configPath embed.FS) {
	logrus.Debugf("Using config path %v", fileName)
	config, _ := GetFileContents(fileName, configPath)

	// unmarshal the yaml to the Config var
	err := yaml.Unmarshal(config, &p.Config)
	if err != nil {
		logrus.Fatalf("Failed to unmarshal %v: %v", configPath, err)
	}

	logrus.Debugf("config: %v", p.Config)
}

func NewPlatform(name string, configPath embed.FS) *BasePlatform {
	p := &BasePlatform{
		Name: name,
	}

	return p
}

func (p *BasePlatform) processQueryResults(query Query, resultChannel <-chan any, wg *sync.WaitGroup) {
	defer wg.Done()

	for result := range resultChannel {
		// Cast record to appropriate type
		record := result.(neo4j.Record)

		node := record.Values[0]
		logrus.Trace(node)

		// Convert record to JSON
		jsonData, err := json.Marshal(node)
		if err != nil {
			logrus.Error("Failed to convert record to JSON:", err)
			continue
		}

		// Derive file name from query.Name
		fileName := strings.ReplaceAll(strings.ToLower(query.Name), " ", "_") + ".json"

		// Get results directory from p.cmd
		resultsDir, _ := p.cmd.Flags().GetString("results")
		// Create the directory if it doesn't exist
		err = os.MkdirAll(resultsDir, os.ModePerm)
		if err != nil {
			logrus.Error("Failed to create results directory:", err)
			continue
		}

		// Create the file
		filePath := filepath.Join(resultsDir, fileName)
		file, err := os.Create(filePath)
		if err != nil {
			logrus.Error("Failed to create file:", err)
			continue
		}
		defer file.Close()

		// Write JSON data to the file
		_, err = file.Write(jsonData)
		if err != nil {
			logrus.Error("Failed to write JSON data to file:", err)
			continue
		}
	}
}

func ListFiles(configPath embed.FS) ([]string, error) {
	var files []string

	err := fs.WalkDir(configPath, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, filepath.Join(path))
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func GetFileContents(fileName string, configPath embed.FS) ([]byte, error) {
	file, err := configPath.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return contents, nil
}
