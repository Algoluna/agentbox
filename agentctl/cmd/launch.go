package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/Algoluna/agentctl/pkg/utils"
)

var launchCmd = &cobra.Command{
	Use:   "launch [directory]",
	Short: "Build, deploy, and launch an agent",
	Long:  `One-command workflow to build, deploy, and launch an agent from the specified directory.`,
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

		// Get kubeconfig and environment
		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
		envName, _ := cmd.Flags().GetString("env")

		// 1. Build the agent image
		fmt.Fprintf(os.Stderr, "=== Building agent image ===\n")
		buildArgs := []string{"build"}
		if len(args) > 0 {
			buildArgs = append(buildArgs, args[0])
		}
		if kubeconfig != "" {
			buildArgs = append(buildArgs, "--kubeconfig", kubeconfig)
		}
		if envName != "" {
			buildArgs = append(buildArgs, "--env", envName)
		}

		buildCmd := exec.Command(os.Args[0], buildArgs...)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("build command failed: %v", err)
		}

		// 2. Deploy the agent
		fmt.Fprintf(os.Stderr, "=== Deploying agent ===\n")
		deployArgs := []string{"deploy"}
		if len(args) > 0 {
			deployArgs = append(deployArgs, args[0])
		}
		if kubeconfig != "" {
			deployArgs = append(deployArgs, "--kubeconfig", kubeconfig)
		}
		if envName != "" {
			deployArgs = append(deployArgs, "--env", envName)
		}

		deployCmd := exec.Command(os.Args[0], deployArgs...)
		deployCmd.Stdout = os.Stdout
		deployCmd.Stderr = os.Stderr
		if err := deployCmd.Run(); err != nil {
			return fmt.Errorf("deploy command failed: %v", err)
		}

		// 3. Show agent logs
		fmt.Fprintf(os.Stderr, "=== Agent logs ===\n")

		// Determine namespace based on agent type
		namespace := utils.GetNamespaceForAgent(agent.Spec.Type)

		// Wait a moment for logs to start flowing
		time.Sleep(2 * time.Second)

		logsArgs := []string{"logs", agent.Metadata.Name, "--follow"}
		if kubeconfig != "" {
			logsArgs = append(logsArgs, "--kubeconfig", kubeconfig)
		}

		logsCmd := exec.Command(os.Args[0], logsArgs...)
		logsCmd.Stdout = os.Stdout
		logsCmd.Stderr = os.Stderr

		// Run logs command but don't wait for it to complete (user can Ctrl+C)
		if err := logsCmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to start logs: %v\n", err)
		}

		fmt.Fprintf(os.Stderr, "\nAgent %s successfully launched in namespace %s!\n", agent.Metadata.Name, namespace)
		fmt.Fprintf(os.Stderr, "Use Ctrl+C to stop watching logs.\n")

		// Wait for logs command to finish (when user presses Ctrl+C)
		logsCmd.Wait()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(launchCmd)
}
