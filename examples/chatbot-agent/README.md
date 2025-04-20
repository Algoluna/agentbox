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

## Usage

### Run locally:

1. Build and deploy the agent:
```bash
agentctl build --path examples/chatbot-agent --tag chatbot-agent:latest
agentctl deploy --manifest examples/chatbot-agent/chatbot-agent.yaml
```

2. Send a message to the agent:
```bash
agentctl message --agent-name chatbot-agent --payload '{"text": "Hello, how are you today?"}'
```

The agent will process the message using the configured LLM and return a response.

### Configuration

The agent's behavior can be customized through environment variables in the deployment manifest:

- `MODEL_NAME`: The Gemini model to use (default: "gemini-pro")
  - Options include "gemini-pro", "gemini-flash", etc.
- `PYTHONUNBUFFERED`: Set to "1" to ensure logs are output immediately

## Implementation Details

### Message Format

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

### Conversation Storage

The agent uses the Agent SDK's context storage to maintain conversation history between messages. This allows for multi-turn conversations where the LLM is aware of previous exchanges.

## Security & Privacy

- The agent does not store any user data outside of the Agent SDK's state storage
- Conversation history is only maintained for the current user session
- Communication with the Gemini API adheres to Google's privacy and security guidelines

## Communication

The agent leverages the Agentbox messaging system through Redis/Valkey streams. Messages can be sent through:

1. Direct Redis/Valkey connection (advanced usage)
2. The agentctl CLI tool (recommended)
3. The agent-operator API proxy (for production/Kubernetes environments)
