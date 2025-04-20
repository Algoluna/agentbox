package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

var (
	messageAgentName string
	messageNamespace string
	messagePayload   string
	messageRedisURL  string
	messageTimeout   int
)

var messageCmd = &cobra.Command{
	Use:   "message",
	Short: "Send a message to a running agent and receive a reply",
	Long:  `Send a message to the agent's Valkey/Redis stream and print the reply.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if messageAgentName == "" {
			return fmt.Errorf("--agent-name is required")
		}
		if messagePayload == "" {
			return fmt.Errorf("--payload is required")
		}
		if messageRedisURL == "" {
			messageRedisURL = "redis://localhost:6379"
		}
		if messageTimeout == 0 {
			messageTimeout = 30
		}

		ctx := context.Background()
		opt, err := redis.ParseURL(messageRedisURL)
		if err != nil {
			return fmt.Errorf("invalid redis url: %v", err)
		}
		rdb := redis.NewClient(opt)

		// Compose stream and reply keys
		streamKey := fmt.Sprintf("agent:%s:inbox", messageAgentName)
		replyKey := fmt.Sprintf("agent:%s:reply", messageAgentName)

		// Send message to agent's inbox stream
		msgID, err := rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: streamKey,
			Values: map[string]interface{}{
				"payload": messagePayload,
			},
		}).Result()
		if err != nil {
			return fmt.Errorf("failed to send message: %v", err)
		}
		fmt.Fprintf(os.Stderr, "Message sent to %s (ID: %s)\n", streamKey, msgID)

		// Wait for reply on reply stream
		fmt.Fprintf(os.Stderr, "Waiting for reply on %s (timeout: %ds)...\n", replyKey, messageTimeout)
		deadline := time.Now().Add(time.Duration(messageTimeout) * time.Second)
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
	},
}

func init() {
	rootCmd.AddCommand(messageCmd)
	messageCmd.Flags().StringVar(&messageAgentName, "agent-name", "", "Name of the agent (required)")
	messageCmd.Flags().StringVar(&messageNamespace, "namespace", "", "Kubernetes namespace (optional)")
	messageCmd.Flags().StringVar(&messagePayload, "payload", "", "Message payload (required)")
	messageCmd.Flags().StringVar(&messageRedisURL, "redis-url", "", "Redis/Valkey URL (default: redis://localhost:6379)")
	messageCmd.Flags().IntVar(&messageTimeout, "timeout", 30, "Timeout in seconds to wait for reply")
}
