package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// KubeConfig helps determine the appropriate kubeconfig to use
type KubeConfig struct {
	Path string
	Env  string
}

// GetClientConfig returns the appropriate Kubernetes client config.
// It prioritizes:
// 1. Explicit kubeconfig path if provided
// 2. If env is "microk8s", use microk8s config
// 3. Default kubeconfig path
// 4. In-cluster config
func (k *KubeConfig) GetClientConfig() (*rest.Config, error) {
	// If explicit path is provided, use it
	if k.Path != "" {
		return clientcmd.BuildConfigFromFlags("", k.Path)
	}

	// If microk8s environment is specified, use microk8s.config
	if k.Env == "microk8s" {
		microk8sConfig, err := GetMicrok8sKubeconfig()
		if err == nil {
			return clientcmd.BuildConfigFromFlags("", microk8sConfig)
		}
		fmt.Fprintf(os.Stderr, "Warning: Could not get microk8s config: %v\n", err)
	}

	// Try default kubeconfig path
	defaultPath := ""
	if home := homedir.HomeDir(); home != "" {
		defaultPath = filepath.Join(home, ".kube", "config")
	}

	if _, err := os.Stat(defaultPath); err == nil {
		return clientcmd.BuildConfigFromFlags("", defaultPath)
	}

	// As a final fallback, try in-cluster config
	return rest.InClusterConfig()
}

// GetMicrok8sKubeconfig returns the path to a temporary file containing the microk8s kubeconfig.
// It first checks if ~/.kube/microk8s.config exists, and if not, it creates it by running
// 'microk8s config'.
func GetMicrok8sKubeconfig() (string, error) {
	home := homedir.HomeDir()
	if home == "" {
		return "", fmt.Errorf("could not determine home directory")
	}

	// Create ~/.kube directory if it doesn't exist
	kubeDir := filepath.Join(home, ".kube")
	if _, err := os.Stat(kubeDir); os.IsNotExist(err) {
		if err := os.MkdirAll(kubeDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %v", kubeDir, err)
		}
	}

	// Path for microk8s config file
	configPath := filepath.Join(kubeDir, "microk8s.config")

	// Check if the file already exists and is not empty
	if info, err := os.Stat(configPath); err == nil && info.Size() > 0 {
		// File exists and not empty, use it
		return configPath, nil
	}

	// Get microk8s config and write to file
	cmd := exec.Command("microk8s", "config")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run 'microk8s config': %v", err)
	}

	// Write the config to the file
	if err := os.WriteFile(configPath, output, 0644); err != nil {
		return "", fmt.Errorf("failed to write microk8s config to %s: %v", configPath, err)
	}

	return configPath, nil
}

// NewKubeConfig creates a new KubeConfig with the given parameters
func NewKubeConfig(kubeconfigPath string, env string) *KubeConfig {
	// If no environment is specified, default to microk8s
	if env == "" {
		env = "microk8s"
	}

	return &KubeConfig{
		Path: kubeconfigPath,
		Env:  env,
	}
}

// IsMicrok8sRunning checks if microk8s is running
func IsMicrok8sRunning() bool {
	cmd := exec.Command("microk8s", "status", "--format", "short")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "microk8s is running")
}
