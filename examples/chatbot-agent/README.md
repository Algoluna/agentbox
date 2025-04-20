# Chatbot Agent Example

This is a simple chatbot agent that:
1. Receives messages from users
2. Processes them using an LLM (Google Gemini Flash by default)
3. Returns the LLM's response
4. Maintains conversation history for context

## Key Features

- **Long-running agent**: The agent stays active indefinitely (runOnce=false)
- **Conversation persistence**: Conversation history is maintained in the agent state
- **LLM integration**: Uses the agent_sdk to communicate with Google's Gemini LLM
- **Configurable model**: Can use different Gemini models via environment variable
- **Environment-specific configuration**: Support for different environments (dev, microk8s, prod)

## Directory Structure

The chatbot-agent follows a simplified and standardized structure:

```
examples/chatbot-agent/
├── Dockerfile       # Container definition for building the agent
├── agent.yaml       # Agent configuration with environment-specific settings
├── main.py          # Agent implementation
└── README.md        # Documentation
```

## Configuration

The agent is configured through the `agent.yaml` file, which follows a standardized format:

```yaml
apiVersion: agents.algoluna.com/v1alpha1
kind: Agent
metadata:
  name: chatbot-agent
spec:
  type: chatbot
  image: "chatbot-agent:latest"
  
  # Base configuration
  env:
    - name: MODEL_NAME
      value: "gemini-pro"
  
  # Environment-specific configurations
  environments:
    dev:
      registry: "localhost:32000"
      cluster: "minikube"
      env:
        - name: DEBUG
          value: "true"
    
    microk8s:
      registry: "localhost:32000"
      cluster: "microk8s"
    
    prod:
      registry: "prod-registry.example.com"
      cluster: "prod-cluster"
      env:
        - name: MODEL_NAME
          value: "gemini-flash-2.0"
```

This configuration allows you to:
- Define common settings in the base spec
- Override settings for specific environments
- Specify different registries and clusters for deployment
- Set environment-specific environment variables

## Usage

### Build and Deploy

Build the agent Docker image:
```bash
agentctl build examples/chatbot-agent --env=dev
```

Deploy the agent to Kubernetes:
```bash
agentctl deploy examples/chatbot-agent --env=dev
```

Both commands use the `--env` flag to select the environment configuration from the agent.yaml file.

### Send Messages

Send a message to the agent:
```bash
agentctl message --agent-name chatbot-agent --payload '{"text": "Hello, how are you today?"}'
```

### Other Commands

Check agent status:
```bash
agentctl status --agent-name chatbot-agent
```

View agent logs:
```bash
agentctl logs --agent-name chatbot-agent
```

## Message Format

**Input**: Expects a JSON payload with a "text" field:
```json
{
  "text": "User's message here"
}
```

**Output**: Returns a JSON payload with:
```json
{
  "text": "LLM's response here",
  "conversation_length": 1 // Number of turns in the conversation
}
```

## Implementation Details

### Conversation Storage

The agent uses the Agent SDK's context storage to maintain conversation history between messages. This allows for multi-turn conversations where the LLM is aware of previous exchanges.

### Environment Variables

- `MODEL_NAME`: The Gemini model to use (default: "gemini-pro")
  - Options include "gemini-pro", "gemini-flash-2.0", etc.
- `DEBUG`: Set to "true" to enable debug logging

## Security & Privacy

- The agent does not store any user data outside of the Agent SDK's state storage
- Conversation history is only maintained for the current user session
- Communication with the Gemini API adheres to Google's privacy and security guidelines
