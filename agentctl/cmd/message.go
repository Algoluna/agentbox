package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Algoluna/agentctl/pkg/utils"
)

var (
	messageAgentName   string
	messagePayload     string
	messageRedisURL    string
	messageTimeout     int
	messageUseAPI      bool
	messageOperatorURL string
)

var messageCmd = &cobra.Command{
	Use:   "message <agent-name>",
	Short: "Send a message to a running agent and receive a reply",
	Long:  `Send a message to the agent's Valkey/Redis stream and print the reply.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Use agent name from args if provided, otherwise use flag
		agentName := messageAgentName
		if len(args) > 0 {
			agentName = args[0]
		}

		if agentName == "" {
			return fmt.Errorf("agent name is required (either as argument or --agent-name flag)")
		}
		if messagePayload == "" {
			return fmt.Errorf("--payload is required")
		}
		if messageTimeout == 0 {
			messageTimeout = 30
		}

		// Get kubeconfig
		kubeconfig, _ := cmd.Flags().GetString("kubeconfig")

		// Get agent type to determine namespace
		agentType, err := utils.GetAgentTypeFromName(agentName, kubeconfig)
		if err != nil {
			return fmt.Errorf("error determining agent type: %v", err)
		}

		namespace := utils.GetNamespaceForAgent(agentType)
		fmt.Fprintf(os.Stderr, "Using namespace: %s for agent %s\n", namespace, agentName)

		// Use the API method if enabled
		if messageUseAPI {
			return sendMessageViaAPI(agentName, messagePayload, messageTimeout, messageOperatorURL)
		}

		// Default direct Redis/Valkey method
		if messageRedisURL == "" {
			messageRedisURL = "redis://localhost:6379"
		}

		return sendMessageViaRedis(agentName, messagePayload, messageRedisURL, messageTimeout)
	},
}

// sendMessageViaRedis sends a message directly to the agent via Redis/Valkey
func sendMessageViaRedis(agentName, payload, redisURL string, timeout int) error {
	ctx := context.Background()
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("invalid redis url: %v", err)
	}
	rdb := redis.NewClient(opt)

	// Compose stream and reply keys
	streamKey := fmt.Sprintf("agent:%s:inbox", agentName)
	replyKey := fmt.Sprintf("agent:%s:reply", agentName)

	// Send message to agent's inbox stream
	msgID, err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"payload": payload,
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Message sent to %s (ID: %s)\n", streamKey, msgID)

	// Wait for reply on reply stream
	fmt.Fprintf(os.Stderr, "Waiting for reply on %s (timeout: %ds)...\n", replyKey, timeout)
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	for {
		now := time.Now()
		if now.After(deadline) {
			return fmt.Errorf("timeout waiting for reply")
		}
		// Read latest message from reply stream
		res, err := rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{replyKey, "0"},
			Count:   1,
			Block:   1000, // 1s
		}).Result()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("error reading reply: %v", err)
		}
		if len(res) > 0 && len(res[0].Messages) > 0 {
			for _, msg := range res[0].Messages {
				fmt.Printf("Reply: %v\n", msg.Values)
				return nil
			}
		}
		// Sleep briefly before next poll
		time.Sleep(500 * time.Millisecond)
	}
}

// sendMessageViaAPI sends a message to an agent using the agent-operator API
func sendMessageViaAPI(agentName, payload string, timeout int, operatorURL string) error {
	// Determine operator URL
	url := operatorURL
	if url == "" {
		// Try to auto-discover from the current Kubernetes context
		discoveredURL, err := getOperatorURLFromKubeconfig()
		if err != nil {
			// Fall back to the default URL with the namespace from the current context
			fmt.Fprintf(os.Stderr, "Failed to get operator URL from kubeconfig: %v\n", err)
			fmt.Fprintf(os.Stderr, "Using default URL: http://agentbox-agent-operator\n")
			url = "http://agentbox-agent-operator"
		} else {
			url = discoveredURL
		}
	}

	// Construct the API endpoint URL
	endpoint := fmt.Sprintf("%s/api/v1/agents/%s/messages", url, agentName)

	// Prepare the request payload
	reqPayload := map[string]interface{}{
		"payload": json.RawMessage(payload),
		"timeout": timeout,
	}
	reqData, err := json.Marshal(reqPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %v", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Set Kubernetes authentication if available
	if err := setKubernetesAuth(req); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to set Kubernetes authentication: %v\n", err)
	}

	// Send the request
	fmt.Fprintf(os.Stderr, "Sending message to agent %s via API (%s)\n", agentName, endpoint)
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed: %s - %s", resp.Status, string(body))
	}

	// Parse the response
	var response struct {
		Reply map[string]interface{} `json:"reply"`
		ID    string                 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	// Print the reply
	fmt.Printf("Reply: %v\n", response.Reply)
	return nil
}

// getOperatorURLFromKubeconfig tries to determine the agent-operator URL from the current Kubernetes context
func getOperatorURLFromKubeconfig() (string, error) {
	// For now, just return a standard in-cluster service URL
	// In the future, this could be enhanced to use the Kubernetes API to look up the service
	return "http://agentbox-agent-operator", nil
}

// setKubernetesAuth adds Kubernetes authentication to the HTTP request
func setKubernetesAuth(req *http.Request) error {
	// Try to get the kubernetes config
	config, err := rest.InClusterConfig()
	if err != nil {
		// Not running in-cluster, try kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = clientcmd.RecommendedHomeFile
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to get Kubernetes config: %v", err)
		}
	}

	// Add the bearer token to the request
	if config.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+config.BearerToken)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(messageCmd)
	messageCmd.Flags().StringVar(&messageAgentName, "agent-name", "", "Name of the agent (optional if provided as argument)")
	messageCmd.Flags().StringVar(&messagePayload, "payload", "", "Message payload (required)")
	messageCmd.Flags().StringVar(&messageRedisURL, "redis-url", "", "Redis/Valkey URL (default: redis://localhost:6379)")
	messageCmd.Flags().IntVar(&messageTimeout, "timeout", 30, "Timeout in seconds to wait for reply")
	messageCmd.Flags().BoolVar(&messageUseAPI, "use-operator-api", true, "Use the operator API instead of direct Valkey connection")
	messageCmd.Flags().StringVar(&messageOperatorURL, "operator-url", "", "Agent operator URL (default: auto-discover from current context)")
}
