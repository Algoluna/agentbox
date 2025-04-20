# Tech Context

_This document details the technologies used, development setup, technical constraints, dependencies, and tool usage patterns._

## Technologies Used
- Go (operator, CLI)
- Python (sample agent)
- Kubernetes (CRDs, controllers, manifests)
- Docker (containerization)
- Helm (deployment)
- PostgreSQL (future: persistent state)
- Valkey/Redis (future: messaging, vector memory)
- Bash (setup and test scripts)

## Development Setup
- Requires Kubernetes cluster (minikube or microk8s recommended for local development)
- Go 1.21+ for building operator and CLI
- Docker for building/pushing images
- Helm v3+ for deployment
- Scripts provided for setup: `scripts/setup_prereqs.sh`, `scripts/deploy.sh`, `test/deploy_to_microk8s.sh`
- Sample agent and manifests in hello-agent/

## Technical Constraints
- Designed for Kubernetes 1.25+ (CRD and controller compatibility)
- Assumes local or accessible Docker registry for image builds
- Helm-based deployment expects certain namespaces and secrets to exist (created by setup scripts)
- Security, multi-tenancy, and production hardening are out of scope for Phase 1
- Python agent SDK (agent_sdk/) must be compatible with Google ADK's configuration-based agent instantiation and tool registration (no subclassing; tools are functions with docstrings)

## Dependencies
- controller-runtime (Go operator SDK)
- cobra (Go CLI)
- Kubernetes API machinery
- Docker images for agents and operator
- Helm charts for deployment
- PostgreSQL and Valkey/Redis (future phases)
- agent_sdk/ (Python package for agent infra: RuntimeContext, messaging, state, LLM/embedding stubs, status reporting)

## Tool Usage Patterns
- Go modules for dependency management
- Makefile for operator build/test
- Bash scripts for setup, deployment, and testing
- Helm for templated, repeatable deployments
- Manual and scripted testing via test/ and scripts/
- No CI/CD pipeline defined yet
- Python agent SDK (agent_sdk/) encapsulates all infra logic for agents, providing a RuntimeContext and registry/decorator for ADK-compatible agent instantiation and main loop orchestration

## Alignment with System Patterns
The tech stack is chosen to maximize Kubernetes-native extensibility, automation, and developer productivity, supporting the system's architecture and design patterns for scalable, declarative agent management.
