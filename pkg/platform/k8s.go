package konstellation

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	graphdb "github.com/praetorian-inc/konstellation/pkg/neo4j"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type K8s struct {
	BasePlatform
}

func NewK8s(configPath embed.FS, cmd *cobra.Command, graphdb graphdb.GraphDB) *K8s {

	k8s := &K8s{
		BasePlatform: BasePlatform{
			Name:    "k8s",
			graphdb: graphdb,
		},
	}

	k8s.BasePlatform.cmd = cmd
	k8s.loadConfig("k8s/config.yml", configPath)

	return k8s
}

func (k *K8s) Enum(ctx context.Context) {
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
			err = os.WriteFile(filePath, resourceJSON, os.ModePerm)
			if err != nil {
				fmt.Printf("Failed to store resource %s: %v\n", resource.Name, err)
				continue
			}

			fmt.Printf("Stored resource %s\n", resource.Name)
		}
	}
}

func (k *K8s) insertNode(ctx context.Context, node *graphdb.Node) (chan any, error) {
	anyChannel := make(chan any, 1)

	apiVersion := strings.Split(node.Props["apiVersion"].(string), "/")[0]
	node.Props["spec.group"] = apiVersion
	if node.Props["metadata.uid"] != nil {
		node.Props["uid"] = node.Props["metadata.uid"]
	}

	node.Props["kind"] = strings.ToLower(node.Labels[0])

	err := k.graphdb.Insert(node, anyChannel)

	if err != nil {
		logrus.Errorf("Failed to insert node: %v", err)
		return nil, err
	}

	return anyChannel, nil
}
