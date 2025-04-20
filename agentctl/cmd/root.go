package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "agentctl",
	Short: "agentctl is a CLI for managing AI agents",
	Long: `
agentctl is a command line interface for managing AI agents on Kubernetes.
It allows launching agents, checking their status, viewing logs, and more.
`,
}

// Execute adds all child commands to the root command and sets flags
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringP("kubeconfig", "k", "", "path to kubeconfig file (default is $HOME/.kube/config)")
	rootCmd.PersistentFlags().StringP("namespace", "n", "default", "kubernetes namespace")
}
