#!/bin/bash
set -e

SCRIPT_DIR=$(dirname "$(readlink -f "$0")")
ROOT_DIR=$(dirname "$SCRIPT_DIR")

echo "=== Building agentctl CLI ==="
cd "$ROOT_DIR/agentctl"
go build -o agentctl .

echo "=== Testing agentctl status command ==="
./agentctl status
echo ""

echo "=== Testing agentctl status for specific agent ==="
./agentctl status hello-agent-test
echo ""

echo "=== Creating a new agent using the CLI ==="
cat << EOF > /tmp/new-agent.yaml
apiVersion: agents.algoluna.com/v1alpha1
kind: Agent
metadata:
  name: hello-agent-cli
spec:
  type: hello-agent
  image: hello-agent:latest
  env:
    - name: LOG_LEVEL
      value: "debug"
    - name: TEST_MODE
      value: "true"
EOF

echo "Launching a new agent with agentctl..."
./agentctl launch /tmp/new-agent.yaml
echo ""

echo "=== Waiting for the new agent to start ==="
sleep 5

echo "=== Checking status of all agents ==="
./agentctl status
echo ""

echo "=== Verifying with kubectl ==="
kubectl get agents
kubectl get pods
echo ""

echo "=== Tailing logs for the new agent with agentctl ==="
./agentctl logs hello-agent-cli --namespace=agent-hello-agent --kubeconfig="$HOME/.kube/config" &
LOG_PID=$!
sleep 5
kill $LOG_PID

echo "=== Sending a message to the agent and waiting for reply ==="
./agentctl message --agent-name=hello-agent-cli --payload='{"test": "ping"}' --redis-url=redis://localhost:6379 --timeout=10

echo "CLI test completed! Both agents should be visible in the cluster."
