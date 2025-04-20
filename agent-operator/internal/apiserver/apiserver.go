// Package apiserver defines the HTTP API for the agent-operator
package apiserver

import (
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Algoluna/agent-operator/internal/apiserver/handlers"
)

// SetupAPIServer configures the HTTP API server for the operator
func SetupAPIServer(client client.Client, scheme *runtime.Scheme) (*Server, error) {
	// Create the message handler
	messageHandler := handlers.NewMessageHandler(client, scheme)

	// Set up routes
	mux := http.NewServeMux()

	// Add API routes
	mux.Handle("/api/v1/agents/", messageHandler)

	// Create the HTTP server
	server := &http.Server{
		Addr:    ":8080", // Default port
		Handler: mux,
	}

	return NewServer(server), nil
}
