# Active Context

## CLI, Helm, and Deployment Integration (April 2025)

- **agentctl CLI** now supports: `build`, `deploy`, `logs`, `message`, `status`, `launch`.
- **Helm Chart**: Now includes a parameterized Agent CR template (agent.yaml). All infra (operator, Postgres, Valkey, secrets) is conditionally deployed. Agent CR is created with values from agentctl deploy.
- **Deployment Workflow**: 
  - scripts/deploy.sh uses agentctl for build, deploy, logs, and status.
  - agentctl deploy passes all required values to Helm for agent CR creation and disables infra for agent-only releases.
  - The system release deploys shared infra; agent releases only deploy agent CRs.
- **Helm Issues Resolved**: 
  - Global secrets and infra are not re-created in agent releases.
  - No more ownership/namespace conflicts.
  - No more stuck Helm upgrades.
- **Pod Creation**: 
  - Agent pod (hello-agent) is reliably created and reaches Running/Ready.
  - Pod uses correct image, env, and configuration.
  - Logs confirm successful Postgres/Valkey connections and agent main loop.
- **Multi-Instance Support**: 
  - CLI and chart support deploying multiple agent instances with unique names/images.
- **Testing**: 
  - test/test_cli.sh uses agentctl for logs and messaging.
  - End-to-end deployment and agent lifecycle is fully automated.

## Current Work Focus
- Maintain robust, parameterized deployment and testing via agentctl and Helm.
- Support multi-instance, multi-namespace agent workflows.
- Ensure all new features and bugfixes are reflected in the running agent pod.

## Recent Changes
- Helm chart updated with agent.yaml and conditional infra.
- scripts/deploy.sh and test/test_cli.sh fully integrated with agentctl.
- All deployment, secret, and pod creation issues resolved.

## Next Steps
- Continue to use agentctl and Helm for all agent lifecycle operations.
- Expand CLI/test coverage as new features are added.
- Monitor for edge cases in multi-instance deployments.

## Active Decisions & Considerations
- All infra is managed by the system release; agent releases are agent-only.
- CLI-first, script-driven workflows for all agent operations.
- Parameterized, reproducible deployments.

## Important Patterns & Preferences
- Kubernetes-native, Helm-driven, CLI-first workflows.
- Modular, testable code and scripts.
- Clear separation of system infra and agent instance deployment.

## Project Insights & Learnings
- Parameterized Helm charts and CLI integration are critical for robust, multi-instance agent platforms.
- Early investment in automation and templating pays off in reliability and developer productivity.
