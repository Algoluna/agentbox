# Chatbot Router Agent Example

This example demonstrates a scalable, Kubernetes-native architecture for per-user ephemeral chatbot agents, using a router agent to provision, route, and manage user-specific chatbot instances with TTL-based cleanup.

## Architecture

- **ChatbotRouter**: A long-running agent that receives all user messages, provisions per-user chatbot agents (with TTL), and routes messages to the correct instance.
- **Per-user Chatbot Agents**: Each user gets a dedicated agent pod, with isolated conversation state and LLM integration (Gemini Flash 2.0 by default).
- **Ephemeral Agents**: Each chatbot agent is provisioned with a TTL (e.g., 30 minutes). If inactive, it is automatically deleted by the control plane.

## Key Features

- **Dynamic per-user agent provisioning**
- **Ephemeral agent lifecycle with TTL**
- **Persistent conversation state**
- **LLM integration via Google ADK**
- **Kubernetes-native, declarative, and scalable**

## Directory Structure

```
examples/
  chatbot-router/
    main.py
    config/
      rbac/
        serviceaccount.yaml
        role.yaml
        rolebinding.yaml
  chatbot-agent/
    main.py
```

## RBAC

The router agent requires permissions to create, delete, and manage Agent CRs. Apply the RBAC resources:

```sh
kubectl apply -f examples/chatbot-router/config/rbac/
```

## Deploying the Router Agent

1. Build and push the chatbot-router image.
2. Create an Agent CR for the router, specifying the `chatbot-router` ServiceAccount.
3. The router will automatically provision per-user chatbot agents as messages arrive.

## Example Agent CR (for router)

```yaml
apiVersion: agents.algoluna.com/v1alpha1
kind: Agent
metadata:
  name: chatbot-router
spec:
  type: chatbot-router
  image: chatbot-router:latest
  runOnce: false
  serviceAccountName: chatbot-router
  env:
    - name: AGENT_TYPE
      value: chatbot-router
    - name: NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
```

## How It Works

- Users send messages to the router agent.
- The router provisions a per-user chatbot agent (if not already running) with a TTL.
- The router routes the message to the user's agent.
- The per-user agent maintains conversation history and responds using the LLM.
- If a chatbot agent is inactive for longer than its TTL, it is automatically deleted.

## Requirements

- Kubernetes cluster with agent-operator and CRDs installed
- Redis/Valkey and Postgres deployed (see system Helm chart)
- Google ADK Python packages installed in agent images

## Notes

- The router and per-user agents use the agent_sdk for messaging, state, and model integration.
- The TTL and ephemeral agent lifecycle are enforced by the operator/controller.
- This pattern can be extended to other multi-tenant, ephemeral agent scenarios.
