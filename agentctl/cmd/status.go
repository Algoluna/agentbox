package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/Algoluna/agentctl/pkg/config"
	"github.com/Algoluna/agentctl/pkg/utils"
)

var statusCmd = &cobra.Command{
	Use:   "status [agent name]",
	Short: "Check the status of an agent",
	Long:  `Check the current status of the specified agent or list all agents if no name is provided.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		// Get Kubernetes configuration
		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")

		// Define AgentGVR
		agentGVR := schema.GroupVersionResource{
			Group:    "agents.algoluna.com",
			Version:  "v1alpha1",
			Resource: "agents",
		}

		// Create Kubernetes client config using our helper
		config, err := getClientConfig(cmd)
		if err != nil {
			return fmt.Errorf("unable to get kubernetes config: %v", err)
		}

		// Create dynamic client
		dynamicClient, err := dynamic.NewForConfig(config)
		if err != nil {
			return fmt.Errorf("error creating client: %v", err)
		}

		// If agent name is specified, get that specific agent
		if len(args) == 1 {
			agentName := args[0]

			// Get agent type to determine namespace
			agentType, err := utils.GetAgentTypeFromName(agentName, kubeconfig)
			if err != nil {
				return fmt.Errorf("error determining agent type: %v", err)
			}

			namespace := utils.GetNamespaceForAgent(agentType)

			agent, err := dynamicClient.Resource(agentGVR).Namespace(namespace).Get(ctx, agentName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("error getting agent %s in namespace %s: %v", agentName, namespace, err)
			}

			// Display agent status
			phase, _, _ := unstructured.NestedString(agent.Object, "status", "phase")
			message, _, _ := unstructured.NestedString(agent.Object, "status", "message")
			agentType, _, _ = unstructured.NestedString(agent.Object, "spec", "type")
			image, _, _ := unstructured.NestedString(agent.Object, "spec", "image")

			fmt.Printf("Agent:    %s\n", agent.GetName())
			fmt.Printf("Type:     %s\n", agentType)
			fmt.Printf("Image:    %s\n", image)
			fmt.Printf("Namespace: %s\n", agent.GetNamespace())
			fmt.Printf("Phase:    %s\n", phase)
			fmt.Printf("Message:  %s\n", message)
			fmt.Printf("Created:  %s\n", agent.GetCreationTimestamp().Time.Format("2006-01-02 15:04:05"))

		} else {
			// List agents from all agent-* namespaces
			fmt.Println("Getting all agents from all agent namespaces...")

			// Get all namespaces
			namespaces, err := dynamicClient.Resource(schema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "namespaces",
			}).List(ctx, metav1.ListOptions{})

			if err != nil {
				return fmt.Errorf("error listing namespaces: %v", err)
			}

			// Filter for namespaces starting with "agent-"
			var agentNamespaces []string
			for _, ns := range namespaces.Items {
				nsName := ns.GetName()
				if len(nsName) > 6 && nsName[:6] == "agent-" {
					agentNamespaces = append(agentNamespaces, nsName)
				}
			}

			// Use a tabwriter to format the output
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAMESPACE\tNAME\tTYPE\tPHASE\tMESSAGE\tCREATED")

			agentCount := 0

			// For each agent namespace, list the agents
			for _, ns := range agentNamespaces {
				agents, err := dynamicClient.Resource(agentGVR).Namespace(ns).List(ctx, metav1.ListOptions{})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error listing agents in namespace %s: %v\n", ns, err)
					continue
				}

				// Display each agent's basic info
				for _, agent := range agents.Items {
					agentCount++
					phase, _, _ := unstructured.NestedString(agent.Object, "status", "phase")
					message, _, _ := unstructured.NestedString(agent.Object, "status", "message")
					agentType, _, _ := unstructured.NestedString(agent.Object, "spec", "type")
					created := agent.GetCreationTimestamp().Time.Format("2006-01-02 15:04:05")

					// Truncate message if it's too long
					if len(message) > 30 {
						message = message[:27] + "..."
					}

					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
						ns,
						agent.GetName(),
						agentType,
						phase,
						message,
						created)
				}
			}

			w.Flush()

			if agentCount == 0 {
				fmt.Printf("No agents found in any namespace\n")
			} else {
				fmt.Printf("\nFound %d agent(s) in %d namespace(s)\n", agentCount, len(agentNamespaces))
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

// getClientConfig returns the Kubernetes client config based on flags and defaults
func getClientConfig(cmd *cobra.Command) (*rest.Config, error) {
	kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
	env, _ := cmd.Flags().GetString("env")

	// Use our config helper
	kubeConfig := config.NewKubeConfig(kubeconfig, env)
	return kubeConfig.GetClientConfig()
}
