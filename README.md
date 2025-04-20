# AgentBox: Kubernetes-Native AI Agent Platform

AgentBox is a comprehensive, modular framework for deploying, managing, and scaling stateful AI agents on Kubernetes. It combines a robust Go-based operator, extensible CRDs, a Python agent SDK, and a fully featured agentctl CLI to deliver declarative agent management, persistent state, messaging, and seamless multi-instance deployments—all orchestrated via Helm and automated scripts.

## Features

- **Kubernetes-Native Orchestration:** Custom Resource Definitions (CRDs) and a Go operator manage agent lifecycle, secrets, and infra.
- **agentctl CLI:** Build, deploy, log, message, and manage agents with a single tool, deeply integrated with Helm and Kubernetes.
- **Python Agent SDK:** Provides a RuntimeContext for Postgres state, Valkey messaging, and LLM/embedding stubs, fully ADK-compatible.
- **Helm Chart:** Parameterized, modular, and supports conditional deployment of shared infra and agent-only releases.
- **Multi-Instance/Namespace:** Deploy and manage multiple agent types and instances in isolated namespaces.
- **Automated Scripts:** End-to-end deployment, testing, and validation via scripts/deploy.sh and test/test_cli.sh.
- **Comprehensive Documentation:** HTML manual, memory bank, and in-repo docs for onboarding and reference.

## Architecture Overview

- **Operator:** Watches Agent CRs, provisions secrets, manages agent pods, and updates status.
- **CRDs:** Declarative agent definitions, extensible for new agent types and configurations.
- **agentctl CLI:** Unified interface for agent lifecycle operations, tightly coupled with Helm and Kubernetes.
- **Python SDK:** Infrastructure-agnostic agent logic, with pluggable state, messaging, and LLM/embedding.
- **Helm:** Modular chart for system and agent releases, supporting parameterized deployments and conditional infra.
- **Scripts:** Automate build, deploy, and test workflows for reproducibility and CI/CD integration.

## Quickstart

1. **Install Prerequisites:** Go 1.21+, Docker, Helm v3+, kubectl, microk8s or compatible Kubernetes cluster.
2. **Build agentctl:**
   ```sh
   cd agentctl
   go build -o agentctl .
   ```
3. **Deploy system infra:**
   ```sh
   bash scripts/deploy.sh
   ```
   This builds images, deploys infra, and launches a sample agent.

4. **Test agentctl CLI:**
   ```sh
   bash test/test_cli.sh
   ```

## CLI Usage

See [agentctl/AGENTCTL_MANUAL.html](agentctl/AGENTCTL_MANUAL.html) for a full manual.

- Build agent image:
  ```sh
  agentctl build --agent-name=hello-agent --image-tag=debug-20250420-123456 --import-microk8s
  ```
- Deploy agent (agent-only release):
  ```sh
  agentctl deploy --agent-name=hello-agent --namespace=agent-hello-agent --image-tag=debug-20250420-123456 \
    --set globalSecrets.enabled=false \
    --set postgresql.enabled=false \
    --set valkey.enabled=false \
    --set agentOperator.enabled=false \
    --set agent.name=hello-agent \
    --set agent.type=hello-agent \
    --set agent.image=localhost:32000/hello-agent:debug-20250420-123456
  ```
- Tail logs:
  ```sh
  agentctl logs hello-agent --namespace=agent-hello-agent
  ```
- Send message:
  ```sh
  agentctl message --agent-name=hello-agent --payload='{"test": "ping"}' --redis-url=redis://localhost:6379 --timeout=10
  ```
- Status:
  ```sh
  agentctl status
  agentctl status hello-agent
  ```
- Launch from manifest:
  ```sh
  agentctl launch /path/to/agent.yaml
  ```

## Helm & Deployment Patterns

- **System Release:** Deploys shared infra (operator, Postgres, Valkey, secrets) in agentbox-system.
- **Agent Releases:** Deploy agent CRs only, with all infra disabled via Helm flags.
- **Parameterization:** All agent properties (name, type, image, env) are set via Helm values for reproducible, multi-instance deployments.

## Multi-Instance & Multi-Namespace

- Deploy multiple agents by specifying unique `--agent-name`, `--namespace`, and `--image-tag` for each instance.
- Each agent runs in its own namespace with isolated secrets and resources.
- Shared infra is only deployed once in the system namespace.

## Testing & Troubleshooting

- Use `scripts/deploy.sh` and `test/test_cli.sh` for automated validation.
- Check pod and CR status with `agentctl status` and `kubectl get pods -n <namespace>`.
- Review logs with `agentctl logs` for debugging agent startup and infra connections.
- For Helm issues, ensure correct values are passed to disable infra in agent-only releases.

## Contribution & Documentation

- See [memory-bank/activeContext.md](memory-bank/activeContext.md) and [memory-bank/progress.md](memory-bank/progress.md) for project context and progress.
- See [agentctl/AGENTCTL_MANUAL.html](agentctl/AGENTCTL_MANUAL.html) for CLI documentation.
- Contributions are welcome! Please follow best practices for modular, testable, and reproducible code.

## License

[MIT License](LICENSE) (or as specified in the repo)

---
AgentBox: Scalable, reproducible, and production-ready agent operations for Kubernetes-native AI.
