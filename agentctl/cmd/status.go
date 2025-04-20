package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"path/filepath"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var statusCmd = &cobra.Command{
	Use:   "status [agent name]",
	Short: "Check the status of an agent",
	Long:  `Check the current status of the specified agent or list all agents if no name is provided.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Define AgentGVR
		agentGVR := schema.GroupVersionResource{
			Group:    "agents.algoluna.com",
			Version:  "v1alpha1",
			Resource: "agents",
		}

		// If agent name is specified, get that specific agent
		if len(args) == 1 {
			agentName := args[0]
			agent, err := dynamicClient.Resource(agentGVR).Namespace(namespace).Get(cmd.Context(), agentName, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("error getting agent %s: %v", agentName, err)
			}

			// Display agent status
			phase, _, _ := unstructured.NestedString(agent.Object, "status", "phase")
			message, _, _ := unstructured.NestedString(agent.Object, "status", "message")
			agentType, _, _ := unstructured.NestedString(agent.Object, "spec", "type")
			image, _, _ := unstructured.NestedString(agent.Object, "spec", "image")

			fmt.Printf("Agent:    %s\n", agent.GetName())
			fmt.Printf("Type:     %s\n", agentType)
			fmt.Printf("Image:    %s\n", image)
			fmt.Printf("Namespace: %s\n", agent.GetNamespace())
			fmt.Printf("Phase:    %s\n", phase)
			fmt.Printf("Message:  %s\n", message)
			fmt.Printf("Created:  %s\n", agent.GetCreationTimestamp().Time.Format("2006-01-02 15:04:05"))

		} else {
			// List all agents
			agents, err := dynamicClient.Resource(agentGVR).Namespace(namespace).List(cmd.Context(), metav1.ListOptions{})
			if err != nil {
				return fmt.Errorf("error listing agents: %v", err)
			}

			if len(agents.Items) == 0 {
				fmt.Printf("No agents found in namespace %s\n", namespace)
				return nil
			}

			// Use a tabwriter to format the output
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tTYPE\tPHASE\tMESSAGE\tCREATED")

			// Display each agent's basic info
			for _, agent := range agents.Items {
				phase, _, _ := unstructured.NestedString(agent.Object, "status", "phase")
				message, _, _ := unstructured.NestedString(agent.Object, "status", "message")
				agentType, _, _ := unstructured.NestedString(agent.Object, "spec", "type")
				created := agent.GetCreationTimestamp().Time.Format("2006-01-02 15:04:05")

				// Truncate message if it's too long
				if len(message) > 30 {
					message = message[:27] + "..."
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					agent.GetName(),
					agentType,
					phase,
					message,
					created)
			}
			w.Flush()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
