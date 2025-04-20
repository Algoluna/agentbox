package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/Algoluna/agentctl/pkg/config"
	"github.com/Algoluna/agentctl/pkg/utils"
)

var logsCmd = &cobra.Command{
	Use:   "logs [agent name]",
	Short: "Tail logs for an agent pod",
	Long:  `Tail the logs for the pod(s) corresponding to the specified agent name.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")

		// Get agent type to determine namespace
		agentType, err := utils.GetAgentTypeFromName(agentName, kubeconfig)
		if err != nil {
			return fmt.Errorf("error determining agent type: %v", err)
		}

		namespace := utils.GetNamespaceForAgent(agentType)

		// Create Kubernetes client config using our helper
		env, _ := cmd.Flags().GetString("env")
		kubeConfig := config.NewKubeConfig(kubeconfig, env)
		restConfig, err := kubeConfig.GetClientConfig()
		if err != nil {
			return fmt.Errorf("unable to get kubernetes config: %v", err)
		}

		clientset, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return fmt.Errorf("error creating clientset: %v", err)
		}

		// Find pod(s) with label agent-name=agentName in the namespace
		labelSelector := fmt.Sprintf("agent-name=%s", agentName)
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return fmt.Errorf("error listing pods: %v", err)
		}
		if len(pods.Items) == 0 {
			return fmt.Errorf("no pods found for agent '%s' in namespace '%s'", agentName, namespace)
		}

		// For now, just tail the first pod found
		pod := pods.Items[0]
		podName := pod.Name

		fmt.Fprintf(os.Stderr, "Tailing logs for pod %s (agent %s) in namespace %s\n", podName, agentName, namespace)

		req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
			Follow:    true,
			TailLines: int64Ptr(50),
		})

		stream, err := req.Stream(context.TODO())
		if err != nil {
			return fmt.Errorf("error streaming logs: %v", err)
		}
		defer stream.Close()

		// Stream logs to stdout
		_, err = io.Copy(os.Stdout, stream)
		if err != nil && err != io.EOF {
			return fmt.Errorf("error copying log stream: %v", err)
		}

		return nil
	},
}

func int64Ptr(i int64) *int64 {
	return &i
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
