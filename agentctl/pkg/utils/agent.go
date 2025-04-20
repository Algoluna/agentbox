package utils

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Algoluna/agentctl/pkg/config"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// Agent represents the structure of agent.yaml
type Agent struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace,omitempty"`
	} `yaml:"metadata"`
	Spec struct {
		Type  string `yaml:"type"`
		Image string `yaml:"image"`
		Env   []struct {
			Name  string `yaml:"name"`
			Value string `yaml:"value"`
		} `yaml:"env,omitempty"`
		RunOnce            bool   `yaml:"runOnce,omitempty"`
		MaxRestarts        int    `yaml:"maxRestarts,omitempty"`
		TTL                int64  `yaml:"ttl,omitempty"`
		ServiceAccountName string `yaml:"serviceAccountName,omitempty"`
	} `yaml:"spec"`
	Environments map[string]Environment `yaml:"environments,omitempty"`
}

// Environment represents environment-specific configuration
type Environment struct {
	Registry string `yaml:"registry"`
}

// ReadAgentYAML reads and parses the agent.yaml file
func ReadAgentYAML(directory string) (*Agent, error) {
	if directory == "" {
		directory = "."
	}

	filePath := filepath.Join(directory, "agent.yaml")
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent.yaml: %w", err)
	}

	var agent Agent
	if err := yaml.Unmarshal(data, &agent); err != nil {
		return nil, fmt.Errorf("failed to parse agent.yaml: %w", err)
	}

	return &agent, nil
}

// GetNamespaceForAgent determines the namespace for an agent based on its type
func GetNamespaceForAgent(agentType string) string {
	return fmt.Sprintf("agent-%s", agentType)
}

// GetAgentTypeFromName queries the Kubernetes API to determine the agent type from its name
func GetAgentTypeFromName(agentName string, kubeconfig string) (string, error) {
	ctx := context.Background()

	// Use our new config helper
	k8sConfig := config.NewKubeConfig(kubeconfig, "microk8s")
	restConfig, err := k8sConfig.GetClientConfig()
	if err != nil {
		return "", err
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return "", fmt.Errorf("error creating client: %w", err)
	}

	// Get all namespaces
	namespaces, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	}).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("error listing namespaces: %w", err)
	}

	// Check all namespaces that start with "agent-"
	for _, ns := range namespaces.Items {
		nsName := ns.GetName()
		if !strings.HasPrefix(nsName, "agent-") {
			continue
		}

		// Get agents in this namespace
		agents, err := dynamicClient.Resource(schema.GroupVersionResource{
			Group:    "agents.algoluna.com",
			Version:  "v1alpha1",
			Resource: "agents",
		}).Namespace(nsName).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue // Skip if namespace doesn't have agent resources
		}

		// Find agent with matching name
		for _, agent := range agents.Items {
			if agent.GetName() == agentName {
				// Found it! Extract type from namespace
				agentType := strings.TrimPrefix(nsName, "agent-")
				return agentType, nil
			}
		}
	}

	return "", fmt.Errorf("agent '%s' not found in any namespace", agentName)
}

// ApplyRBACResources applies any RBAC resources found in the rbac/ directory
func ApplyRBACResources(directory string, namespace string, kubeconfig string) error {
	rbacDir := filepath.Join(directory, "rbac")
	if _, err := os.Stat(rbacDir); os.IsNotExist(err) {
		// No RBAC directory, nothing to do
		return nil
	}

	// List YAML files in the rbac directory
	files, err := ioutil.ReadDir(rbacDir)
	if err != nil {
		return fmt.Errorf("failed to read rbac directory: %w", err)
	}

	if len(files) == 0 {
		// No files in RBAC directory
		return nil
	}

	// Build kubectl command to apply all YAML files
	kubectlArgs := []string{"apply"}
	if namespace != "" {
		kubectlArgs = append(kubectlArgs, "-n", namespace)
	}
	if kubeconfig != "" {
		kubectlArgs = append(kubectlArgs, "--kubeconfig", kubeconfig)
	}
	kubectlArgs = append(kubectlArgs, "-f", rbacDir)

	cmd := execCommand("kubectl", kubectlArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GetImageNameForAgent constructs the full image name based on agent config and environment
func GetImageNameForAgent(agent *Agent, envName string) string {
	if envName == "" {
		envName = "microk8s" // Default to microk8s environment
	}

	environment, ok := agent.Environments[envName]
	if !ok {
		// No environment-specific registry, use default image
		return agent.Spec.Image
	}

	// Parse image name and tag
	imageParts := strings.Split(agent.Spec.Image, ":")
	imageName := imageParts[0]
	tag := "latest"
	if len(imageParts) > 1 {
		tag = imageParts[1]
	}

	// Construct full image name with registry
	return fmt.Sprintf("%s/%s:%s", environment.Registry, imageName, tag)
}

// Helper function to get kubeconfig
func getKubeConfig(kubeconfig string) (*rest.Config, error) {
	// Use our new config helper
	k8sConfig := config.NewKubeConfig(kubeconfig, "microk8s")
	return k8sConfig.GetClientConfig()
}

// execCommand is a wrapper around exec.Command that can be overridden in tests
var execCommand = exec.Command
