package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	buildAgentName    string
	buildDockerfile   string
	buildContextDir   string
	buildImageTag     string
	buildRegistry     string
	buildImportToMk8s bool
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the agent Docker image and import to microk8s",
	Long:  `Build the agent Docker image, tag it for the local registry, and import it into microk8s/containerd.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if buildAgentName == "" {
			return fmt.Errorf("--agent-name is required")
		}
		if buildDockerfile == "" {
			buildDockerfile = filepath.Join("..", "hello-agent", "Dockerfile")
		}
		if buildContextDir == "" {
			buildContextDir = filepath.Join("..", "hello-agent")
		}
		if buildImageTag == "" {
			buildImageTag = "latest"
		}
		if buildRegistry == "" {
			buildRegistry = "localhost:32000"
		}

		imageName := fmt.Sprintf("%s/%s:%s", buildRegistry, buildAgentName, buildImageTag)

		fmt.Fprintf(os.Stderr, "Building Docker image: %s\n", imageName)
		buildArgs := []string{
			"build", "-t", imageName, "-f", buildDockerfile, buildContextDir,
		}
		buildCmd := exec.Command("docker", buildArgs...)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("docker build failed: %v", err)
		}

		if buildImportToMk8s {
			// Save and import image to microk8s
			tarPath := fmt.Sprintf("%s.tar", buildAgentName)
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
	buildCmd.Flags().StringVar(&buildAgentName, "agent-name", "", "Name of the agent (required)")
	buildCmd.Flags().StringVar(&buildDockerfile, "dockerfile", "", "Path to Dockerfile (default: ../hello-agent/Dockerfile)")
	buildCmd.Flags().StringVar(&buildContextDir, "context", "", "Build context directory (default: ../hello-agent)")
	buildCmd.Flags().StringVar(&buildImageTag, "image-tag", "", "Image tag (default: latest)")
	buildCmd.Flags().StringVar(&buildRegistry, "registry", "", "Registry (default: localhost:32000)")
	buildCmd.Flags().BoolVar(&buildImportToMk8s, "import-microk8s", true, "Import image to microk8s/containerd")
}
