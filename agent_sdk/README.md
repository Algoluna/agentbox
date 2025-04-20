# Agent SDK

Infrastructure SDK for ADK-compatible agents: state, messaging, LLM/embedding, status reporting.

## Overview

This SDK provides a `RuntimeContext` object and supporting infrastructure for running Google ADK-compatible agents in a Kubernetes + Postgres + Valkey (Redis) environment. It enables:

- Persistent agent state (Postgres)
- Redis-based message receipt and delivery (Valkey Streams)
- LLM and Embedding infrastructure access (stubs, backend-agnostic)
- Status reporting to Postgres
- Decorator-based agent registration and universal entrypoint pattern

## ADK Compatibility

- Agents are created via configuration (not subclassing) using `google.adk.agents.Agent`, with tool functions registered via the `tools` parameter.
- The SDK is designed to inject context and infrastructure without altering this pattern.
- No subclassing is required; agents remain pure ADK objects.

## Directory Structure

```
agent_sdk/
  runtime/
    context.py         # RuntimeContext implementation
    messaging.py       # Messaging abstraction (Redis/Valkey)
    llm.py             # LLMManager stub
    embedding.py       # EmbeddingManager stub
    registry.py        # Agent registration/decorator
  db/
    state.py           # State persistence (Postgres)
  types.py             # Message, agent state, etc.
  utils.py             # Shared helpers
  pyproject.toml       # Packaging and dependencies
```

## Usage Example

**Agent author only writes agent logic and registration:**

```python
from agent_sdk.runtime.registry import register_agent
from agent_sdk.types import IncomingMessage

@register_agent("hello_agent")
class HelloAgent:
    def __init__(self, name):
        self.name = name
        self.state = {}

    def on_message(self, message: IncomingMessage):
        # Agent logic here
        self.state["last_message"] = message.payload
        self.ctx.messaging().reply(message, {"response": "Hello!"})
```

**Entrypoint:**  
Set your Dockerfile or Helm chart to use the SDK's standard entrypoint:

```
CMD ["python", "-m", "agent_sdk.runtime.entrypoint"]
```

The SDK will:
- Read secrets and set up all infra
- Instantiate and run your agent
- Enter the main message loop

No main method or boilerplate is needed in your agent code.

## Main Components

- **RuntimeContext:** Injected into agents, provides access to infra (state, messaging, LLM, embedding, status).
- **Messaging:** Handles Redis/Valkey Streams for message receipt and delivery.
- **StateManager:** Loads and saves agent state to Postgres.
- **LLMManager/EmbeddingManager:** Stubs for future LLM/embedding integration.
- **Agent Registry:** Decorator and registry for agent registration and dynamic selection.

## Installation

**Dependencies:**  
- [google-adk](https://pypi.org/project/google-adk/) (Google Agent Development Kit)
- psycopg2-binary
- redis

Install as a local package:

```
pip install -e ./agent_sdk
```

## Development

- Add new infra integrations as needed.
- Extend LLM/embedding stubs for backend support.
- See code comments and docstrings for extension points.

## License

MIT License
