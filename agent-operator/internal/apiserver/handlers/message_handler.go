package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	agentsv1alpha1 "github.com/Algoluna/agent-operator/api/v1alpha1"
)

var log = logf.Log.WithName("message-handler")

// MessageHandler handles agent messaging API requests
type MessageHandler struct {
	client client.Client
	scheme *runtime.Scheme
	redis  *redis.Client
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(client client.Client, scheme *runtime.Scheme) *MessageHandler {
	// Get Valkey connection info from environment
	valkeyHost := os.Getenv("VALKEY_HOST")
	if valkeyHost == "" {
		valkeyHost = "agentbox-valkey"
	}
	valkeyPort := os.Getenv("VALKEY_PORT")
	if valkeyPort == "" {
		valkeyPort = "6379"
	}
	valkeyUser := os.Getenv("VALKEY_ADMIN_USER")
	if valkeyUser == "" {
		valkeyUser = "default"
	}
	valkeyPassword := os.Getenv("VALKEY_ADMIN_PASSWORD")

	// Connect to Valkey using the operator's admin credentials
	redisURL := fmt.Sprintf("redis://%s:%s@%s:%s", valkeyUser, valkeyPassword, valkeyHost, valkeyPort)
	redisOpt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Error(err, "Failed to parse Valkey URL", "url", redisURL)
		// Continue with nil client, will be handled in ServeHTTP
	}

	rdb := redis.NewClient(redisOpt)

	return &MessageHandler{
		client: client,
		scheme: scheme,
		redis:  rdb,
	}
}

// ServeHTTP handles HTTP requests
func (h *MessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.redis == nil {
		http.Error(w, "Valkey connection not available", http.StatusServiceUnavailable)
		return
	}

	// Extract agent name from URL path
	// Expected format: /api/v1/agents/{agent-name}/messages
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 || pathParts[1] != "api" || pathParts[2] != "v1" || pathParts[3] != "agents" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	agentName := pathParts[4]
	if agentName == "" {
		http.Error(w, "Agent name required", http.StatusBadRequest)
		return
	}

	// Check if this is a messages endpoint
	if len(pathParts) < 6 || pathParts[5] != "messages" {
		http.Error(w, "Invalid endpoint", http.StatusBadRequest)
		return
	}

	// Handle based on HTTP method
	switch r.Method {
	case http.MethodPost:
		h.handleSendMessage(w, r, agentName)
	case http.MethodGet:
		h.handleGetMessages(w, r, agentName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSendMessage sends a message to an agent
func (h *MessageHandler) handleSendMessage(w http.ResponseWriter, r *http.Request, agentName string) {
	ctx := r.Context()

	// Verify agent exists
	var agent agentsv1alpha1.Agent
	if err := h.client.Get(ctx, types.NamespacedName{Name: agentName}, &agent); err != nil {
		http.Error(w, fmt.Sprintf("Agent not found: %v", err), http.StatusNotFound)
		return
	}

	// Parse request body
	var messageReq struct {
		Payload json.RawMessage `json:"payload"`
		Timeout int             `json:"timeout,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&messageReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Set default timeout
	timeout := 30
	if messageReq.Timeout > 0 {
		timeout = messageReq.Timeout
	}

	// Compose stream and reply keys
	streamKey := fmt.Sprintf("agent:%s:inbox", agentName)
	replyKey := fmt.Sprintf("agent:%s:reply", agentName)

	// Send message to agent's inbox stream
	msgID, err := h.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"payload": string(messageReq.Payload),
			"sender":  r.Header.Get("X-User-ID"), // Optional: capture sender ID if provided
		},
	}).Result()

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to send message: %v", err), http.StatusInternalServerError)
		return
	}

	log.Info("Message sent to agent", "agent", agentName, "messageID", msgID)

	// Wait for reply on reply stream
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)

	for {
		now := time.Now()
		if now.After(deadline) {
			http.Error(w, "Timeout waiting for reply", http.StatusGatewayTimeout)
			return
		}

		// Read latest message from reply stream
		res, err := h.redis.XRead(ctx, &redis.XReadArgs{
			Streams: []string{replyKey, "0"},
			Count:   1,
			Block:   1000, // 1s
		}).Result()

		if err != nil && err != redis.Nil {
			http.Error(w, fmt.Sprintf("Error reading reply: %v", err), http.StatusInternalServerError)
			return
		}

		if len(res) > 0 && len(res[0].Messages) > 0 {
			// Return the reply
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"reply": res[0].Messages[0].Values,
				"id":    res[0].Messages[0].ID,
			})
			return
		}

		// Sleep briefly before next poll
		time.Sleep(500 * time.Millisecond)
	}
}

// handleGetMessages gets messages for an agent
func (h *MessageHandler) handleGetMessages(w http.ResponseWriter, r *http.Request, agentName string) {
	ctx := r.Context()

	// Verify agent exists
	var agent agentsv1alpha1.Agent
	if err := h.client.Get(ctx, types.NamespacedName{Name: agentName}, &agent); err != nil {
		http.Error(w, fmt.Sprintf("Agent not found: %v", err), http.StatusNotFound)
		return
	}

	// Get parameters
	limit := 10 // Default limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
		if limit <= 0 {
			limit = 10
		}
	}

	// Compose reply key - for now, we only read replies
	replyKey := fmt.Sprintf("agent:%s:reply", agentName)

	// Read messages from reply stream
	res, err := h.redis.XRevRange(ctx, replyKey, "+", "-").Result()

	// Limit the results if needed
	if len(res) > limit {
		res = res[:limit]
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Error reading messages: %v", err), http.StatusInternalServerError)
		return
	}

	// Format as JSON response
	messages := make([]map[string]interface{}, 0, len(res))
	for _, msg := range res {
		messages = append(messages, map[string]interface{}{
			"id":     msg.ID,
			"values": msg.Values,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
	})
}
