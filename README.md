# AI Agent Platform

A Kubernetes-based platform for long-running, stateful agents with LLM integration, vector memory, persistent state, observability, and CLI + dashboard controls.

## Project Overview

The AI Agent Platform integrates long-running, stateful agents with Kubernetes orchestration, providing capabilities such as:
- LLM integration
- Vector memory
- Persistent state
- Observability
- CLI + dashboard controls

## Repository Structure

- **agent-operator/**: Go-based Kubernetes operator (CRDs, controller, manifests)
- **agentctl/**: CLI tool for agent management
- **hello-agent/**: Sample agent implementation
- **test/**: Testing and deployment scripts
- **plan/**: Planning and roadmap documents

## Phase 1 Implementation

Phase 1 implements the core CRD and operator:

- **Agent CRD**: Defines agent types and instances declaratively
- **Go Operator**: Manages CR lifecycle and orchestrates pods
- **Hello World Agent**: Simple example agent
- **CLI**: Basic commands: launch and status

### Agent CRD

The `Agent` custom resource allows you to define:
- Agent type
- Container image
- Environment variables
- Input/output schema references (future)

Status fields track:
- Phase (Pending, Running, Completed, Failed)
- Status message

### Operator Functionality

The operator:
- Creates a pod for each Agent CR
- Updates the Agent status based on pod lifecycle
- Sets proper ownership references

### CLI (agentctl)

The CLI provides:
- `launch`: Deploy an agent from a YAML file
- `status`: Check agent status or list all agents

## Getting Started

### Prerequisites

- Kubernetes cluster (minikube or microk8s for local development)
- Go 1.21+
- Docker
- Helm v3+

### Deploying to MicroK8s

Run the deployment script:

```bash
./test/deploy_to_microk8s.sh
```

This will:
1. Set up necessary MicroK8s addons (DNS, registry, storage)
2. Build the hello-agent image
3. Install the Agent CRD
4. Deploy the operator
5. Create a sample agent instance

### Deploying with Helm

We provide a Helm chart for deploying the full platform (PostgreSQL, Valkey, Agent Operator).

First, set up the prerequisites:

```bash
# Create namespaces and PostgreSQL admin credentials secret
./scripts/setup_prereqs.sh
```

Then deploy using Helm:

```bash
# Install or upgrade the deployment
helm upgrade --install agentbox ./helm \
  --namespace=agentbox-system \
  -f helm/values.yaml \
  -f helm/values-microk8s.yaml
```

The `scripts/setup_prereqs.sh` script does the following:
1. Creates the `agentbox-system` namespace for core components
2. Creates the `agent-hello-agent` namespace for agent instances
3. Creates a Secret containing PostgreSQL admin credentials

Alternatively, use the all-in-one deploy script:

```bash
./scripts/deploy.sh
```

### Testing the CLI

After deploying, test the CLI:

```bash
./test/test_cli.sh
```

This will:
1. Build the CLI
2. Test status commands
3. Launch a new agent via the CLI
4. Verify everything is working

## Manually Testing Components

### Create an Agent

Sample agent manifests are located in their respective directories:
- Agent Operator samples: `agent-operator/config/samples/`
- Hello Agent samples: `hello-agent/config/samples/`

Example of an agent YAML:

```yaml
apiVersion: agents.algoluna.com/v1alpha1
kind: Agent
metadata:
  name: hello-001
spec:
  type: hello-agent
  image: hello-agent:latest
  env:
    - name: LOG_LEVEL
      value: "info"
```

### CLI Commands

```bash
# Check agent status
./agentctl status

# Check specific agent
./agentctl status hello-001

# Launch a new agent
./agentctl launch my-agent.yaml
```

## Next Steps

Phase 2 will implement the Agent SDK with:
- Persistent state via Postgres
- Messaging via Redis Streams
- Stateless and stateful LLM interactions
- Embedding API for semantic memory

## License

Apache 2.0
