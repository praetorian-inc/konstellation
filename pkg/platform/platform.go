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

	"io/ioutil"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	kneo4j "github.com/praetorian-inc/konstellation/pkg/neo4j"
	utils "github.com/praetorian-inc/konstellation/pkg/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Platform struct {
	Name   string
	Config PlatformConfig
	driver neo4j.DriverWithContext
	cmd    *cobra.Command
}

func (p *Platform) Enum(ctx context.Context) {
	fmt.Printf("platform enum\n")

	// Get the home directory
	home := homedir.HomeDir()

	// Set the kubeconfig file path
	kubeconfig := filepath.Join(home, ".kube", "config")

	// Build the client config from the kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Printf("Failed to build client config: %v\n", err)
		return
	}

	// Create a new Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Failed to create clientset: %v\n", err)
		return
	}

	// List all API resources
	apiResources, err := clientset.Discovery().ServerPreferredResources()
	if err != nil {
		fmt.Printf("Failed to list API resources: %v\n", err)
		return
	}

	// Create the k8s-enum folder if it doesn't exist
	err = os.MkdirAll("k8s-enum", os.ModePerm)
	if err != nil {
		fmt.Printf("Failed to create k8s-enum folder: %v\n", err)
		return
	}

	// Retrieve and store each resource as JSON
	for _, resourceList := range apiResources {
		for _, resource := range resourceList.APIResources {
			// Get the resource in JSON format
			resourceJSON, err := clientset.RESTClient().Get().AbsPath(resourceList.GroupVersion + "/" + resource.Name).DoRaw(ctx)
			if err != nil {
				fmt.Printf("Failed to retrieve resource %s: %v\n", resource.Name, err)
				continue
			}

			// Store the resource in the k8s-enum folder
			filePath := filepath.Join("k8s-enum", resource.Name+".json")
			err = ioutil.WriteFile(filePath, resourceJSON, os.ModePerm)
			if err != nil {
				fmt.Printf("Failed to store resource %s: %v\n", resource.Name, err)
				continue
			}

			fmt.Printf("Stored resource %s\n", resource.Name)
		}
	}
}

func (p *Platform) SetDriver(driver neo4j.DriverWithContext) {
	p.driver = driver
}

func (p *Platform) SetCmd(cmd *cobra.Command) {
	p.cmd = cmd
}

func (p *Platform) Push(ctx context.Context) {
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

func (p *Platform) pushNodes(ctx context.Context) error {
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
		var nodeChannel = make(chan neo4j.Node, 1)
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

				node := kneo4j.Item2Node(item, itemLabel)
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
func (p *Platform) getEnumFiles() ([]string, error) {

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

func (p *Platform) getMappingValue(fileName string, key string) (string, error) {

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

	logrus.Trace("file: %v, key: %v, value: %v", fileName, key, fieldVal)

	return fmt.Sprintf("%v", fieldVal), nil
}

func (p *Platform) insertNode(ctx context.Context, node neo4j.Node) (chan any, error) {
	errorChannel := make(chan error, 1)
	anyChannel := make(chan any, 1)

	err := p.driver.VerifyConnectivity(ctx)
	if err != nil {
		logrus.Errorf("Exception: %v", err)
		errorChannel <- err
	}

	session := p.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	if session == nil {
		logrus.Error("Failed to create session")
		errorChannel <- err
	} else {
		logrus.Trace("Created session")
	}

	apiVersion := strings.Split(node.Props["apiVersion"].(string), "/")[0]
	node.Props["spec.group"] = apiVersion
	if node.Props["metadata.uid"] != nil {
		node.Props["uid"] = node.Props["metadata.uid"]
	}

	node.Props["kind"] = strings.ToLower(node.Labels[0])

	val, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		val, err := tx.Run(ctx, "CREATE (n:"+node.Labels[0]+") SET n = $nodeProps", map[string]interface{}{
			"labels":    node.Labels,
			"nodeProps": node.Props,
		})
		if err != nil {
			return nil, err
		}
		return val, nil
	})

	if err != nil {
		logrus.Errorf("Failed to insert node: %v", err)
		return nil, err
	}

	anyChannel <- val

	return anyChannel, nil
}

func (p *Platform) Query(ctx context.Context) {
	logrus.Info(p.Name + " query")

	err := p.driver.VerifyConnectivity(ctx)
	if err != nil {
		logrus.Errorf("Exception: %v", err)
	}
	fmt.Println("passed verify connectivity")
	session := p.driver.NewSession(ctx, neo4j.SessionConfig{})

	resultChannel := make(chan neo4j.ResultWithContext, 10)
	var wg sync.WaitGroup

	for _, query := range p.Config.Queries {
		logrus.Info("Running query: " + query.Name)
		logrus.Debug("Query: " + query.Query)

		//TODO write data to file
		// res, err := session...
		session.ExecuteRead(ctx, func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				query.Query,
				//map[string]any{"message": "hello, world"})
				nil)
			if err != nil {
				return nil, err
			}

			wg.Add(1)
			go p.processQueryResults(query, resultChannel, &wg)

			if result.Next(ctx) {
				resultChannel <- result
				return result.Record().Values[0], nil
			}

			return nil, result.Err()
		})
	}

	if err != nil {
		return
	}

	close(resultChannel)
	defer session.Close(ctx)
}

func (p *Platform) loadConfig(fileName string, configPath embed.FS) {
	logrus.Debugf("Using config path %v", fileName)
	config, _ := GetFileContents(fileName, configPath)

	// unmarshal the yaml to the Config var
	err := yaml.Unmarshal(config, &p.Config)
	if err != nil {
		logrus.Fatalf("Failed to unmarshal %v: %v", configPath, err)
	}

	logrus.Debugf("config: %v", p.Config)
}

func NewPlatform(name string, configPath embed.FS) *Platform {
	p := &Platform{
		Name: name,
	}

	p.loadConfig("k8s/config.yml", configPath)

	return p
}

func (p *Platform) processQueryResults(query Query, resultChannel <-chan neo4j.ResultWithContext, wg *sync.WaitGroup) {
	defer wg.Done()

	for record := range resultChannel {
		node := record.Record().Values[0]
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
