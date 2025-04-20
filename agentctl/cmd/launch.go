package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var launchCmd = &cobra.Command{
	Use:   "launch [agent yaml file]",
	Short: "Launch an agent from YAML file",
	Long:  `Launch an agent by applying the specified YAML file to the Kubernetes cluster.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get file path
		filename := args[0]

		// Read YAML file
		yamlFile, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("error reading file: %v", err)
		}

		// Get Kubernetes configuration
		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
		namespace, _ := cmd.Flags().GetString("namespace")

		if kubeconfig == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}

		// Create Kubernetes client config
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			// Try in-cluster config as fallback
			config, err = rest.InClusterConfig()
			if err != nil {
				return fmt.Errorf("unable to get kubernetes config: %v", err)
			}
		}

		// Create dynamic client
		dynamicClient, err := dynamic.NewForConfig(config)
		if err != nil {
			return fmt.Errorf("error creating client: %v", err)
		}

		// Decode YAML to unstructured object
		decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		obj := &unstructured.Unstructured{}
		_, gvk, err := decUnstructured.Decode(yamlFile, nil, obj)
		if err != nil {
			return fmt.Errorf("error decoding YAML: %v", err)
		}

		// Make sure it's an Agent resource
		if gvk.Group != "agents.algoluna.com" || gvk.Kind != "Agent" {
			return fmt.Errorf("file doesn't contain an agents.algoluna.com/Agent resource")
		}

		// Set namespace if not already set
		if obj.GetNamespace() == "" {
			obj.SetNamespace(namespace)
		}

		// Get resource for Agent
		agentsResource := dynamicClient.Resource(
			gvk.GroupVersion().WithResource("agents"))

		// Create the Agent resource
		result, err := agentsResource.Namespace(obj.GetNamespace()).Create(cmd.Context(), obj, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("error creating agent: %v", err)
		}

		fmt.Printf("Agent '%s' launched in namespace '%s'\n", result.GetName(), result.GetNamespace())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(launchCmd)
}
