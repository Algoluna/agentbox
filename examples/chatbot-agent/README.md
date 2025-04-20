# Chatbot Agent Example

This is a per-user chatbot agent that maintains conversation history and uses Google Gemini Flash 2.0 (via Google ADK) to generate responses.

## Environment Variables

- `MODEL_NAME`: (optional) The model to use (default: `models/gemini-flash-2.0`)
- `GEMINI_API_KEY`: **(required)** Your Gemini API key for LLM access

## Example Agent CR

```yaml
apiVersion: agents.algoluna.com/v1alpha1
kind: Agent
metadata:
  name: chatbot-agent
spec:
  type: chatbot-agent
  image: chatbot-agent:latest
  runOnce: false
  env:
    - name: AGENT_TYPE
      value: chatbot-agent
    - name: GEMINI_API_KEY
      valueFrom:
        secretKeyRef:
          name: gemini-api-key
          key: api-key
```

## Providing the API Key

For security, store your Gemini API key in a Kubernetes Secret:

```sh
kubectl create secret generic gemini-api-key --from-literal=api-key=YOUR_GEMINI_API_KEY
```

The agent will read the key from the environment and pass it to the Google ADK LLM client.

## Notes

- The ModelManager expects the API key in the `GEMINI_API_KEY` environment variable.
- Make sure the secret is created in the same namespace as the agent.
- The router agent can also be configured with the API key if it needs to use the LLM.
