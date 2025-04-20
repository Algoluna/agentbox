# System Patterns

_This document captures the system architecture, key technical decisions, design patterns, component relationships, and critical implementation paths._

## System Architecture Overview
The platform is composed of:
- A Kubernetes Custom Resource Definition (CRD) for agents
- A Go-based operator that manages agent lifecycle via the CRD
- A CLI tool (agentctl) for agent management
- Sample agent(s) (hello-agent) demonstrating integration
- Helm charts and scripts for deployment
- Future: PostgreSQL for persistent state, Valkey/Redis for messaging and vector memory

## Key Technical Decisions
- Use of Kubernetes CRDs and controllers for extensibility and declarative management
- Go as the primary language for operator and CLI for performance and ecosystem support
- Helm for deployment automation and templating
- Docker for containerization of all components
- Scripts for reproducible local and CI setup

## Design Patterns in Use
- Operator pattern for Kubernetes resource management
- Command pattern in CLI (cobra)
- Declarative configuration via YAML manifests and Helm values
- Reconciliation loop in the operator for state management

## Agent SDK & ADK Integration Pattern (April 2025)

- **SDK Subdirectory:** All agent infrastructure (state, messaging, LLM/embedding stubs, status reporting) is encapsulated in a reusable Python package (`agent_sdk/`).
- **RuntimeContext:** The SDK provides a `RuntimeContext` object, injected into ADK agents at runtime. It manages Postgres state, Valkey messaging, and future LLM/embedding access, without requiring agent subclassing.
- **ADK Compatibility:** Agents are created via configuration (not subclassing) using `google.adk.agents.Agent`, with tool functions registered via the `tools` parameter. The SDK is designed to inject context and infra without altering this pattern.
- **Agent Registration & Entrypoint:** The SDK provides a decorator/registry for agent instances, supporting universal entrypoints and dynamic agent selection. The main loop loads the agent, injects context, receives messages, and persists state.
- **hello-agent Refactor:** The sample agent is refactored to use the SDK for all infra/state/messaging, focusing main.py on agent logic and the main loop.
- **Future-Proofing:** LLM/embedding interfaces are stubs, ready for future backend integration. State and messaging are abstracted for easy extension.

**Rationale:**  
This pattern ensures full compatibility with Google ADK's configuration-based agent pattern, maximizes code reuse, and cleanly separates infrastructure from agent logic. No subclassing is required; agents remain pure ADK objects.

---
## Component Relationships
- The operator watches Agent CRs and manages corresponding pods
- The CLI interacts with the Kubernetes API to launch and monitor agents
- Sample agents are deployed as pods managed by the operator
- Helm charts deploy all components and dependencies in a coordinated fashion

## Critical Implementation Paths
- Agent CR creation triggers the operator to launch a pod and update status
- CLI commands (launch, status) interact with the Kubernetes API and operator
- Deployment scripts and Helm charts set up namespaces, secrets, and all required resources

## Alignment with Product & Tech Context
These patterns and decisions ensure the platform is extensible, cloud-native, and easy to operate, directly supporting the product and technical goals of scalable, declarative agent management.
