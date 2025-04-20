package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	deployAgentName    string
	deployNamespace    string
	deployImageTag     string
	deployChartPath    string
	deployValuesFile   string
	deployMicrok8sVals string
	deployReleaseName  string
	deploySetValues    []string
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy an agent using Helm",
	Long:  `Build and deploy an agent using Helm, supporting multiple instances and custom image tags.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate required flags
		if deployAgentName == "" {
			return fmt.Errorf("--agent-name is required")
		}
		if deployNamespace == "" {
			return fmt.Errorf("--namespace is required")
		}
		if deployImageTag == "" {
			return fmt.Errorf("--image-tag is required")
		}
		if deployChartPath == "" {
			deployChartPath = filepath.Join("..", "helm")
		}
		if deployValuesFile == "" {
			deployValuesFile = filepath.Join("..", "helm", "values.yaml")
		}
		if deployMicrok8sVals == "" {
			deployMicrok8sVals = filepath.Join("..", "helm", "values-microk8s.yaml")
		}
		if deployReleaseName == "" {
			deployReleaseName = "agent-" + deployAgentName
		}

		// Build the agent image (optional: could call agentctl build here)
		fmt.Fprintf(os.Stderr, "NOTE: Ensure the agent image is built and imported before deploying.\n")

		// Construct Helm command
		helmArgs := []string{
			"upgrade", "--install", deployReleaseName, deployChartPath,
			"--namespace", deployNamespace,
			"-f", deployValuesFile,
			"-f", deployMicrok8sVals,
			"--set", fmt.Sprintf("agentOperator.image.tag=%s", deployImageTag),
			"--set", fmt.Sprintf("agentType=%s", deployAgentName),
			"--set", fmt.Sprintf("agentName=%s", deployAgentName),
			"--create-namespace",
			"--wait",
		}
		for _, setVal := range deploySetValues {
			helmArgs = append(helmArgs, "--set", setVal)
		}

		fmt.Fprintf(os.Stderr, "Running: helm %s\n", strings.Join(helmArgs, " "))

		helmCmd := exec.Command("helm", helmArgs...)
		helmCmd.Stdout = os.Stdout
		helmCmd.Stderr = os.Stderr

		if err := helmCmd.Run(); err != nil {
			return fmt.Errorf("helm deploy failed: %v", err)
		}

		// Wait for agent pod to be ready
		fmt.Fprintf(os.Stderr, "Waiting for agent pod to be ready...\n")
		kubectlArgs := []string{
			"wait", "--for=condition=ready", "pod",
			"-l", fmt.Sprintf("agent-name=%s", deployAgentName),
			"-n", deployNamespace,
			"--timeout=180s",
		}
		kubectlCmd := exec.Command("kubectl", kubectlArgs...)
		kubectlCmd.Stdout = os.Stdout
		kubectlCmd.Stderr = os.Stderr
		if err := kubectlCmd.Run(); err != nil {
			return fmt.Errorf("kubectl wait failed: %v", err)
		}

		fmt.Fprintf(os.Stderr, "Agent %s deployed and ready in namespace %s.\n", deployAgentName, deployNamespace)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().StringVar(&deployAgentName, "agent-name", "", "Name of the agent instance (required)")
	deployCmd.Flags().StringVar(&deployNamespace, "namespace", "", "Kubernetes namespace to deploy to (required)")
	deployCmd.Flags().StringVar(&deployImageTag, "image-tag", "", "Agent Docker image tag (required)")
	deployCmd.Flags().StringVar(&deployChartPath, "chart", "", "Path to Helm chart (default: ../helm)")
	deployCmd.Flags().StringVar(&deployValuesFile, "values", "", "Path to Helm values.yaml (default: ../helm/values.yaml)")
	deployCmd.Flags().StringVar(&deployMicrok8sVals, "microk8s-values", "", "Path to microk8s values file (default: ../helm/values-microk8s.yaml)")
	deployCmd.Flags().StringVar(&deployReleaseName, "release", "", "Helm release name (default: agent-<agent-name>)")
	deployCmd.Flags().StringArrayVar(&deploySetValues, "set", []string{}, "Additional Helm --set values")
}
