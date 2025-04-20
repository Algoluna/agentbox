package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Algoluna/agentctl/pkg/utils"
)

var buildCmd = &cobra.Command{
	Use:   "build [directory]",
	Short: "Build the agent Docker image",
	Long:  `Build the agent Docker image from the Dockerfile in the specified directory.`,
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

		// Get environment
		envName, _ := cmd.Flags().GetString("env")

		// Determine image name
		imageName := utils.GetImageNameForAgent(agent, envName)

		// Get Dockerfile path and build context
		dockerfilePath := filepath.Join(directory, "Dockerfile")
		if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
			return fmt.Errorf("Dockerfile not found in %s", directory)
		}

		fmt.Fprintf(os.Stderr, "Building Docker image: %s\n", imageName)
		buildArgs := []string{
			"build", "-t", imageName, "-f", dockerfilePath, directory,
		}
		buildCmd := exec.Command("docker", buildArgs...)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("docker build failed: %v", err)
		}

		// Check if we should import to microk8s (only for microk8s environment)
		if envName == "microk8s" {
			// Save and import image to microk8s
			tarPath := fmt.Sprintf("%s.tar", agent.Metadata.Name)
			fmt.Fprintf(os.Stderr, "Saving Docker image to %s\n", tarPath)
			saveCmd := exec.Command("docker", "save", imageName, "-o", tarPath)
			saveCmd.Stdout = os.Stdout
			saveCmd.Stderr = os.Stderr
			if err := saveCmd.Run(); err != nil {
				return fmt.Errorf("docker save failed: %v", err)
			}

			fmt.Fprintf(os.Stderr, "Importing image into microk8s...\n")
			importCmd := exec.Command("microk8s", "ctr", "image", "import", tarPath)
			importCmd.Stdout = os.Stdout
			importCmd.Stderr = os.Stderr
			if err := importCmd.Run(); err != nil {
				return fmt.Errorf("microk8s ctr image import failed: %v", err)
			}

			fmt.Fprintf(os.Stderr, "Cleaning up image tarball...\n")
			os.Remove(tarPath)
		}

		fmt.Fprintf(os.Stderr, "Build complete: %s\n", imageName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
