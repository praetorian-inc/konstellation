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
	"strings"
	"sync"

	"io/ioutil"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
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

func (p *Platform) pushNodes(ctx context.Context) {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debug("Pushing nodes")
	enumPath, _ := utils.GetDirectoryPath(p.cmd, "enum", "k8s-enum")
	files, err := os.ReadDir(enumPath)
	if err != nil {
		logrus.Errorf("Failed to read directory: %v", err)
		return
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(enumPath, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				logrus.Errorf("Failed to read file: %v", err)
				continue
			}

			template, _ := p.getMappingValue(file.Name(), "template")
			jsonPath, _ := p.getMappingValue(file.Name(), "JsonPath")
			labelField, _ := p.getMappingValue(file.Name(), "LabelField")
			label, _ := p.getMappingValue(file.Name(), "Label")
			nameField, _ := p.getMappingValue(file.Name(), "NameField")

			logrus.Trace("template: ", template)
			logrus.Trace("jsonPath: ", jsonPath)
			logrus.Trace("labelField: ", labelField)
			logrus.Trace("label: ", label)
			logrus.Trace("nameField: ", nameField)
			//p.getMappingValue(file, "labelField")
			/*
				kind, exists := jsonDataMap[p.config.mappings]
				if !exists {
					log.Fatal("Items does not exist in jsonData")
				}
			*/

			// Parse JSON using jsonPath
			filtered, err := utils.FilterJson(data, jsonPath)
			if err != nil {
				logrus.Errorf("Failed to filter JSON: %v", err)
				continue
			}

			//fmt.Println("jsonPath: ", filtered)
			var items interface{}
			err = json.Unmarshal(filtered, &items)
			if err != nil {
				logrus.Errorf("Failed to parse JSON: %v", err)
				continue
			}
			logrus.Infof("Parsed JSON file: %v", file.Name())

			itemsSlice, ok := items.([]interface{})

			if !ok {
				log.Fatal("Could not assert items to slice")
			}

			logrus.Debug("Number of items: ", len(itemsSlice))

			var itemLabel string
			//TODO make the channel size configurable
			var nodeChannel = make(chan neo4j.Node, 1)
			var wg sync.WaitGroup

			// Iterate over items
			for _, item := range itemsSlice {
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

				node := p.item2Node(item, itemLabel)
				nodeChannel <- node
				go p.insertNode(ctx, nodeChannel, wg)
				logrus.Trace(node)

			}
			wg.Wait()
			close(nodeChannel)

		}

	}
}

func (p *Platform) getMappingValue(fileName string, key string) (string, error) {
	mapping := Mapping{}

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
	return fmt.Sprintf("%v", fieldVal), nil
}

func (p *Platform) item2Node(item map[string]interface{}, label string) neo4j.Node {
	flat := utils.FlattenMap(item)
	node := neo4j.Node{Labels: []string{label}, Props: flat}
	logrus.Trace(node)
	return node
}

func (p *Platform) insertNode(ctx context.Context, nodeChannel <-chan neo4j.Node, wg sync.WaitGroup) {
	wg.Add(1)
	if nodeChannel == nil {
		logrus.Error("Node channel is nil")
		return
	}

	err := p.driver.VerifyConnectivity(ctx)
	if err != nil {
		logrus.Errorf("Exception: %v", err)
		return
	}

	session := p.driver.NewSession(ctx, neo4j.SessionConfig{})
	if session == nil {
		logrus.Error("Failed to create session")
		return
	} else {
		logrus.Trace("Created session")
	}

	for node := range nodeChannel {
		//apoc.text.split(items.apiVersion, "/")[0] as specGroup

		apiVersion := strings.Split(node.Props["apiVersion"].(string), "/")[0]
		node.Props["spec.group"] = apiVersion
		if node.Props["metadata.uid"] != nil {
			node.Props["uid"] = node.Props["metadata.uid"]
		}

		node.Props["kind"] = strings.ToLower(node.Labels[0])

		// TODO this needs to be converted to use the cypher queries defined in the config
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
			_, err := tx.Run(ctx, "CREATE (n:"+node.Labels[0]+") SET n = $nodeProps", map[string]interface{}{
				"labels":    node.Labels,
				"nodeProps": node.Props,
			})
			return nil, err
		})

		if err != nil {
			logrus.Errorf("Failed to insert node: %v", err)
		}
	}

	wg.Done()
	defer session.Close(ctx)
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
