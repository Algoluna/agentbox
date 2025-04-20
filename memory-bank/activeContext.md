# Active Context

## Agent Configuration Standardization (April 2025)

- **Simplified Agent Examples**: Removed chatbot-router, focusing on a single chatbot-agent example for simplicity.
- **Standardized Configuration**: Created a convention-driven structure for all agents with:
  - Dockerfile: standard container definition 
  - agent.yaml: a unified configuration file with environment-specific settings
  - main.py: agent implementation
  - README.md: consistent documentation

- **Environment-specific Configurations**:
  - agent.yaml now includes an `environments` section supporting:
  - Different registries for different environments (dev, microk8s, prod)
  - Environment-specific cluster targeting
  - Environment-variable overrides per environment
  - Base configuration that applies to all environments

- **Enhanced agentctl**:
  - Updated build and deploy commands to support `--env` flag
  - Improved environment handling with environment-specific configs
  - Enhanced kubeconfig management for multi-cluster support

- **Directory Structure**:
  - Eliminated complex config/base and config/local directory structure
  - Consolidated to minimal, convention-based file structure
  - Removed the need for kustomize manipulations

- **Updated Documentation**:
  - Comprehensive README updates for examples
  - Main project README updated with new standardized approach
  - Detailed documentation of the environment-specific configuration

## Current Work Focus
- Maintain consistent, simplified configuration across all agents
- Ensure agentctl properly reads and applies environment-specific configurations
- Support multi-environment, multi-cluster deployments with minimal configuration

## Recent Changes
- Updated Agent CRD to support EnvironmentConfig
- Updated utils.Agent struct to match new CRD and parse config correctly
- Modified agentctl commands for build and deploy to handle environment selection
- Removed chatbot-router example
- Standardized hello-agent and chatbot-agent configurations

## Next Steps
- Continue to improve agentctl for better environment handling
- Add automated testing for environment-specific deployments
- Consider adding support for secret management per environment

## Active Decisions & Considerations
- Convention over configuration: standardized, minimal agent structure
- Environment-specific configuration in agent.yaml, not in separate files
- CLI flags for environment selection in agentctl
- Agent definition in a single, self-contained directory

## Important Patterns & Preferences
- Kubernetes-native, convention-driven agent definitions
- Environment-specific configurations for registries, clusters, env vars
- Clean, minimal directory structure
- Clear separation between agent definition and deployment configuration

## Project Insights & Learnings
- Standardized configuration significantly reduces complexity
- Convention-based approaches improve developer productivity
- Environment-specific configurations provide flexibility without complexity
- Removing unnecessary abstractions (like the router) simplifies the system
