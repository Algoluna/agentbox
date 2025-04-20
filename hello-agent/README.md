# Hello Agent

Minimal example agent for the AI Agent Platform using the agent SDK.

## Key Features

- **Long-running agent**: The agent stays active indefinitely (runOnce=false)
- **Simple messaging**: Demonstrates basic messaging capabilities
- **State persistence**: Shows how to use the agent's state
- **Environment-specific configuration**: Support for different environments (dev, microk8s, prod)

## Directory Structure

The hello-agent follows a simplified and standardized structure:

```
hello-agent/
├── Dockerfile       # Container definition for building the agent
├── agent.yaml       # Agent configuration with environment-specific settings
├── main.py          # Agent implementation
└── README.md        # Documentation
```

## Agent Logic

The agent implements minimal logic to demonstrate the platform capabilities:

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

## Configuration

The agent is configured through the `agent.yaml` file, which follows a standardized format:

```yaml
apiVersion: agents.algoluna.com/v1alpha1
kind: Agent
metadata:
  name: hello-agent
spec:
  type: hello
  image: "hello-agent:latest"
  
  # Optional configuration with sensible defaults
  runOnce: false
  maxRestarts: -1
  ttl: 0
  
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
        - name: DEBUG
          value: "false"
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
agentctl build hello-agent --env=dev
```

Deploy the agent to Kubernetes:
```bash
agentctl deploy hello-agent --env=dev
```

Both commands use the `--env` flag to select the environment configuration from the agent.yaml file.

### Send Messages

Send a message to the agent:
```bash
agentctl message --agent-name hello-agent --payload '{"text": "Hello, agent!"}'
```

### Other Commands

Check agent status:
```bash
agentctl status --agent-name hello-agent
```

View agent logs:
```bash
agentctl logs --agent-name hello-agent
```

## Entrypoint

The agent is run using the SDK's standard entrypoint:

```
CMD ["python", "-m", "agent_sdk.runtime.entrypoint"]
```

This entrypoint:
- Reads all secrets and sets up infrastructure
- Instantiates and runs your agent
- Handles the main message loop
- Manages state persistence

## Requirements

- agent_sdk (see ../agent_sdk)
- google-adk
- psycopg2-binary
- redis

## License

MIT License
