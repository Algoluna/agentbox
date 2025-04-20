# hello-agent

Minimal example agent for the AI Agent Platform using the agent SDK.

## Agent Logic

The agent only implements its logic and registers itself:

```python
from agent_sdk.runtime.registry import register_agent
from agent_sdk.types import IncomingMessage
import logging
import time

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('hello-agent')

@register_agent("hello_agent")
class HelloAgent:
    def __init__(self, name):
        self.name = name
        self.state = {}

    def on_message(self, message: IncomingMessage):
        logger.info(f"Received message: {message}")
        self.state["last_message"] = message.payload
        for i in range(5):
            logger.info(f"Hello... iteration {i}")
            time.sleep(2)
        self.ctx.messaging().reply(message, {"response": "Hello from agent!"})
```

## Entrypoint

**Do not write a main method.**  
The agent is run using the SDK's standard entrypoint:

```
CMD ["python", "-m", "agent_sdk.runtime.entrypoint"]
```

This entrypoint:
- Reads all secrets and sets up infra
- Instantiates and runs your agent
- Handles the main message loop

## Deployment

- Use the provided Helm chart or Dockerfile to deploy the agent.
- Ensure secrets for Postgres and Valkey are mounted at `/etc/secrets/postgres` and `/etc/secrets/valkey`.

## Requirements

- agent_sdk (see ../agent_sdk)
- google-adk
- psycopg2-binary
- redis

## License

MIT License
