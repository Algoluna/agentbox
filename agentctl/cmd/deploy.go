package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	"github.com/Algoluna/agentctl/pkg/utils"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [directory]",
	Short: "Deploy an agent to the cluster",
	Long:  `Deploy an agent to the cluster by applying RBAC resources and creating the Agent CR.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get directory
		directory := "."
		if len(args) > 0 {
			directory = args[0]
		}

		// Read agent.yaml
		agent, err := utils.ReadAgentYAML(directory)
		if err != nil {
			return err
		}

		// Get kubeconfig
		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")

		// Determine namespace based on agent type
		namespace := utils.GetNamespaceForAgent(agent.Spec.Type)

		// Apply RBAC resources if present
		if err := utils.ApplyRBACResources(directory, namespace, kubeconfig); err != nil {
			return fmt.Errorf("failed to apply RBAC resources: %w", err)
		}

		// Create Agent CR
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to build kubeconfig: %w", err)
		}

		dynamicClient, err := dynamic.NewForConfig(config)
		if err != nil {
			return fmt.Errorf("failed to create dynamic client: %w", err)
		}

		// Create namespace if it doesn't exist
		nsClient := dynamicClient.Resource(schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "namespaces",
		})

		ctx := context.Background()

		// Check if namespace exists
		_, err = nsClient.Get(ctx, namespace, metav1.GetOptions{})
		if err != nil {
			// Create namespace
			nsObj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Namespace",
					"metadata": map[string]interface{}{
						"name": namespace,
					},
				},
			}
			_, err = nsClient.Create(ctx, nsObj, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
			}
			fmt.Fprintf(os.Stderr, "Created namespace %s\n", namespace)
		}

		// Construct full Agent resource YAML
		agentYAML, err := yaml.Marshal(agent)
		if err != nil {
			return fmt.Errorf("failed to marshal agent yaml: %w", err)
		}

		// Apply using kubectl for simplicity
		tmpFile, err := os.CreateTemp("", "agent-*.yaml")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write(agentYAML); err != nil {
			return fmt.Errorf("failed to write to temp file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			return fmt.Errorf("failed to close temp file: %w", err)
		}

		// Apply the Agent CR
		applyArgs := []string{"apply", "-f", tmpFile.Name()}
		if kubeconfig != "" {
			applyArgs = append(applyArgs, "--kubeconfig", kubeconfig)
		}

		applyCmd := exec.Command("kubectl", applyArgs...)
		applyCmd.Stdout = os.Stdout
		applyCmd.Stderr = os.Stderr
		if err := applyCmd.Run(); err != nil {
			return fmt.Errorf("kubectl apply failed: %w", err)
		}

		// Wait for agent pod to be ready
		fmt.Fprintf(os.Stderr, "Waiting for agent pod to be ready...\n")
		waitArgs := []string{
			"wait", "--for=condition=ready", "pod",
			"-l", fmt.Sprintf("agent-name=%s", agent.Metadata.Name),
			"-n", namespace,
			"--timeout=180s",
		}
		if kubeconfig != "" {
			waitArgs = append(waitArgs, "--kubeconfig", kubeconfig)
		}

		waitCmd := exec.Command("kubectl", waitArgs...)
		waitCmd.Stdout = os.Stdout
		waitCmd.Stderr = os.Stderr
		if err := waitCmd.Run(); err != nil {
			return fmt.Errorf("kubectl wait failed: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Agent %s deployed and ready in namespace %s.\n", agent.Metadata.Name, namespace)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
