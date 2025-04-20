# Project Brief

_This is the foundation document for the project memory bank. It defines the core requirements, goals, and scope. All other memory bank files build upon this document._

## Project Name
AI Agent Platform

## Overview
A Kubernetes-based platform for long-running, stateful agents with LLM integration, vector memory, persistent state, observability, and CLI + dashboard controls.

## Core Requirements
- Kubernetes-native orchestration of agent workloads
- Agent CRD for declarative agent definitions
- Go-based operator to manage agent lifecycle
- CLI for agent management (launch, status)
- Sample agent implementation
- Helm chart for deployment
- Persistent state and vector memory (future phases)
- Observability and dashboard controls (future phases)

## Goals
- Enable easy deployment and management of stateful AI agents on Kubernetes
- Provide extensible CRDs for agent types and configuration
- Integrate LLMs and vector memory for advanced agent capabilities
- Ensure robust lifecycle management and observability
- Deliver a CLI and dashboard for user interaction

## Scope
In Scope:
- Agent CRD and Go operator
- CLI tool (agentctl)
- Sample agent (hello-agent)
- Helm-based deployment
- Scripts for local and microk8s setup
- Phase 1: Core agent lifecycle, CRD, CLI, sample agent

Out of Scope (Phase 1):
- Full dashboard UI
- Advanced LLM/embedding integrations
- Production-grade security and multi-tenancy

## Stakeholders
- Platform engineers (primary developers)
- AI/ML engineers (agent developers)
- End users (consumers of agent services)
- Project maintainers


## Source of Truth
_This document is the authoritative reference for project scope and direction. All changes to project direction should be reflected here._
