#!/bin/bash
set -e

# Apply the agent CRDs
echo "--- Installing Agent CRDs ---"
kubectl apply -f agent-operator/config/crd/bases/agents.algoluna.com_agents.yaml

echo "--- CRDs installed successfully ---"
