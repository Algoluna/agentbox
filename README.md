# AgentBox: Production-Ready AI Agent Platform for Kubernetes

AgentBox is a powerful, extensible platform designed to simplify the deployment, management, and operation of stateful AI agents in Kubernetes environments. Created to bridge the gap between AI development and production operations, AgentBox provides a cohesive infrastructure that allows AI agents to be deployed, monitored, and scaled with enterprise-grade reliability and minimal operational overhead.

## Purpose and Design Goals

The core mission of AgentBox is to transform complex, stateful AI agents from experimental projects into production-ready services by providing:

1. **Infrastructure Abstraction:** Shield AI developers from the complexities of Kubernetes, networking, and cloud infrastructure
2. **Operational Consistency:** Ensure all agents follow consistent deployment, monitoring, and scaling patterns
3. **Stateful Operation:** Enable persistent state management that survives container restarts and redeployments
4. **Secure Agent Communication:** Provide robust, scalable messaging between agents and external systems
5. **Development Acceleration:** Minimize the gap between development and production environments

AgentBox achieves these goals through an architecture that embraces Kubernetes-native patterns while providing higher-level abstractions that make sense for AI agent workloads.

## Components

### Agent Operator

The operator is the brain of the AgentBox platform, written in Go and built on the Kubernetes operator pattern. It:

- Manages the full lifecycle of Agent custom resources in the cluster
- Creates and maintains namespaces for each agent type
- Provisions required secrets and credentials automatically
- Monitors agent health and manages restart policies
- Enforces consistency through Kubernetes-native declarative configuration
- Provides a RESTful API server for direct agent messaging and management

### PostgreSQL Database

PostgreSQL serves as the durable state store for all agents, providing:

- Persistent storage that survives container restarts
- Schema separation by agent type and instance
- Optimized query patterns for agent state retrieval
- Transactional guarantees for state mutations
- Backup and restore capabilities

### Valkey Messaging Infrastructure

Valkey (Redis-compatible) provides the real-time messaging backbone:

- High-performance publish/subscribe channels for agent communication
- Message queuing for asynchronous workloads
- Temporary data caching for performance optimization
- Agent discovery and coordination
- Heartbeat and health monitoring

### Google ADK Integration

AgentBox is fully compatible with Google's Agent Development Kit (ADK), offering:

- Drop-in compatibility with ADK-based agents
- Standardized interfaces for LLM and embedding providers
- Tools and utilities that complement the ADK programming model
- Enhanced deployment capabilities for ADK agents
- Production-grade infrastructure for ADK experimental projects

### agentctl CLI

The agentctl command-line interface is the primary tool for interacting with the AgentBox ecosystem:

- Build agent container images with automatic Docker and registry integration
- Deploy agents to Kubernetes with appropriate configuration and resources
- Monitor agent status and view logs through a unified interface
- Send and receive messages directly to running agents
- Perform end-to-end testing of agent deployments
- Use smart MicroK8s integration for local development

## Agent Configuration

Agents use a standardized configuration approach with `agent.yaml` files that support environment-specific settings:

```yaml
apiVersion: agents.algoluna.com/v1alpha1
kind: Agent
metadata:
  name: agent-name
spec:
  type: agent-type
  image: "agent-image:latest"
  
  # Base configuration (applied to all environments)
  env:
    - name: CONFIG_VAR
      value: "base-value"
  
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

This configuration approach allows:
- Common settings in the base spec
- Environment-specific overrides
- Different registries and clusters for deployment
- Environment-specific environment variables

## Installation

Getting started with AgentBox is straightforward:

1. **Prerequisites Setup:**
   ```bash
   # Install MicroK8s and enable required addons
   sudo snap install microk8s --classic
   microk8s enable registry dns
   
   # Clone repository and set up environment
   git clone https://github.com/yourusername/agentbox.git
   cd agentbox
   
   # Install agentctl to your local bin directory
   ./scripts/install_agentctl.sh
   
   # Set up MicroK8s and other prerequisites
   ./scripts/setup_microk8s.sh
   ./scripts/setup_prereqs.sh
   ```

2. **Deploy Core Infrastructure:**
   ```bash
   # Install CRDs
   ./scripts/install_crds.sh
   
   # Deploy the operator and infrastructure
   ./scripts/deploy.sh
   ```

3. **Deploy Your First Agent:**
   ```bash
   # Build and deploy the hello-agent example
   agentctl build hello-agent --env=dev
   agentctl deploy hello-agent --env=dev
   
   # Check status and logs
   agentctl status hello-agent
   agentctl logs hello-agent
   
   # Test with a message
   agentctl message hello-agent --payload='{"text": "Hello, agent!"}'
   ```

## Examples

The repository contains example agents that demonstrate different capabilities:

- **hello-agent:** A simple example demonstrating basic agent functionality
- **examples/chatbot-agent:** Shows how to build a conversational agent with LLM integration

Each example follows a standardized structure:
```
agent-directory/
├── Dockerfile       # Container definition for building the agent
├── agent.yaml       # Agent configuration with environment-specific settings
├── main.py          # Agent implementation
└── README.md        # Documentation
```

Each example includes its own README.md with detailed explanation and customization options. You can explore these examples to understand how to build your own agents with different capabilities.

For more advanced usage patterns, see the automated test scripts in the `test/` directory and the comprehensive HTML manual at `agentctl/AGENTCTL_MANUAL.html`.

## Author

AgentBox is created and maintained by Shyam Santhanam (santhanamss@gmail.com).

---

AgentBox: Production-ready AI agents on Kubernetes, simplified.
